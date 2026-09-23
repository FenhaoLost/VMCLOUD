package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"eyvescloud/internal/config"
)

// 内存超售 / KSM 调优管理。
//
// 超售：可分配内存上限 = 物理内存 × MemoryOvercommitRatio（默认关闭）。
// KSM：写入 /sys/kernel/mm/ksm/{run,pages_to_scan,sleep_millisecs} 合并重复页，
//      降低超售场景下的真实内存占用。仅管理员可配置。

const (
	ksmRunPath        = "/sys/kernel/mm/ksm/run"
	ksmPagesPath      = "/sys/kernel/mm/ksm/pages_to_scan"
	ksmSleepPath      = "/sys/kernel/mm/ksm/sleep_millisecs"
	maxOvercommitRatio = 16.0
)

// HandleOvercommitSettings 读取/更新内存超售与 KSM 调优配置。
func HandleOvercommitSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		jsonResponse(w, http.StatusOK, APIResponse{Success: true, Data: overcommitSettingsResponse()})
	case http.MethodPut:
		if !requireScope(w, r, "admin:access") {
			return
		}
		var req struct {
			MemoryOvercommitEnabled bool     `json:"memory_overcommit_enabled"`
			MemoryOvercommitRatio   float64  `json:"memory_overcommit_ratio"`
			KSMTuning               *struct {
				Enabled        bool `json:"enabled"`
				PagesToScan    int  `json:"pages_to_scan"`
				SleepMillisecs int  `json:"sleep_millisecs"`
				UseTuneKSM     bool `json:"use_tune_ksm"`
			} `json:"ksm_tuning"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request body"})
			return
		}
		if req.MemoryOvercommitRatio < 1.0 || req.MemoryOvercommitRatio > maxOvercommitRatio {
			jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: fmt.Sprintf("memory_overcommit_ratio must be %.2f - %.2f", 1.0, maxOvercommitRatio)})
			return
		}
		var ksm config.KSMTuningConfig
		if req.KSMTuning != nil {
			if req.KSMTuning.PagesToScan < 0 || req.KSMTuning.PagesToScan > 1_000_000 {
				jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: "ksm pages_to_scan must be 0-1000000"})
				return
			}
			if req.KSMTuning.SleepMillisecs < 1 || req.KSMTuning.SleepMillisecs > 60000 {
				jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: "ksm sleep_millisecs must be 1-60000"})
				return
			}
			ksm = config.KSMTuningConfig{
				Enabled:        req.KSMTuning.Enabled,
				PagesToScan:    req.KSMTuning.PagesToScan,
				SleepMillisecs: req.KSMTuning.SleepMillisecs,
				UseTuneKSM:     req.KSMTuning.UseTuneKSM,
			}
		} else {
			ksm = config.GetKSMTuning()
		}
		config.MutateGlobal(func(cfg *config.EyvescloudConfig) {
			cfg.MemoryOvercommitEnabled = req.MemoryOvercommitEnabled
			cfg.MemoryOvercommitRatio = req.MemoryOvercommitRatio
			cfg.KSMTuning = ksm
		})
		if err := config.SaveConfig(); err != nil {
			jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Message: err.Error()})
			return
		}
		// 保存后立即尝试应用 KSM 内核参数（不可写时返回警告而非失败）。
		applyWarnings := applyKSMTuning(ksm)
		auditRequest(r, "overcommit.settings", "resources", fmt.Sprintf(
			"overcommit=%v ratio=%.2f ksm=%v apply_warnings=%v",
			req.MemoryOvercommitEnabled, req.MemoryOvercommitRatio, ksm.Enabled, applyWarnings), true, "")

		jsonResponse(w, http.StatusOK, APIResponse{Success: true, Data: overcommitSettingsResponse()})
	default:
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
	}
}

func overcommitSettingsResponse() map[string]interface{} {
	enabled, ratio := config.GetMemoryOvercommit()
	ksm := config.GetKSMTuning()
	return map[string]interface{}{
		"memory_overcommit_enabled": enabled,
		"memory_overcommit_ratio":   ratio,
		"physical_ram_mb":           hostRAMTotalMB(),
		"allocatable_ram_mb":        alocatableRAMMB(enabled, ratio),
		"ksm_tuning": map[string]interface{}{
			"enabled":         ksm.Enabled,
			"pages_to_scan":   ksm.PagesToScan,
			"sleep_millisecs": ksm.SleepMillisecs,
			"use_tune_ksm":    ksm.UseTuneKSM,
		},
		"notes": "内存超售可分配上限 = 物理内存 × 超售比；KSM 合并重复内存页降低实际占用。",
	}
}

func hostRAMTotalMB() int {
	h := getHostInfo()
	return int(h.RAM.TotalMB)
}

func alocatableRAMMB(enabled bool, ratio float64) int64 {
	total := int64(0)
	h := getHostInfo()
	if h.RAM.TotalMB <= 0 {
		return 0
	}
	total = int64(h.RAM.TotalMB)
	if enabled && ratio > 0 {
		total = int64(float64(h.RAM.TotalMB) * ratio)
	}
	return total
}

// applyKSMTuning 将配置写入内核 KSM 参数。返回无法应用的警告列表，
// 不阻塞调用方（元数据已持久化，重启/下次可重试）。非 root / 非 Linux 时静默降级。
func applyKSMTuning(cfg config.KSMTuningConfig) []string {
	var warnings []string
	if cfg.Enabled {
		// KSM 启用：写 pages_to_scan / sleep_millisecs 后置 run=1。
		writeKSMValue(ksmPagesPath, strconv.Itoa(cfg.PagesToScan), &warnings)
		writeKSMValue(ksmSleepPath, strconv.Itoa(cfg.SleepMillisecs), &warnings)
		writeKSMValue(ksmRunPath, "1", &warnings)
	} else {
		// KSM 关闭：run=0（停止合并）。
		writeKSMValue(ksmRunPath, "0", &warnings)
	}
	return warnings
}

func writeKSMValue(path, value string, warnings *[]string) {
	if err := os.WriteFile(path, []byte(value), 0644); err != nil {
		if os.IsNotExist(err) {
			return // 内核未开启 KSM / 非 Linux，静默忽略
		}
		*warnings = append(*warnings, fmt.Sprintf("%s: %v", path, err))
	}
}