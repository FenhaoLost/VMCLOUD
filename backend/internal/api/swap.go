package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type SwapInfo struct {
	TotalMB   int64  `json:"total_mb"`
	UsedMB    int64  `json:"used_mb"`
	FreeMB    int64  `json:"free_mb"`
	Enabled   bool   `json:"enabled"`
	SwapFile  string `json:"swap_file"`
}

// HandleSwapInfo returns current swap status
func HandleSwapInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}

	info := getSwapInfo()
	jsonResponse(w, http.StatusOK, APIResponse{Success: true, Data: info})
}

// HandleSwapManage creates/enables/disables swap
func HandleSwapManage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}

	var req struct {
		Action string `json:"action"` // create, enable, disable, resize
		SizeMB int    `json:"size_mb"` // for create/resize
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request body"})
		return
	}

	var msg string

	switch req.Action {
	case "create":
		if req.SizeMB <= 0 {
			req.SizeMB = 2048
		}
		err := createSwap(req.SizeMB)
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Message: err.Error()})
			return
		}
		msg = fmt.Sprintf("已创建 %d MB SWAP", req.SizeMB)

	case "enable":
		err := enableSwap()
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Message: err.Error()})
			return
		}
		msg = "SWAP 已启用"

	case "disable":
		err := disableSwap()
		if err != nil {
			jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Message: err.Error()})
			return
		}
		msg = "SWAP 已禁用"

	case "resize":
		if req.SizeMB <= 0 {
			jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid size"})
			return
		}
		disableSwap()
		createSwap(req.SizeMB)
		enableSwap()
		msg = fmt.Sprintf("SWAP 已调整为 %d MB", req.SizeMB)

	default:
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid action: " + req.Action})
		return
	}

	info := getSwapInfo()
	jsonResponse(w, http.StatusOK, APIResponse{Success: true, Message: msg, Data: info})
}

func getSwapInfo() SwapInfo {
	info := SwapInfo{SwapFile: "/swapfile"}

	// Read /proc/meminfo for swap stats
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return info
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.ParseInt(fields[1], 10, 64)
		switch fields[0] {
		case "SwapTotal:":
			info.TotalMB = val / 1024
		case "SwapFree:":
			info.FreeMB = val / 1024
		}
	}

	info.UsedMB = info.TotalMB - info.FreeMB
	if info.TotalMB > 0 {
		info.Enabled = true
	}

	return info
}

func createSwap(sizeMB int) error {
	swapFile := "/swapfile"

	// Check if swap file already exists
	if _, err := os.Stat(swapFile); err == nil {
		// Remove old swap file
		exec.Command("swapoff", swapFile).Run()
		os.Remove(swapFile)
	}

	// Create swap file
	cmd := exec.Command("dd", "if=/dev/zero", "of="+swapFile, "bs=1M", "count="+strconv.Itoa(sizeMB))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("创建 swap 文件失败: %v, %s", err, string(output))
	}

	// Set permissions
	os.Chmod(swapFile, 0600)

	// Make swap
	cmd = exec.Command("mkswap", swapFile)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mkswap 失败: %v, %s", err, string(output))
	}

	// Enable swap
	return enableSwap()
}

func enableSwap() error {
	swapFile := "/swapfile"
	if _, err := os.Stat(swapFile); os.IsNotExist(err) {
		return fmt.Errorf("swap 文件不存在，请先创建")
	}

	cmd := exec.Command("swapon", swapFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if already enabled
		if strings.Contains(string(output), "already") {
			return nil
		}
		return fmt.Errorf("启用 swap 失败: %v, %s", err, string(output))
	}
	return nil
}

func disableSwap() error {
	swapFile := "/swapfile"
	cmd := exec.Command("swapoff", swapFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "No such") {
			return nil
		}
		return fmt.Errorf("禁用 swap 失败: %v, %s", err, string(output))
	}
	return nil
}
