package api

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"eyvescloud/internal/config"
)

type ContainerMetricPoint struct {
	TS        int64   `json:"ts"`
	CPU       float64 `json:"cpu"`
	Memory    float64 `json:"memory"`
	Network   float64 `json:"network"`
	NetworkRx float64 `json:"network_rx"`
	NetworkTx float64 `json:"network_tx"`
	DiskIO    float64 `json:"disk_io"`
	DiskRead  float64 `json:"disk_read"`
	DiskWrite float64 `json:"disk_write"`
}

var containerMetricSamplerOnce sync.Once
var containerMetricMu sync.RWMutex
var containerMetricHistory = map[string][]ContainerMetricPoint{}
var containerMetricInFlight sync.Map

const (
	containerMetricSampleInterval = 30 * time.Second
	containerMetricSampleTimeout  = 20 * time.Second
	containerMetricConcurrency    = 4
)

func StartContainerMetricSampler() {
	containerMetricSamplerOnce.Do(func() {
		go func() {
			sampleAllContainerMetrics()
			ticker := time.NewTicker(containerMetricSampleInterval)
			defer ticker.Stop()
			for range ticker.C {
				sampleAllContainerMetrics()
			}
		}()
	})
}

func sampleAllContainerMetrics() {
	containers, _ := listByRuntime()
	sem := make(chan struct{}, containerMetricConcurrency)
	var wg sync.WaitGroup

	for _, c := range containers {
		c := c
		if c.Status != "running" {
			continue
		}
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			sampleContainerMetricWithTimeout(c)
		}()
	}
	wg.Wait()
	pruneContainerMetricHistory()
}

func sampleContainerMetricWithTimeout(c config.Container) {
	key := containerMetricKey(c)
	if key == "" {
		return
	}
	if _, loaded := containerMetricInFlight.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	done := make(chan struct{}, 1)
	go func() {
		defer containerMetricInFlight.Delete(key)
		if usage, err := usageByRuntime(c.ID); err == nil {
			appendContainerMetricPoint(c, usage)
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(containerMetricSampleTimeout):
	}
}

func appendContainerMetricPoint(c config.Container, usage map[string]interface{}) {
	key := containerMetricKey(c)
	if key == "" {
		return
	}
	memoryTotal := numberFromUsage(usage, "memory_total_bytes")
	if memoryTotal <= 0 {
		memoryTotal = float64(c.RAMMB) * 1024 * 1024
	}
	memoryPct := 0.0
	if memoryTotal > 0 {
		memoryPct = clampPercent(numberFromUsage(usage, "memory_usage_bytes") / memoryTotal * 100)
	}
	vcpu := c.VCPU
	if vcpu <= 0 {
		vcpu = 1
	}
	cpuPct := clampPercent(numberFromUsage(usage, "cpu_usage_pct") / vcpu)
	networkRx := positiveNumberFromUsage(usage, "network_rx_bps")
	networkTx := positiveNumberFromUsage(usage, "network_tx_bps")
	diskRead := positiveNumberFromUsage(usage, "disk_read_bps")
	diskWrite := positiveNumberFromUsage(usage, "disk_write_bps")
	point := ContainerMetricPoint{
		TS:        time.Now().UnixMilli(),
		CPU:       cpuPct,
		Memory:    memoryPct,
		NetworkRx: networkRx,
		NetworkTx: networkTx,
		Network:   networkRx + networkTx,
		DiskRead:  diskRead,
		DiskWrite: diskWrite,
		DiskIO:    diskRead + diskWrite,
	}
	cutoff := time.Now().Add(-hostMetricRetention).UnixMilli()

	containerMetricMu.Lock()
	defer containerMetricMu.Unlock()

	history := containerMetricHistory[key]
	keepFrom := 0
	for keepFrom < len(history) && history[keepFrom].TS < cutoff {
		keepFrom++
	}
	if keepFrom > 0 {
		copy(history, history[keepFrom:])
		history = history[:len(history)-keepFrom]
	}
	containerMetricHistory[key] = append(history, point)

	// 持久化到 SQLite，面板重启后历史不丢失
	go func(key string, point ContainerMetricPoint) {
		_ = config.SaveMetricSamples(key, []config.MetricSample{{
			TS:        point.TS,
			CPU:       point.CPU,
			Memory:    point.Memory,
			NetworkRx: point.NetworkRx,
			NetworkTx: point.NetworkTx,
			DiskRead:  point.DiskRead,
			DiskWrite: point.DiskWrite,
		}})
	}(key, point)
}

func getContainerMetricHistory(c *config.Container) []ContainerMetricPoint {
	if c == nil {
		return nil
	}
	key := containerMetricKey(*c)
	cutoff := time.Now().Add(-hostMetricRetention).UnixMilli()

	// 合并 SQLite 持久化历史与内存最新采样
	dbSamples, _ := config.LoadMetricSamples(key, cutoff)
	merged := make([]ContainerMetricPoint, 0, len(dbSamples)+8)
	seen := map[int64]bool{}
	for _, s := range dbSamples {
		if s.TS < cutoff || seen[s.TS] {
			continue
		}
		seen[s.TS] = true
		merged = append(merged, ContainerMetricPoint{
			TS:        s.TS,
			CPU:       s.CPU,
			Memory:    s.Memory,
			NetworkRx: s.NetworkRx,
			NetworkTx: s.NetworkTx,
			Network:   s.NetworkRx + s.NetworkTx,
			DiskRead:  s.DiskRead,
			DiskWrite: s.DiskWrite,
			DiskIO:    s.DiskRead + s.DiskWrite,
		})
	}

	containerMetricMu.RLock()
	mem := containerMetricHistory[key]
	for _, p := range mem {
		if p.TS < cutoff || seen[p.TS] {
			continue
		}
		seen[p.TS] = true
		merged = append(merged, p)
	}
	containerMetricMu.RUnlock()

	sort.Slice(merged, func(i, j int) bool { return merged[i].TS < merged[j].TS })
	if merged == nil {
		return []ContainerMetricPoint{}
	}
	return merged
}

func pruneContainerMetricHistory() {
	cutoff := time.Now().Add(-hostMetricRetention).UnixMilli()
	valid := map[string]bool{}
	if config.AppConfig != nil {
		config.AppConfigMu.RLock()
		containers := append([]config.Container(nil), config.AppConfig.Containers...)
		config.AppConfigMu.RUnlock()
		for _, c := range containers {
			valid[containerMetricKey(c)] = true
		}
	}

	// 清理 SQLite 持久化历史（过期 + 已删除容器）
	_ = config.PruneMetricSamples(cutoff)
	_ = config.PruneMetricSamplesForContainers(valid)

	containerMetricMu.Lock()
	defer containerMetricMu.Unlock()

	for key, history := range containerMetricHistory {
		if !valid[key] {
			delete(containerMetricHistory, key)
			continue
		}
		keepFrom := 0
		for keepFrom < len(history) && history[keepFrom].TS < cutoff {
			keepFrom++
		}
		if keepFrom > 0 {
			copy(history, history[keepFrom:])
			containerMetricHistory[key] = history[:len(history)-keepFrom]
		}
	}
}

func containerMetricKey(c config.Container) string {
	if c.UUID != "" {
		return "uuid:" + c.UUID
	}
	if c.ID > 0 {
		return fmt.Sprintf("id:%d", c.ID)
	}
	if c.Name != "" {
		return "name:" + c.Name
	}
	return ""
}

func numberFromUsage(usage map[string]interface{}, key string) float64 {
	value, ok := usage[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0
		}
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case uint:
		return float64(v)
	case uint64:
		return float64(v)
	case uint32:
		return float64(v)
	case json.Number:
		n, _ := v.Float64()
		return n
	case string:
		n, _ := strconv.ParseFloat(v, 64)
		return n
	default:
		return 0
	}
}

func positiveNumberFromUsage(usage map[string]interface{}, key string) float64 {
	value := numberFromUsage(usage, key)
	if value < 0 {
		return 0
	}
	return value
}
