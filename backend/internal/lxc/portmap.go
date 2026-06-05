package lxc

import (
	"fmt"
	"os/exec"
	"strconv"

	"clicd/internal/config"
)

// ApplyPortMappings applies iptables DNAT rules for a container's port mappings
func (m *Manager) ApplyPortMappings(id int) error {
	c := config.FindContainer(id)
	if c == nil {
		return fmt.Errorf("container not found: %d", id)
	}
	if c.IP == "" {
		return fmt.Errorf("container has no IP")
	}
	tag := clicdTag(id)

	EnsureForwardRules()
	m.CleanPortMappings(id)

	for _, pm := range c.PortMappings {
		cmd := exec.Command("iptables",
			"-t", "nat",
			"-I", "PREROUTING", "1",
			"-p", pm.Protocol,
			"--dport", fmt.Sprintf("%d", pm.HostPort),
			"-j", "DNAT",
			"--to-destination", fmt.Sprintf("%s:%d", c.IP, pm.ContainerPort),
			"-m", "comment", "--comment", fmt.Sprintf("clicd-%s-%d", tag, pm.HostPort),
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Warning: failed to apply port mapping %d->%s:%d: %v, output: %s\n",
				pm.HostPort, c.IP, pm.ContainerPort, err, string(output))
			continue
		}
		fmt.Printf("Port mapping: host:%d -> %s:%d\n", pm.HostPort, c.IP, pm.ContainerPort)
	}

	if exec.Command("iptables", "-t", "nat", "-C", "POSTROUTING", "-s", "10.0.3.0/24", "-o", "eth+", "-j", "MASQUERADE").Run() != nil {
		exec.Command("iptables", "-t", "nat", "-I", "POSTROUTING", "1", "-s", "10.0.3.0/24", "-o", "eth+", "-j", "MASQUERADE").Run()
	}

	return nil
}

func clicdTag(id int) string { return "c" + strconv.Itoa(id) }

// EnsureForwardRules makes sure iptables FORWARD chain allows LXC bridge traffic
func EnsureForwardRules() {
	rules := [][]string{
		{"-A", "FORWARD", "-i", "lxcbr0", "-j", "ACCEPT"},
		{"-A", "FORWARD", "-o", "lxcbr0", "-j", "ACCEPT"},
		{"-A", "FORWARD", "-i", "lxcbr0", "-o", "lxcbr0", "-j", "ACCEPT"},
	}
	for _, args := range rules {
		checkArgs := append([]string{"-C", "FORWARD"}, args[2:]...)
		if exec.Command("iptables", checkArgs...).Run() != nil {
			exec.Command("iptables", args...).Run()
		}
	}
}

// CleanPortMappings removes all iptables rules for a container
func (m *Manager) CleanPortMappings(id int) error {
	tag := clicdTag(id)
	cmd := exec.Command("sh", "-c",
		fmt.Sprintf("iptables -t nat -L PREROUTING -n --line-numbers 2>/dev/null | grep 'clicd-%s' | awk '{print $1}' | sort -rn | while read num; do iptables -t nat -D PREROUTING $num; done", tag))
	cmd.Run()
	return nil
}

// SetupDefaultPortMappings creates default port mappings
func SetupDefaultPortMappings(sshPort int) []config.PortMapping {
	return []config.PortMapping{
		{ContainerPort: 22, HostPort: sshPort, Protocol: "tcp", Description: "SSH"},
	}
}

// AddPortMapping adds a NAT rule to a container
func (m *Manager) AddPortMapping(id int, pm config.PortMapping) ([]config.PortMapping, error) {
	c := config.FindContainer(id)
	if c == nil {
		return nil, fmt.Errorf("container not found: %d", id)
	}
	if c.PortMappingLimit > 0 && len(c.PortMappings) >= c.PortMappingLimit {
		return nil, fmt.Errorf("port mapping quota exceeded: %d/%d", len(c.PortMappings), c.PortMappingLimit)
	}
	normalized, err := normalizePortMapping(c, -1, pm)
	if err != nil {
		return nil, err
	}
	c.PortMappings = append(c.PortMappings, normalized)
	if err := persistAndReloadMappings(m, c); err != nil {
		return nil, err
	}
	return c.PortMappings, nil
}

// UpdatePortMapping updates an existing NAT rule
func (m *Manager) UpdatePortMapping(id int, index int, pm config.PortMapping) ([]config.PortMapping, error) {
	c := config.FindContainer(id)
	if c == nil {
		return nil, fmt.Errorf("container not found: %d", id)
	}
	if index < 0 || index >= len(c.PortMappings) {
		return nil, fmt.Errorf("invalid port mapping index: %d", index)
	}
	normalized, err := normalizePortMapping(c, index, pm)
	if err != nil {
		return nil, err
	}
	c.PortMappings[index] = normalized
	if err := persistAndReloadMappings(m, c); err != nil {
		return nil, err
	}
	return c.PortMappings, nil
}

// DeletePortMapping removes a NAT rule
func (m *Manager) DeletePortMapping(id int, index int) ([]config.PortMapping, error) {
	c := config.FindContainer(id)
	if c == nil {
		return nil, fmt.Errorf("container not found: %d", id)
	}
	if index < 0 || index >= len(c.PortMappings) {
		return nil, fmt.Errorf("invalid port mapping index: %d", index)
	}
	if c.PortMappings[index].Description == "SSH" {
		return nil, fmt.Errorf("SSH default mapping cannot be deleted")
	}
	c.PortMappings = append(c.PortMappings[:index], c.PortMappings[index+1:]...)
	if err := persistAndReloadMappings(m, c); err != nil {
		return nil, err
	}
	return c.PortMappings, nil
}

func persistAndReloadMappings(m *Manager, c *config.Container) error {
	config.SaveConfig()
	if c.Status == "running" && c.IP != "" {
		return m.ApplyPortMappings(c.ID)
	}
	return nil
}

func normalizePortMapping(c *config.Container, skipIndex int, pm config.PortMapping) (config.PortMapping, error) {
	if pm.ContainerPort < 1 || pm.ContainerPort > 65535 {
		return pm, fmt.Errorf("container port must be 1-65535")
	}
	if pm.Protocol == "" {
		pm.Protocol = "tcp"
	}
	if pm.Description == "" {
		pm.Description = fmt.Sprintf("Port-%d", pm.ContainerPort)
	}
	if pm.HostPort <= 0 {
		pm.HostPort = pm.ContainerPort
	}
	for i, existing := range c.PortMappings {
		if i == skipIndex {
			continue
		}
		if existing.HostPort == pm.HostPort && existing.Protocol == pm.Protocol {
			return pm, fmt.Errorf("host port %d/%s already mapped", pm.HostPort, pm.Protocol)
		}
	}
	return pm, nil
}

func allocateDefaultEqualPorts(c *config.Container, count int) []int {
	if count <= 0 {
		return nil
	}
	used := map[int]bool{}
	for _, pm := range c.PortMappings {
		used[pm.HostPort] = true
		used[pm.ContainerPort] = true
	}
	ports := make([]int, 0, count)
	next := 20000
	for len(ports) < count {
		if !used[next] {
			ports = append(ports, next)
		}
		next++
		if next > 65535 || len(ports) >= count {
			break
		}
	}
	return ports
}
