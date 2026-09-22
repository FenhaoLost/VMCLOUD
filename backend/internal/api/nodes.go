package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"eyvescloud/internal/config"
)

// 主控（Controller）节点管理 API。
// 被控节点通过一键安装脚本注册到主控，此后主控可以直接查看/操作被控的容器。

func randomNodeSecret(bytesLen int) string {
	b := make([]byte, bytesLen)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func newNodeID() string {
	return "node-" + randomNodeSecret(6)
}

// HandleNodes 列出或创建被控节点。
func HandleNodes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !requireScope(w, r, "node:read") {
			return
		}
		reconcileNodeOnlineStatuses()
		config.AppConfigMu.RLock()
		nodes := append([]config.Node(nil), config.AppConfig.Nodes...)
		config.AppConfigMu.RUnlock()
		if nodes == nil {
			nodes = []config.Node{}
		}
		jsonResponse(w, http.StatusOK, APIResponse{Success: true, Data: nodes})
	case http.MethodPost:
		if !requireScope(w, r, "node:write") {
			return
		}
		createNode(w, r)
	default:
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
	}
}

// HandleNodeSubRoutes 分发 /api/nodes/{id}[/...] 的各类子操作。
func HandleNodeSubRoutes(w http.ResponseWriter, r *http.Request) {
	nodeID, rest, ok := splitNodeSubPath(r.URL.Path)
	if !ok || nodeID == "" {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Node ID required"})
		return
	}
	switch {
	case rest == "" && nodeID == "register" && r.Method == http.MethodPost:
		handleNodeRegister(w, r)
	case rest == "":
		AdminMiddleware(func(w http.ResponseWriter, r *http.Request) { handleNodeItem(w, r, nodeID) })(w, r)
	case rest == "heartbeat":
		handleNodeHeartbeat(w, r, nodeID)
	case rest == "install-script":
		AdminMiddleware(func(w http.ResponseWriter, r *http.Request) { handleNodeInstallScript(w, r, nodeID) })(w, r)
	case rest == "containers" && r.Method == http.MethodGet:
		AdminMiddleware(func(w http.ResponseWriter, r *http.Request) { handleNodeContainers(w, r, nodeID) })(w, r)
	case strings.HasPrefix(rest, "containers/") && r.Method == http.MethodPost:
		AdminMiddleware(func(w http.ResponseWriter, r *http.Request) { handleNodeContainerAction(w, r, nodeID, strings.TrimPrefix(rest, "containers/")) })(w, r)
	default:
		jsonResponse(w, http.StatusNotFound, APIResponse{Success: false, Message: "Action not found"})
	}
}

// handleNodeRegister 由被控 agent 首次接入时调用，使用安装密钥换取节点 token。
func handleNodeRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InstallKey string `json:"install_key"`
		Name       string `json:"name"`
		Address    string `json:"address"`
		Version    string `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request body"})
		return
	}
	if strings.TrimSpace(req.InstallKey) == "" {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: "install_key required"})
		return
	}
	node, ok := config.FindNodeByInstallKey(strings.TrimSpace(req.InstallKey))
	if !ok || node.Token == "" {
		jsonResponse(w, http.StatusUnauthorized, APIResponse{Success: false, Message: "Invalid install key"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = node.Name
	}
	address := normalizeNodeAddress(req.Address)
	if address != "" {
		if err := validateNodeAddress(address); err != nil {
			jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: err.Error()})
			return
		}
	}
	_, ok = config.UpdateNode(node.ID, func(n *config.Node) {
		n.Status = "online"
		n.LastSeen = time.Now().Format("2006-01-02 15:04:05")
		n.Name = name
		n.Version = req.Version
		if address != "" {
			n.Address = address
		}
	})
	if !ok {
		jsonResponse(w, http.StatusNotFound, APIResponse{Success: false, Message: "Node not found"})
		return
	}
	jsonResponse(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{
		"node_id": node.ID,
		"token":   node.Token,
		"name":    name,
	}})
}

func createNode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request body"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "node-" + randomNodeSecret(4)
	}
	address := normalizeNodeAddress(req.Address)
	if address != "" {
		if err := validateNodeAddress(address); err != nil {
			jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: err.Error()})
			return
		}
	}
	node := config.Node{
		ID:         newNodeID(),
		Name:       name,
		Address:    address,
		Token:      randomNodeSecret(32),
		InstallKey: randomNodeSecret(32),
		Status:     "pending",
		CreatedAt:  time.Now().Format("2006-01-02 15:04:05"),
	}
	if err := config.AddNode(node); err != nil {
		jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Message: err.Error()})
		return
	}
	auditRequest(r, "node.create", node.Name, "创建被控节点", true, "")
	jsonResponse(w, http.StatusCreated, APIResponse{Success: true, Data: node})
}

func handleNodeItem(w http.ResponseWriter, r *http.Request, nodeID string) {
	switch r.Method {
	case http.MethodGet:
		if !requireScope(w, r, "node:read") {
			return
		}
		node, ok := config.FindNode(nodeID)
		if !ok {
			jsonResponse(w, http.StatusNotFound, APIResponse{Success: false, Message: "Node not found"})
			return
		}
		jsonResponse(w, http.StatusOK, APIResponse{Success: true, Data: node})
	case http.MethodDelete:
		if !requireScope(w, r, "node:write") {
			return
		}
		node, ok := config.FindNode(nodeID)
		if !ok {
			jsonResponse(w, http.StatusNotFound, APIResponse{Success: false, Message: "Node not found"})
			return
		}
		config.RemoveNode(nodeID)
		auditRequest(r, "node.delete", node.Name, "删除被控节点", true, "")
		jsonResponse(w, http.StatusOK, APIResponse{Success: true, Message: "Node deleted"})
	default:
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
	}
}

// handleNodeHeartbeat 由被控 agent 周期性上报资源与在线状态。
func handleNodeHeartbeat(w http.ResponseWriter, r *http.Request, nodeID string) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	token := tokenFromRequest(r)
	node, ok := config.FindNode(nodeID)
	if !ok || node.Token == "" || token != node.Token {
		jsonResponse(w, http.StatusUnauthorized, APIResponse{Success: false, Message: "Invalid node token"})
		return
	}
	var req struct {
		Version        string  `json:"version"`
		OSName         string  `json:"os_name"`
		CPUCount       int     `json:"cpu_count"`
		RAMTotalMB     int64   `json:"ram_total_mb"`
		RAMUsedMB      int64   `json:"ram_used_mb"`
		DiskTotalGB    float64 `json:"disk_total_gb"`
		DiskUsedGB     float64 `json:"disk_used_gb"`
		ContainerCount int     `json:"container_count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request body"})
		return
	}
	_, ok = config.UpdateNode(nodeID, func(n *config.Node) {
		n.LastSeen = time.Now().Format("2006-01-02 15:04:05")
		n.Status = "online"
		if req.Version != "" {
			n.Version = req.Version
		}
		if req.OSName != "" {
			n.OSName = req.OSName
		}
		if req.CPUCount > 0 {
			n.CPUCount = req.CPUCount
		}
		if req.RAMTotalMB > 0 {
			n.RAMTotalMB = req.RAMTotalMB
		}
		n.RAMUsedMB = req.RAMUsedMB
		if req.DiskTotalGB > 0 {
			n.DiskTotalGB = req.DiskTotalGB
		}
		n.DiskUsedGB = req.DiskUsedGB
		n.ContainerCount = req.ContainerCount
	})
	if !ok {
		jsonResponse(w, http.StatusNotFound, APIResponse{Success: false, Message: "Node not found"})
		return
	}
	jsonResponse(w, http.StatusOK, APIResponse{Success: true, Message: "ok"})
}

// handleNodeInstallScript 生成被控一键安装脚本。
func handleNodeInstallScript(w http.ResponseWriter, r *http.Request, nodeID string) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	if !requireScope(w, r, "node:write") {
		return
	}
	node, ok := config.FindNode(nodeID)
	if !ok {
		jsonResponse(w, http.StatusNotFound, APIResponse{Success: false, Message: "Node not found"})
		return
	}
	controller := requestOrigin(r)
	if controller == "" {
		controller = "http://127.0.0.1:8999"
	}
	// 无需也无法在母侧确定子的真实地址：地址留空，由子端 agent 通过与母建连推算，或用脚本第 2 个参数显式指定。
	defaultAddr := ""
	script := buildAgentInstallScript(controller, node.InstallKey, node.Name, defaultAddr)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=eyvescloud-agent-%s.sh", node.Name))
	_, _ = w.Write([]byte(script))
}

func buildAgentInstallScript(controller, installKey, nodeName, defaultAddr string) string {
	name := shellDQ(nodeName)
	addr := shellDQ(defaultAddr)
	controllerEsc := shellDQ(controller)
	keyEsc := shellDQ(installKey)
	nameSQ := shellEscape(nodeName)
	addrSQ := shellEscape(defaultAddr)
	controllerSQ := shellEscape(controller)
	keySQ := shellEscape(installKey)
	return fmt.Sprintf(`#!/bin/bash
# EyvesCloud 被控节点一键安装脚本
# 用法: bash %s.sh [节点名称] [被控面板地址]
set -e

CONTROLLER="%s"
INSTALL_KEY="%s"
NODE_NAME="${1:-%s}"
NODE_ADDR="${2:-%s}"

if [ "$(id -u)" -ne 0 ]; then
  echo "请使用 root 权限运行: sudo bash $0"
  exit 1
fi

command -v curl >/dev/null 2>&1 || { echo "缺少 curl，请先安装"; exit 1; }

echo "==> [1/3] 下载 EyvesCloud 二进制"
curl -fsSL -o /usr/local/bin/eyvescloud "$CONTROLLER/api/nodes/binary"
chmod +x /usr/local/bin/eyvescloud

echo "==> [2/3] 注册被控节点"
/usr/local/bin/eyvescloud agent --controller="$CONTROLLER" --install-key="$INSTALL_KEY" --name="$NODE_NAME" --addr="$NODE_ADDR" || {
  echo "注册失败（已注册过的节点可忽略）";
}

echo "==> [3/3] 配置自启动服务"
cat > /etc/systemd/system/eyvescloud-agent.service <<'UNIT'
[Unit]
Description=EyvesCloud Agent
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/eyvescloud agent --controller=%s --install-key=%s --name=%s --addr=%s
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
UNIT
systemctl daemon-reload
systemctl enable --now eyvescloud-agent

echo ""
echo "=============================================="
echo "  EyvesCloud Agent 安装完成"
echo "  节点: $NODE_NAME"
echo "  主控: $CONTROLLER"
echo "  请回到主控面板查看节点状态"
echo "=============================================="
`,
		nameSQ, controllerEsc, keyEsc, name, addr,
		controllerSQ, keySQ, nameSQ, addrSQ,
	)
}

// shellDQ 转义用于双引号包裹的 shell 变量值。
func shellDQ(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "$", `\$`, "`", "\\`")
	return replacer.Replace(value)
}

func defaultNodeAddress(r *http.Request) string {
	host := strings.TrimSpace(r.Host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if host == "" || host == "localhost" {
		host = "127.0.0.1"
	}
	return "http://" + host + ":8999"
}

// HandleNodeBinary 供被控一键安装脚本下载主控二进制（同一二进制支持 agent 模式）。
func HandleNodeBinary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	exe, err := os.Executable()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	f, err := os.Open(exe)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename=eyvescloud`)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	_, _ = io.Copy(w, f)
}

func handleNodeContainers(w http.ResponseWriter, r *http.Request, nodeID string) {
	node, ok := config.FindNode(nodeID)
	if !ok {
		jsonResponse(w, http.StatusNotFound, APIResponse{Success: false, Message: "Node not found"})
		return
	}
	if node.Address == "" {
		jsonResponse(w, http.StatusBadGateway, APIResponse{Success: false, Message: "节点未配置地址，无法代理访问"})
		return
	}
	data, status, err := proxyNodeRequest(r, node, http.MethodGet, "/api/agent/containers", nil)
	if err != nil {
		jsonResponse(w, http.StatusBadGateway, APIResponse{Success: false, Message: "代理请求被控节点失败: " + err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func handleNodeContainerAction(w http.ResponseWriter, r *http.Request, nodeID, rest string) {
	node, ok := config.FindNode(nodeID)
	if !ok {
		jsonResponse(w, http.StatusNotFound, APIResponse{Success: false, Message: "Node not found"})
		return
	}
	if node.Address == "" {
		jsonResponse(w, http.StatusBadGateway, APIResponse{Success: false, Message: "节点未配置地址，无法代理访问"})
		return
	}
	// 转发容器操作请求体（如重置密码的 {password}），供子端 agent 使用。
	data, status, err := proxyNodeRequest(r, node, http.MethodPost, "/api/agent/containers/"+strings.TrimPrefix(rest, "/"), r.Body)
	if err != nil {
		jsonResponse(w, http.StatusBadGateway, APIResponse{Success: false, Message: "代理请求被控节点失败: " + err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

// nodeOnlineTimeout 心跳间隔为 10s；超过该窗口仍未心跳则视为离线。
const nodeOnlineTimeout = 60 * time.Second

// reconcileNodeOnlineStatuses 将超过心跳超时仍为 online 的节点置为 offline（离线检测）。
// 在节点列表读取前调用；心跳续期由 handleNodeHeartbeat 负责。
func reconcileNodeOnlineStatuses() {
	now := time.Now()
	var stale []config.Node
	config.AppConfigMu.RLock()
	for _, n := range config.AppConfig.Nodes {
		if n.Status != "online" || n.LastSeen == "" {
			continue
		}
		last, err := time.ParseInLocation("2006-01-02 15:04:05", n.LastSeen, time.Local)
		if err == nil && now.Sub(last) > nodeOnlineTimeout {
			stale = append(stale, n)
		}
	}
	config.AppConfigMu.RUnlock()
	for _, n := range stale {
		config.UpdateNode(n.ID, func(x *config.Node) { x.Status = "offline" })
	}
}

func splitNodeSubPath(path string) (nodeID, rest string, ok bool) {
	trimmed := strings.TrimPrefix(path, "/api/nodes/")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", "", false
	}
	rest = ""
	if len(parts) > 1 {
		rest = parts[1]
	}
	return parts[0], rest, true
}

func proxyNodeRequest(r *http.Request, node config.Node, method, path string, body io.Reader) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(r.Context(), method, strings.TrimSuffix(node.Address, "/")+path, body)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	req.Header.Set("Authorization", "Bearer "+node.Token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return data, resp.StatusCode, nil
}

func normalizeNodeAddress(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		value = "http://" + value
	}
	return strings.TrimSuffix(value, "/")
}

func validateNodeAddress(value string) error {
	u, err := url.Parse(value)
	if err != nil || u.Host == "" {
		return fmt.Errorf("无效的节点地址: %s", value)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("节点地址 scheme 必须为 http 或 https")
	}
	return nil
}

func shellEscape(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
