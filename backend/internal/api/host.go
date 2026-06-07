package api

import (
	"bufio"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"clicd/internal/lxc"

	"golang.org/x/sys/unix"
)

type HostInfo struct {
	CPU     CpuInfo     `json:"cpu"`
	RAM     MemoryInfo  `json:"ram"`
	Disk    DiskInfo    `json:"disk"`
	Network NetworkInfo `json:"network"`
	DiskIO  DiskIOInfo  `json:"disk_io"`
	Load    LoadInfo    `json:"load"`
}

type LoadInfo struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

type CpuInfo struct {
	Cores int     `json:"cores"`
	Usage float64 `json:"usage_pct"`
}

type MemoryInfo struct {
	TotalMB int64 `json:"total_mb"`
	UsedMB  int64 `json:"used_mb"`
	FreeMB  int64 `json:"free_mb"`
}

type DiskInfo struct {
	TotalGB float64 `json:"total_gb"`
	UsedGB  float64 `json:"used_gb"`
	FreeGB  float64 `json:"free_gb"`
}

type NetworkInfo struct {
	RXBytes             uint64               `json:"rx_bytes"`
	TXBytes             uint64               `json:"tx_bytes"`
	RXBps               float64              `json:"rx_bps"`
	TXBps               float64              `json:"tx_bps"`
	PublicIPv4          string               `json:"public_ipv4"`
	PublicIPv4Interface string               `json:"public_ipv4_interface"`
	PublicIPv6          string               `json:"public_ipv6"`
	PublicIPv6Interface string               `json:"public_ipv6_interface"`
	IPv6Prefixes        []lxc.IPv6PrefixInfo `json:"ipv6_prefixes"`
}

type DiskIOInfo struct {
	ReadBytes  uint64  `json:"read_bytes"`
	WriteBytes uint64  `json:"write_bytes"`
	ReadBps    float64 `json:"read_bps"`
	WriteBps   float64 `json:"write_bps"`
}

var hostCPUMu sync.Mutex
var lastHostCPU cpuTimes
var hostIOMu sync.Mutex
var lastHostIO hostIOSample

type cpuTimes struct {
	Total uint64
	Idle  uint64
}

type hostIOSample struct {
	RXBytes    uint64
	TXBytes    uint64
	ReadBytes  uint64
	WriteBytes uint64
	At         int64
}

func getHostInfo() HostInfo {
	info := HostInfo{
		CPU: CpuInfo{Cores: runtime.NumCPU()},
	}

	info.RAM = getMemoryInfo()
	info.Disk = getDiskInfo()
	info.CPU.Usage = getCPUUsage()
	info.Network, info.DiskIO = getHostRates()
	info.Load = getLoadInfo()
	return info
}

func getMemoryInfo() MemoryInfo {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return MemoryInfo{TotalMB: 0, UsedMB: 0, FreeMB: 0}
	}
	defer f.Close()

	var total, available, free int64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.ParseInt(fields[1], 10, 64)
		switch fields[0] {
		case "MemTotal:":
			total = val / 1024
		case "MemAvailable:":
			available = val / 1024
		case "MemFree:":
			free = val / 1024
		}
	}

	used := total - available
	if available == 0 {
		used = total - free
	}

	return MemoryInfo{
		TotalMB: total,
		UsedMB:  used,
		FreeMB:  available,
	}
}

func getDiskInfo() DiskInfo {
	var stat unix.Statfs_t
	if err := unix.Statfs("/", &stat); err != nil {
		// Try command-based fallback
		cmd := exec.Command("df", "-BG", "/")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			if len(lines) >= 2 {
				fields := strings.Fields(lines[1])
				if len(fields) >= 4 {
					total, _ := parseSizeGBf(fields[1])
					used, _ := parseSizeGBf(fields[2])
					free, _ := parseSizeGBf(fields[3])
					return DiskInfo{TotalGB: total, UsedGB: used, FreeGB: free}
				}
			}
		}
		return DiskInfo{}
	}

	total := float64(int64(stat.Blocks)*int64(stat.Bsize)) / (1024 * 1024 * 1024)
	free := float64(int64(stat.Bavail)*int64(stat.Bsize)) / (1024 * 1024 * 1024)
	used := total - free

	return DiskInfo{
		TotalGB: total,
		UsedGB:  used,
		FreeGB:  free,
	}
}

func getCPUUsage() float64 {
	current, err := readCPUTimes()
	if err != nil {
		return 0
	}

	hostCPUMu.Lock()
	defer hostCPUMu.Unlock()

	if lastHostCPU.Total == 0 {
		lastHostCPU = current
		return 0
	}

	totalDelta := current.Total - lastHostCPU.Total
	idleDelta := current.Idle - lastHostCPU.Idle
	lastHostCPU = current

	if totalDelta == 0 {
		return 0
	}

	usage := (1 - float64(idleDelta)/float64(totalDelta)) * 100
	if usage < 0 {
		return 0
	}
	if usage > 100 {
		return 100
	}
	return usage
}

func readCPUTimes() (cpuTimes, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return cpuTimes{}, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return cpuTimes{}, scanner.Err()
	}

	fields := strings.Fields(scanner.Text())
	if len(fields) < 8 || fields[0] != "cpu" {
		return cpuTimes{}, nil
	}

	var values []uint64
	for _, field := range fields[1:] {
		value, _ := strconv.ParseUint(field, 10, 64)
		values = append(values, value)
	}

	var total uint64
	for _, value := range values {
		total += value
	}

	idle := values[3]
	if len(values) > 4 {
		idle += values[4]
	}

	return cpuTimes{Total: total, Idle: idle}, nil
}

func parseSizeGB(s string) (int64, error) {
	s = strings.TrimSuffix(s, "G")
	s = strings.TrimSpace(s)
	val, err := strconv.ParseInt(s, 10, 64)
	return val, err
}

func parseSizeGBf(s string) (float64, error) {
	s = strings.TrimSuffix(s, "G")
	s = strings.TrimSpace(s)
	val, err := strconv.ParseFloat(s, 64)
	return val, err
}

func getHostRates() (NetworkInfo, DiskIOInfo) {
	rx, tx := readHostNetworkBytes()
	readBytes, writeBytes := readHostDiskBytes()
	now := unixNano()

	network := NetworkInfo{RXBytes: rx, TXBytes: tx}
	publicIPv4 := lxc.DetectPublicIPv4()
	network.PublicIPv4 = publicIPv4.Address
	network.PublicIPv4Interface = publicIPv4.Interface
	network.IPv6Prefixes = lxc.DetectPublicIPv6Prefixes()
	if len(network.IPv6Prefixes) > 0 {
		network.PublicIPv6 = network.IPv6Prefixes[0].Address
		network.PublicIPv6Interface = network.IPv6Prefixes[0].Interface
	}
	diskIO := DiskIOInfo{ReadBytes: readBytes, WriteBytes: writeBytes}

	hostIOMu.Lock()
	defer hostIOMu.Unlock()

	if lastHostIO.At == 0 {
		lastHostIO = hostIOSample{RXBytes: rx, TXBytes: tx, ReadBytes: readBytes, WriteBytes: writeBytes, At: now}
		return network, diskIO
	}

	elapsed := float64(now-lastHostIO.At) / 1_000_000_000
	if elapsed > 0 {
		if rx >= lastHostIO.RXBytes {
			network.RXBps = float64(rx-lastHostIO.RXBytes) / elapsed
		}
		if tx >= lastHostIO.TXBytes {
			network.TXBps = float64(tx-lastHostIO.TXBytes) / elapsed
		}
		if readBytes >= lastHostIO.ReadBytes {
			diskIO.ReadBps = float64(readBytes-lastHostIO.ReadBytes) / elapsed
		}
		if writeBytes >= lastHostIO.WriteBytes {
			diskIO.WriteBps = float64(writeBytes-lastHostIO.WriteBytes) / elapsed
		}
	}

	lastHostIO = hostIOSample{RXBytes: rx, TXBytes: tx, ReadBytes: readBytes, WriteBytes: writeBytes, At: now}
	return network, diskIO
}

func readHostNetworkBytes() (uint64, uint64) {
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return 0, 0
	}

	var rx, tx uint64
	for _, entry := range entries {
		name := entry.Name()
		if name == "lo" {
			continue
		}
		rx += readUintFile("/sys/class/net/" + name + "/statistics/rx_bytes")
		tx += readUintFile("/sys/class/net/" + name + "/statistics/tx_bytes")
	}
	return rx, tx
}

func readHostDiskBytes() (uint64, uint64) {
	f, err := os.Open("/proc/diskstats")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	var readSectors, writeSectors uint64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 14 {
			continue
		}
		device := fields[2]
		if strings.HasPrefix(device, "loop") ||
			strings.HasPrefix(device, "ram") ||
			strings.HasPrefix(device, "fd") ||
			strings.HasPrefix(device, "sr") {
			continue
		}
		read, _ := strconv.ParseUint(fields[5], 10, 64)
		write, _ := strconv.ParseUint(fields[9], 10, 64)
		readSectors += read
		writeSectors += write
	}
	return readSectors * 512, writeSectors * 512
}

func readUintFile(path string) uint64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	value, _ := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	return value
}

func unixNano() int64 {
	return time.Now().UnixNano()
}

func getLoadInfo() LoadInfo {
	f, err := os.Open("/proc/loadavg")
	if err != nil {
		return LoadInfo{}
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return LoadInfo{}
	}

	fields := strings.Fields(scanner.Text())
	if len(fields) < 3 {
		return LoadInfo{}
	}

	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)
	return LoadInfo{Load1: load1, Load5: load5, Load15: load15}
}
