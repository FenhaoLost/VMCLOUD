package api

import (
	"net/http"
	"sync"
	"time"

	"eyvescloud/internal/cli"
	"eyvescloud/internal/version"
)

// 版本检查结果带短缓存：避免每个面板请求都打 GitHub API（会触发限流）。
// 面板「系统设置 → 版本」每次进入页面触发一次检查，缓存 10 分钟。

type cachedUpdateCheck struct {
	mu        sync.Mutex
	checkedAt time.Time
	result    cli.CheckUpdateResult
}

var updateCheckCache cachedUpdateCheck

// checkUpdateCached 返回缓存中的版本检测结果；超过 ttl 重新检测。
func checkUpdateCached(ttl time.Duration) cli.CheckUpdateResult {
	updateCheckCache.mu.Lock()
	defer updateCheckCache.mu.Unlock()

	if !updateCheckCache.checkedAt.IsZero() && time.Since(updateCheckCache.checkedAt) < ttl {
		return updateCheckCache.result
	}

	result := cli.CheckForUpdate()
	// 缓存任何结果（包括错误），避免连续失败打满 GitHub API。
	updateCheckCache.result = result
	updateCheckCache.checkedAt = time.Now()
	return result
}

// HandleCheckUpdate 返回当前版本与最新版本对比，供面板提示“有可用更新”。
// 仅做检测，不下载也不重启服务（升级由 install.sh / CLI 完成）。
func HandleCheckUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	result := checkUpdateCached(10 * time.Minute)
	data := map[string]interface{}{
		"current":    version.Current(),
		"latest":     result.Latest,
		"has_update": result.HasUpdate,
		"err":        result.Err,
	}
	jsonResponse(w, http.StatusOK, APIResponse{Success: true, Data: data})
}