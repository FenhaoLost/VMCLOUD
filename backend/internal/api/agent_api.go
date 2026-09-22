package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"eyvescloud/internal/config"
	"eyvescloud/internal/lxc"
)

// 被控节点（agent 模式）专用 API，仅供主控（Controller）通过节点 token 调用。

// AgentTokenMiddleware 校验请求携带的主控 token。
func AgentTokenMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentToken := config.AgentToken()
		if agentToken == "" {
			jsonResponse(w, http.StatusForbidden, APIResponse{Success: false, Message: "Agent API is not enabled on this node"})
			return
		}
		token := tokenFromRequest(r)
		if token == "" || token != agentToken {
			jsonResponse(w, http.StatusUnauthorized, APIResponse{Success: false, Message: "Invalid agent token"})
			return
		}
		next(w, r)
	}
}

// HandleAgentContainers 返回本机容器列表（供主控查看）。
func HandleAgentContainers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	containers, err := listByRuntime()
	if err != nil {
		containers = config.AppConfig.Containers
	}
	if containers == nil {
		containers = []config.Container{}
	}
	jsonResponse(w, http.StatusOK, APIResponse{Success: true, Data: containers})
}

// HandleAgentContainerAction 执行本机容器的电源操作（供主控下发）。
func HandleAgentContainerAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/agent/containers/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid container action path"})
		return
	}
	id, err := strconv.Atoi(parts[0])
	if err != nil || id <= 0 {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid container ID"})
		return
	}
	action := parts[1]
	c := config.FindContainer(id)
	if c == nil {
		jsonResponse(w, http.StatusNotFound, APIResponse{Success: false, Message: "Container not found"})
		return
	}
	var runErr error
	switch action {
	case "start":
		runErr = startByRuntime(id)
	case "stop":
		runErr = stopByRuntime(id)
	case "restart":
		runErr = restartByRuntime(id)
	case "reset-password":
		newPassword, pwErr := agentResetPassword(id, r)
		if pwErr != nil {
			jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Message: pwErr.Error()})
			return
		}
		jsonResponse(w, http.StatusOK, APIResponse{Success: true, Message: "SSH password reset successfully", Data: map[string]string{"password": newPassword}})
		return
	default:
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Unknown action: " + action})
		return
	}
	if runErr != nil {
		jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Message: runErr.Error()})
		return
	}
	// 同步状态到配置
	switch action {
	case "start", "restart":
		config.UpdateContainerStatus(id, "running")
	case "stop":
		config.UpdateContainerStatus(id, "stopped")
	}
	jsonResponse(w, http.StatusOK, APIResponse{Success: true, Message: "OK"})
}

// agentResetPassword 处理母控下发的子容器密码重置。未填密码时由运行时自动生成新密码。
func agentResetPassword(id int, r *http.Request) (string, error) {
	var req struct {
		Password string `json:"password"`
	}
	if r.Body != nil {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&req); err != nil && err.Error() != "EOF" {
			return "", err
		}
	}
	password := strings.TrimSpace(req.Password)
	if password != "" {
		if err := lxc.ValidateCustomSSHPassword(password); err != nil {
			return "", err
		}
	}
	return resetPasswordByRuntime(id, password)
}

// AgentContainerActionFromQuery 兼容 /api/agent/containers?action=start&id=1 的调用形式。
func AgentContainerActionFromQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	var req struct {
		ID     int    `json:"id"`
		Action string `json:"action"`
	}
	body := r.Body
	if body != nil {
		_ = json.NewDecoder(body).Decode(&req)
	}
	if req.ID <= 0 {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: "container id required"})
		return
	}
	rr := r.Clone(r.Context())
	rr.URL.Path = "/api/agent/containers/" + strconv.Itoa(req.ID) + "/" + req.Action
	HandleAgentContainerAction(w, rr)
}
