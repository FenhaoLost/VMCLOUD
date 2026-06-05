package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"clicd/internal/config"
	"clicd/internal/lxc"
)

var manager = lxc.NewManager()

// Run starts the CLI interface.
func Run() {
	reader := bufio.NewReader(os.Stdin)

	for {
		clearScreen()
		printMenu()
		fmt.Print("\nSelect action [1-9,0/q]: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch strings.ToLower(input) {
		case "1":
			clearScreen()
			cliListContainers()
			waitEnter(reader)
		case "2":
			clearScreen()
			cliCreateContainer(reader)
			waitEnter(reader)
		case "3":
			clearScreen()
			cliStartContainer(reader)
			waitEnter(reader)
		case "4":
			clearScreen()
			cliStopContainer(reader)
			waitEnter(reader)
		case "5":
			clearScreen()
			cliRestartContainer(reader)
			waitEnter(reader)
		case "6":
			clearScreen()
			cliDeleteContainer(reader)
			waitEnter(reader)
		case "7":
			clearScreen()
			cliReinstallContainer(reader)
			waitEnter(reader)
		case "8":
			clearScreen()
			cliResetPassword(reader)
			waitEnter(reader)
		case "9":
			clearScreen()
			cliToggleWebPanel()
			waitEnter(reader)
		case "0":
			clearScreen()
			cliShowInfo()
			waitEnter(reader)
		case "q", "exit", "quit":
			fmt.Println("Bye")
			return
		default:
			fmt.Println("Invalid selection")
		}
	}
}

func printMenu() {
	webStatus := "start"
	if isWebPanelRunning() {
		webStatus = "stop"
	}
	fmt.Println()
	fmt.Println("  ==========================================")
	fmt.Println("       CLICD - LXC Container Manager")
	fmt.Println("  ==========================================")
	fmt.Println()
	fmt.Printf("  Web panel: %s (port %d)\n", func() string {
		if isWebPanelRunning() {
			return "running"
		}
		return "stopped"
	}(), config.AppConfig.Port)
	fmt.Println()
	fmt.Println("  1. List containers")
	fmt.Println("  2. Create container")
	fmt.Println("  3. Start container")
	fmt.Println("  4. Stop container")
	fmt.Println("  5. Restart container")
	fmt.Println("  6. Delete container")
	fmt.Println("  7. Reinstall container")
	fmt.Println("  8. Reset web admin password")
	fmt.Printf("  9. %s web panel\n", webStatus)
	fmt.Println("  0. System info")
	fmt.Println("  q. Quit")
}

func cliListContainers() {
	containers, err := manager.ListContainers()
	if err != nil {
		fmt.Printf("Failed to list containers: %v\n", err)
		return
	}

	if len(containers) == 0 {
		fmt.Println("\nNo containers")
		return
	}

	fmt.Println()
	fmt.Printf("%-18s %-10s %-18s %-6s %-10s %-10s %-16s\n", "Name", "Status", "Template", "vCPU", "RAM(MB)", "Disk(GB)", "SSH")
	fmt.Println(strings.Repeat("-", 94))
	for _, c := range containers {
		ssh := "-"
		if c.SSHPort > 0 {
			ssh = fmt.Sprintf("%d->22", c.SSHPort)
		}
		fmt.Printf("%-18s %-10s %-18s %-6.2f %-10d %-10d %-16s\n",
			c.Name, c.Status, c.Template, c.VCPU, c.RAMMB, c.DiskGB, ssh)
	}
}

func cliCreateContainer(reader *bufio.Reader) {
	fmt.Println("\n--- Create container ---")

	name := promptString(reader, "Container name", "")
	if name == "" {
		fmt.Println("Container name is required")
		return
	}

	templates := lxc.GetTemplates()
	fmt.Println("\nAvailable templates:")
	for i, template := range templates {
		fmt.Printf("  %d. %s\n", i+1, template.Name)
	}

	tmplIdx := promptInt(reader, fmt.Sprintf("Template [1-%d]", len(templates)), 1)
	if tmplIdx < 1 || tmplIdx > len(templates) {
		fmt.Println("Invalid template selection")
		return
	}

	cfg := lxc.ContainerConfig{
		Name:             name,
		TemplateID:       templates[tmplIdx-1].ID,
		VCPU:             promptFloat(reader, "vCPU", 1),
		RAMMB:            promptInt(reader, "Memory (MB)", 512),
		DiskGB:           promptInt(reader, "Disk (GB)", 10),
		NetworkBWMbps:    promptInt(reader, "Network bandwidth (Mbps)", 100),
		MonthlyTrafficGB: promptInt(reader, "Monthly traffic (GB)", 1000),
		IOSpeedMBps:      promptInt(reader, "IO speed (MB/s)", 500),
		ExtraPorts:       promptPortList(reader, "Extra NAT ports, comma separated"),
	}

	fmt.Printf("\nCreating container %s ...\n", name)
	if err := manager.CreateContainer(cfg); err != nil {
		fmt.Printf("Create failed: %v\n", err)
		return
	}

	container := config.FindContainerByName(name)
	fmt.Printf("Container %s created successfully\n", name)
	if container != nil {
		fmt.Printf("SSH: root / %s, port %d -> 22\n", container.SSHPassword, container.SSHPort)
	}
}

func cliStartContainer(reader *bufio.Reader) {
	id, name := selectContainer(reader, "start")
	if id == 0 {
		return
	}
	if err := manager.StartContainer(id); err != nil {
		fmt.Printf("Start failed: %v\n", err)
		return
	}
	fmt.Printf("Container %s started\n", name)
}

func cliStopContainer(reader *bufio.Reader) {
	id, name := selectContainer(reader, "stop")
	if id == 0 {
		return
	}
	if err := manager.StopContainer(id); err != nil {
		fmt.Printf("Stop failed: %v\n", err)
		return
	}
	fmt.Printf("Container %s stopped\n", name)
}

func cliRestartContainer(reader *bufio.Reader) {
	id, name := selectContainer(reader, "restart")
	if id == 0 {
		return
	}
	if err := manager.RestartContainer(id); err != nil {
		fmt.Printf("Restart failed: %v\n", err)
		return
	}
	fmt.Printf("Container %s restarted\n", name)
}

func cliDeleteContainer(reader *bufio.Reader) {
	id, name := selectContainer(reader, "delete")
	if id == 0 {
		return
	}
	confirm := promptString(reader, fmt.Sprintf("Delete container %s? Type yes", name), "no")
	if strings.ToLower(confirm) != "yes" {
		fmt.Println("Canceled")
		return
	}
	if err := manager.DestroyContainer(id); err != nil {
		fmt.Printf("Delete failed: %v\n", err)
		return
	}
	fmt.Printf("Container %s deleted\n", name)
}

func cliReinstallContainer(reader *bufio.Reader) {
	id, name := selectContainer(reader, "reinstall")
	if id == 0 {
		return
	}

	templates := lxc.GetTemplates()
	fmt.Println("\nAvailable templates:")
	for i, template := range templates {
		fmt.Printf("  %d. %s\n", i+1, template.Name)
	}

	tmplIdx := promptInt(reader, fmt.Sprintf("Template [1-%d]", len(templates)), 1)
	if tmplIdx < 1 || tmplIdx > len(templates) {
		fmt.Println("Invalid template selection")
		return
	}

	confirm := promptString(reader, fmt.Sprintf("Reinstall container %s? Type yes", name), "no")
	if strings.ToLower(confirm) != "yes" {
		fmt.Println("Canceled")
		return
	}

	if err := manager.ReinstallContainer(id, templates[tmplIdx-1].ID); err != nil {
		fmt.Printf("Reinstall failed: %v\n", err)
		return
	}
	fmt.Printf("Container %s reinstalled\n", name)
}

func cliResetPassword(reader *bufio.Reader) {
	newPass := promptString(reader, "New admin password (at least 6 chars)", "")
	if len(newPass) < 6 {
		fmt.Println("Password must be at least 6 chars")
		return
	}
	confirm := promptString(reader, "Confirm password", "")
	if newPass != confirm {
		fmt.Println("Passwords do not match")
		return
	}

	if err := config.ResetAdminPassword(newPass); err != nil {
		fmt.Printf("Reset failed: %v\n", err)
		return
	}
	fmt.Println("Admin password reset. Restart the web service for it to take effect.")
}

func cliToggleWebPanel() {
	if isWebPanelRunning() {
		cmd := exec.Command("systemctl", "stop", "clicd")
		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed to stop web panel: %v\n", err)
			return
		}
		fmt.Println("Web panel stopped. LXC containers are not affected.")
		return
	}

	cmd := exec.Command("systemctl", "start", "clicd")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to start web panel: %v\n", err)
		return
	}
	fmt.Println("Web panel started")
}

func isWebPanelRunning() bool {
	cmd := exec.Command("systemctl", "is-active", "clicd")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "active"
}

func cliShowInfo() {
	containers, err := manager.ListContainers()
	if err != nil {
		fmt.Printf("Failed to read container status: %v\n", err)
	}

	total := len(containers)
	running := 0
	for _, container := range containers {
		if container.Status == "running" {
			running++
		}
	}

	fmt.Println("\n--- System info ---")
	fmt.Printf("Web port: %d\n", config.AppConfig.Port)
	fmt.Printf("Admin user: %s\n", config.AppConfig.AdminUser)
	fmt.Printf("Containers: %d\n", total)
	fmt.Printf("Running: %d\n", running)
	fmt.Printf("Stopped: %d\n", total-running)

	if hostname, err := os.Hostname(); err == nil {
		fmt.Printf("Hostname: %s\n", hostname)
	}

	cmd := exec.Command("lxc-info", "--version")
	output, err := cmd.Output()
	if err == nil {
		fmt.Printf("LXC version: %s", string(output))
	}
}

func selectContainer(reader *bufio.Reader, action string) (int, string) {
	containers, err := manager.ListContainers()
	if err != nil {
		fmt.Printf("Failed to list containers: %v\n", err)
		return 0, ""
	}
	if len(containers) == 0 {
		fmt.Println("No containers available")
		return 0, ""
	}

	fmt.Printf("\n--- Select container to %s ---\n", action)
	for i, container := range containers {
		fmt.Printf("  %d. [%d] %s [%s]\n", i+1, container.ID, container.Name, container.Status)
	}

	idx := promptInt(reader, "Container", 0)
	if idx < 1 || idx > len(containers) {
		fmt.Println("Invalid selection")
		return 0, ""
	}

	c := containers[idx-1]
	return c.ID, c.Name
}

func promptString(reader *bufio.Reader, label string, fallback string) string {
	if fallback == "" {
		fmt.Printf("%s: ", label)
	} else {
		fmt.Printf("%s [%s]: ", label, fallback)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return fallback
	}
	return input
}

func promptInt(reader *bufio.Reader, label string, fallback int) int {
	input := promptString(reader, label, strconv.Itoa(fallback))
	value, err := strconv.Atoi(input)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func promptFloat(reader *bufio.Reader, label string, fallback float64) float64 {
	input := promptString(reader, label, strconv.FormatFloat(fallback, 'f', -1, 64))
	value, err := strconv.ParseFloat(input, 64)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func waitEnter(reader *bufio.Reader) {
	fmt.Print("\nPress Enter to return to menu...")
	reader.ReadString('\n')
}

func promptPortList(reader *bufio.Reader, label string) []int {
	input := promptString(reader, label, "")
	if input == "" {
		return nil
	}

	var ports []int
	for _, part := range strings.Split(input, ",") {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || value <= 0 || value > 65535 {
			fmt.Printf("Ignoring invalid port: %s\n", strings.TrimSpace(part))
			continue
		}
		ports = append(ports, value)
	}
	return ports
}
