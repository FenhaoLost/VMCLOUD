package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"clicd/internal/api"
	"clicd/internal/cli"
	"clicd/internal/config"
	"clicd/internal/lxc"
	"clicd/internal/server"

	"golang.org/x/term"
)

func main() {
	isTerminal := term.IsTerminal(int(os.Stdin.Fd()))

	isServerMode := false
	isCliMode := false
	noWebAutostart := false
	for _, arg := range os.Args[1:] {
		if arg == "server" || arg == "-s" || arg == "--server" {
			isServerMode = true
		}
		if arg == "cli" || arg == "-c" || arg == "--cli" {
			isCliMode = true
		}
		if arg == "--no-web" || arg == "--cli-only" {
			noWebAutostart = true
			isCliMode = true
		}
	}

	// Initialize config
	cfg, err := config.InitConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize config: %v\n", err)
		os.Exit(1)
	}
	_ = cfg

	if isServerMode || (!isTerminal && !isCliMode) {
		// Restore persisted state
		api.RestoreTasks()
		api.RestoreLoginLogs()

		// Start security scanner
		api.InitScanner()

		// Ensure iptables FORWARD rules allow LXC traffic
		lxc.EnsureForwardRules()

		// Start expiry scanner (stops expired containers every 30s)
		manager := lxc.NewManager()
		manager.StartExpiryScanner()

		// Start usage monitor (computes CPU/network/disk rates every 5s)
		manager.StartUsageMonitor()

		// Clean up stale container configs (LXC dir was deleted but config remains)
		config.CleanStaleContainers()

		// Pre-warm SSH for containers already running after host boot or service restart.
		manager.StartSSHWarmupScanner()

		// Run in server mode (frontend embedded in binary)
		if err := server.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// CLI mode normally keeps the web panel available. Use --no-web to avoid
		// starting the systemd web service on locked-down hosts.
		if !noWebAutostart && !isWebPanelSystemdRunning() {
			startWebPanelSystemd()
		}

		// Run CLI interface
		cli.Run()
	}
}

func isWebPanelSystemdRunning() bool {
	cmd := exec.Command("systemctl", "is-active", "clicd")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "active"
}

func startWebPanelSystemd() {
	cmd := exec.Command("systemctl", "start", "clicd")
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "警告: 自动启动 Web 面板失败: %v\n", err)
	} else {
		fmt.Println("Web 面板已自动启动")
	}
}
