package agent

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"clicd/internal/config"
	"clicd/internal/server"
	"clicd/internal/version"
)

// agentConfig 是被控节点保存的注册信息。
type agentConfig struct {
	Controller string `json:"controller"`
	NodeID     string `json:"node_id"`
	Token      string `json:"token"`
	Name       string `json:"name"`
	Address    string `json:"address"`
}

// Run 启动被控节点 agent 模式：注册到主控、上报心跳、运行本地面板。
// 用法: clicd agent --controller=http://master:18999 [--install-key=xxx] [--name=node1] [--addr=http://1.2.3.4:8999]
func Run(args []string) {
	fs := flag.NewFlagSet("agent", flag.ExitOnError)
	controller := fs.String("controller", "", "主控地址，如 http://1.2.3.4:18999")
	installKey := fs.String("install-key", "", "安装密钥（首次注册时使用）")
	name := fs.String("name", "", "节点名称（默认使用主机名）")
	addr := fs.String("addr", "", "本节点面板地址，如 http://1.2.3.4:8999")
	_ = fs.Parse(args)

	if strings.TrimSpace(*controller) == "" {
		fmt.Fprintln(os.Stderr, "agent 模式需要 --controller 主控地址")
		os.Exit(1)
	}

	if _, err := config.InitConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "初始化配置失败: %v\n", err)
		os.Exit(1)
	}

	cfgPath := filepath.Join(config.AppConfig.DataDir, "agent.json")
	ac := loadAgentConfig(cfgPath)

	needRegister := ac == nil
	if strings.TrimSpace(*installKey) != "" && (ac == nil || ac.Controller != strings.TrimSpace(*controller)) {
		needRegister = true
	}

	if needRegister {
		nc, err := register(strings.TrimSpace(*controller), strings.TrimSpace(*installKey), strings.TrimSpace(*name), strings.TrimSpace(*addr))
		if err != nil {
			fmt.Fprintf(os.Stderr, "注册到主控失败: %v\n", err)
			os.Exit(1)
		}
		ac = nc
		saveAgentConfig(cfgPath, ac)
		fmt.Printf("已注册到主控 %s（节点 %s）\n", ac.Controller, ac.Name)
	}

	config.SetAgentToken(ac.Token)
	fmt.Printf("EyvesCloud Agent 启动完成，主控: %s，节点: %s\n", ac.Controller, ac.Name)

	go heartbeatLoop(ac)

	// 被控自身也是完整面板，运行本地 Web 服务。
	if err := server.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Web server error: %v\n", err)
		os.Exit(1)
	}
}

func register(controller, installKey, name, addr string) (*agentConfig, error) {
	if installKey == "" {
		return nil, fmt.Errorf("首次注册需要 --install-key 安装密钥")
	}
	if name == "" {
		host, _ := os.Hostname()
		name = host
	}
	payload := map[string]string{
		"install_key": installKey,
		"name":        name,
		"address":     addr,
		"version":     version.Current(),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(strings.TrimSuffix(controller, "/")+"/api/nodes/register", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			NodeID string `json:"node_id"`
			Token  string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if !out.Success || out.Data.NodeID == "" || out.Data.Token == "" {
		return nil, fmt.Errorf("主控返回: %s", out.Message)
	}
	return &agentConfig{
		Controller: controller,
		NodeID:     out.Data.NodeID,
		Token:      out.Data.Token,
		Name:       name,
		Address:    addr,
	}, nil
}

func heartbeatLoop(ac *agentConfig) {
	for {
		sendHeartbeat(ac)
		time.Sleep(10 * time.Second)
	}
}

func sendHeartbeat(ac *agentConfig) {
	payload := collectNodeStatus()
	payload["version"] = version.Current()
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	req, err := http.NewRequest(http.MethodPost,
		strings.TrimSuffix(ac.Controller, "/")+"/api/nodes/"+ac.NodeID+"/heartbeat",
		bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+ac.Token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func collectNodeStatus() map[string]interface{} {
	status := map[string]interface{}{}
	status["cpu_count"] = runtime.NumCPU()
	if totalKB, availKB, ok := readMemInfo(); ok {
		status["ram_total_mb"] = totalKB / 1024
		status["ram_used_mb"] = (totalKB - availKB) / 1024
	}
	if totalBytes, freeBytes, ok := diskUsage("/"); ok {
		totalGB := float64(totalBytes) / 1024 / 1024 / 1024
		usedGB := float64(totalBytes-freeBytes) / 1024 / 1024 / 1024
		status["disk_total_gb"] = totalGB
		status["disk_used_gb"] = usedGB
	}
	status["os_name"] = osName()
	if config.AppConfig != nil {
		config.AppConfigMu.RLock()
		status["container_count"] = len(config.AppConfig.Containers)
		config.AppConfigMu.RUnlock()
	}
	return status
}

func readMemInfo() (totalKB, availableKB int64, ok bool) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, false
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value := int64(0)
		fmt.Sscanf(fields[1], "%d", &value)
		switch fields[0] {
		case "MemTotal:":
			totalKB = value
		case "MemAvailable:":
			availableKB = value
		}
	}
	return totalKB, availableKB, totalKB > 0
}

func diskUsage(path string) (total, free int64, ok bool) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, false
	}
	return int64(stat.Blocks) * int64(stat.Bsize), int64(stat.Bavail) * int64(stat.Bsize), true
}

func osName() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return runtime.GOOS
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "PRETTY_NAME="))
			value = strings.Trim(value, `"`)
			if value != "" {
				return value
			}
		}
	}
	return runtime.GOOS
}

func loadAgentConfig(path string) *agentConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var ac agentConfig
	if err := json.Unmarshal(data, &ac); err != nil || ac.NodeID == "" || ac.Token == "" {
		return nil
	}
	return &ac
}

func saveAgentConfig(path string, ac *agentConfig) {
	data, err := json.MarshalIndent(ac, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0700)
	_ = os.WriteFile(path, data, 0600)
}
