package api

import (
	"testing"

	"eyvescloud/internal/config"
)

// TestFilterSubUserAuditLogsExactMatch 保障子用户审计日志按用户名精确匹配，
// 不允许通过 substring 误匹配到其他子用户的记录（回归 M5）。
func TestFilterSubUserAuditLogsExactMatch(t *testing.T) {
	previous := config.AppConfig
	t.Cleanup(func() { config.AppConfig = previous })

	config.AppConfig = &config.EyvescloudConfig{
		AuditLogs: []config.AuditLog{
			{Action: "container.start", Target: "c1", User: "alice"},          // 精确命中
			{Action: "container.stop", Target: "c2", User: "user:alice"},      // 精确命中（子用户 actor 前缀）
			{Action: "container.start", Target: "c3", User: "al"},             // 短前缀：若用 Contains 会误命中 alice，必须拒
			{Action: "container.start", Target: "c4", User: "alice2"},         // 相邻子串，不得命中
			{Action: "container.start", Target: "c5", User: "alice_evil"},     // 前缀子串，不得命中
			{Action: "container.start", Target: "c6", User: "bob"},            // 无关用户，不得命中
			{Action: "container.start", Target: "c7", User: "admin"},          // 管理员，不得命中
		},
	}

	got := filterSubUserAuditLogs("alice")
	want := 2
	if len(got) != want {
		t.Fatalf("matched %d logs, want %d (got: %+v)", len(got), want, got)
	}
	for _, l := range got {
		if l.User != "alice" && l.User != "user:alice" {
			t.Fatalf("unexpected log matched with user=%q", l.User)
		}
	}
}

// TestFilterSubUserLoginLogsExactMatch 保障子用户登录日志按用户名精确匹配。
func TestFilterSubUserLoginLogsExactMatch(t *testing.T) {
	previous := config.AppConfig
	t.Cleanup(func() { config.AppConfig = previous })

	config.AppConfig = &config.EyvescloudConfig{
		LoginLogs: []config.SavedLoginLog{
			{Username: "alice"},
			{Username: "alice2"},
			{Username: "alice_dex"},
			{Username: "bob"},
		},
	}

	got := filterSubUserLoginLogs("alice")
	if len(got) != 1 || got[0].Username != "alice" {
		t.Fatalf("got %+v, want only exact 'alice'", got)
	}
}