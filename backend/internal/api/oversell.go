package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"clicd/internal/config"
)

// HandleOversell handles GET/POST for oversell config
func HandleOversell(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getOversell(w, r)
	case http.MethodPost:
		updateOversell(w, r)
	default:
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
	}
}

func getOversell(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, APIResponse{Success: true, Data: config.AppConfig.Oversell})
}

func updateOversell(w http.ResponseWriter, r *http.Request) {
	var cfg config.OversellConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request body"})
		return
	}

	// Apply KSM
	if cfg.KSMEnabled {
		exec.Command("sh", "-c", "echo 1 > /sys/kernel/mm/ksm/run 2>/dev/null").Run()
		exec.Command("sh", "-c", "echo 1000 > /sys/kernel/mm/ksm/sleep_millisecs 2>/dev/null").Run()
	} else {
		exec.Command("sh", "-c", "echo 0 > /sys/kernel/mm/ksm/run 2>/dev/null").Run()
	}

	// Apply swappiness
	if cfg.Swappiness >= 0 && cfg.Swappiness <= 100 {
		exec.Command("sh", "-c", fmt.Sprintf("echo %d > /proc/sys/vm/swappiness", cfg.Swappiness)).Run()
	}

	// Oversell multipliers are capacity-planning values. They must not increase
	// an individual container's CPU or RAM limits.
	reapplyContainerLimits()

	config.AppConfig.Oversell = cfg
	if err := config.SaveConfig(); err != nil {
		jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Message: "Failed to save config"})
		return
	}

	jsonResponse(w, http.StatusOK, APIResponse{Success: true, Message: "Oversell config updated", Data: cfg})
}

// reapplyContainerLimits restores cgroup limits for all running containers from
// their assigned container resources.
func reapplyContainerLimits() {
	for _, c := range config.AppConfig.Containers {
		if c.Status != "running" {
			continue
		}
		if err := lxcManager.ApplyContainerLimits(&c); err != nil {
			fmt.Printf("Warning: failed to reapply resource limits for %s: %v\n", c.LxcName(), err)
		}
	}
}

// HandleOversellStatus returns current oversell resource usage
func HandleOversellStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}

	status := map[string]interface{}{
		"ksm_active":        isKSMEnabled(),
		"ksm_pages":         getKSMPages(),
		"ksm_supported":     isKSMSupported(),
		"swappiness":        getSwappiness(),
		"reclaim_supported": isMemoryReclaimSupported(),
		"allocated_cpu":     getAllocatedCPU(),
		"allocated_ram_mb":  getAllocatedRAM(),
		"allocated_disk_gb": getAllocatedDisk(),
	}

	jsonResponse(w, http.StatusOK, APIResponse{Success: true, Data: status})
}

// HandleOversellReclaim triggers one cgroup v2 memory.reclaim pass for running containers.
func HandleOversellReclaim(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}

	result := reclaimContainerMemory()
	jsonResponse(w, http.StatusOK, APIResponse{Success: true, Message: "Memory reclaim triggered", Data: result})
}

func reclaimContainerMemory() map[string]interface{} {
	attempted := 0
	reclaimed := 0
	unsupported := 0
	errors := make([]string, 0)

	for _, c := range config.AppConfig.Containers {
		if c.Status != "running" {
			continue
		}
		attempted++
		reclaimPath := findMemoryReclaimPath(c.LxcName())
		if reclaimPath == "" {
			unsupported++
			continue
		}
		if err := os.WriteFile(reclaimPath, []byte("64M"), 0644); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", c.Name, err))
			continue
		}
		reclaimed++
	}

	return map[string]interface{}{
		"attempted":   attempted,
		"reclaimed":   reclaimed,
		"unsupported": unsupported,
		"errors":      errors,
	}
}

func isKSMEnabled() bool {
	data, err := os.ReadFile("/sys/kernel/mm/ksm/run")
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "1"
}

func isKSMSupported() bool {
	if _, err := os.Stat("/sys/kernel/mm/ksm/run"); err != nil {
		return false
	}
	return true
}

func getKSMPages() int64 {
	data, err := os.ReadFile("/sys/kernel/mm/ksm/pages_shared")
	if err != nil {
		return 0
	}
	val, _ := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	return val
}

func getSwappiness() int {
	data, err := os.ReadFile("/proc/sys/vm/swappiness")
	if err != nil {
		return 60
	}
	val, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return val
}

func isMemoryReclaimSupported() bool {
	if _, err := os.Stat("/sys/fs/cgroup/memory.reclaim"); err == nil {
		return true
	}
	for _, c := range config.AppConfig.Containers {
		if c.Status != "running" {
			continue
		}
		if findMemoryReclaimPath(c.LxcName()) != "" {
			return true
		}
	}
	return false
}

func findMemoryReclaimPath(lxcName string) string {
	candidates := []string{
		fmt.Sprintf("/sys/fs/cgroup/lxc/%s/memory.reclaim", lxcName),
		fmt.Sprintf("/sys/fs/cgroup/lxc.payload.%s/memory.reclaim", lxcName),
		fmt.Sprintf("/sys/fs/cgroup/system.slice/lxc@%s.service/memory.reclaim", lxcName),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func getAllocatedCPU() float64 {
	total := 0.0
	for _, c := range config.AppConfig.Containers {
		total += c.VCPU
	}
	return total
}

func getAllocatedRAM() int64 {
	total := int64(0)
	for _, c := range config.AppConfig.Containers {
		total += int64(c.RAMMB)
	}
	return total
}

func getAllocatedDisk() int64 {
	total := int64(0)
	for _, c := range config.AppConfig.Containers {
		total += int64(c.DiskGB)
	}
	return total
}
