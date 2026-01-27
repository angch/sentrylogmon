package sysstat

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/procfs"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

type ProcessInfo struct {
	Pid     string `json:"pid"`
	RSS     string `json:"rss"`
	CPU     string `json:"cpu"`
	MEM     string `json:"mem"`
	Command string `json:"command"`

	// Internal fields for sorting
	cpuUsage float64
	memUsage float64
}

type PressureInfo struct {
	Avg10  float64 `json:"avg10"`
	Avg60  float64 `json:"avg60"`
	Avg300 float64 `json:"avg300"`
	Total  float64 `json:"total"`
}

type SystemState struct {
	Timestamp      time.Time              `json:"timestamp"`
	Uptime         uint64                 `json:"uptime"`
	Load           *load.AvgStat          `json:"load"`
	Memory         *mem.VirtualMemoryStat `json:"memory"`
	DiskPressure   *PressureInfo          `json:"disk_pressure,omitempty"`
	TopCPU         []ProcessInfo          `json:"top_cpu"`
	TopMem         []ProcessInfo          `json:"top_mem"`
	ProcessSummary string                 `json:"process_summary"`
}

// ToMap converts SystemState to map[string]interface{} for Sentry context.
// This is more efficient than JSON marshal/unmarshal round-trip.
func (s *SystemState) ToMap() map[string]interface{} {
	if s == nil {
		return nil
	}

	result := make(map[string]interface{})
	result["timestamp"] = s.Timestamp
	result["uptime"] = s.Uptime
	result["process_summary"] = s.ProcessSummary

	if s.Load != nil {
		result["load"] = map[string]interface{}{
			"load1":  s.Load.Load1,
			"load5":  s.Load.Load5,
			"load15": s.Load.Load15,
		}
	}

	if s.Memory != nil {
		result["memory"] = map[string]interface{}{
			"total":        s.Memory.Total,
			"available":    s.Memory.Available,
			"used":         s.Memory.Used,
			"used_percent": s.Memory.UsedPercent,
			"free":         s.Memory.Free,
		}
	}

	if s.DiskPressure != nil {
		result["disk_pressure"] = map[string]interface{}{
			"avg10":  s.DiskPressure.Avg10,
			"avg60":  s.DiskPressure.Avg60,
			"avg300": s.DiskPressure.Avg300,
			"total":  s.DiskPressure.Total,
		}
	}

	if len(s.TopCPU) > 0 {
		topCPU := make([]map[string]interface{}, len(s.TopCPU))
		for i, p := range s.TopCPU {
			topCPU[i] = map[string]interface{}{
				"pid":     p.Pid,
				"rss":     p.RSS,
				"cpu":     p.CPU,
				"mem":     p.MEM,
				"command": p.Command,
			}
		}
		result["top_cpu"] = topCPU
	}

	if len(s.TopMem) > 0 {
		topMem := make([]map[string]interface{}, len(s.TopMem))
		for i, p := range s.TopMem {
			topMem[i] = map[string]interface{}{
				"pid":     p.Pid,
				"rss":     p.RSS,
				"cpu":     p.CPU,
				"mem":     p.MEM,
				"command": p.Command,
			}
		}
		result["top_mem"] = topMem
	}

	return result
}

type Collector struct {
	mu       sync.RWMutex
	state    *SystemState
	stopChan chan struct{}
	stopOnce sync.Once
}

func New() *Collector {
	return &Collector{
		state:    &SystemState{},
		stopChan: make(chan struct{}),
	}
}

func (c *Collector) GetState() *SystemState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.state == nil {
		return nil
	}

	// Create a deep copy to avoid data races
	copyState := *c.state

	// Deep-copy pointer fields
	if c.state.Load != nil {
		loadCopy := *c.state.Load
		copyState.Load = &loadCopy
	}

	if c.state.Memory != nil {
		memCopy := *c.state.Memory
		copyState.Memory = &memCopy
	}

	if c.state.DiskPressure != nil {
		dpCopy := *c.state.DiskPressure
		copyState.DiskPressure = &dpCopy
	}

	// Deep-copy slice fields to avoid sharing backing arrays
	if c.state.TopCPU != nil {
		topCPUCopy := make([]ProcessInfo, len(c.state.TopCPU))
		copy(topCPUCopy, c.state.TopCPU)
		copyState.TopCPU = topCPUCopy
	}

	if c.state.TopMem != nil {
		topMemCopy := make([]ProcessInfo, len(c.state.TopMem))
		copy(topMemCopy, c.state.TopMem)
		copyState.TopMem = topMemCopy
	}

	return &copyState
}

func (c *Collector) Run() {
	// Initial collection
	c.collect()

	// Start with 1 minute interval
	currentInterval := 1 * time.Minute
	ticker := time.NewTicker(currentInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.collect()

			// Determine next interval based on load
			c.mu.RLock()
			nextInterval := 1 * time.Minute
			if c.state.Load != nil {
				// If Load1 > NumCPU, consider it high load and back off
				if c.state.Load.Load1 > float64(runtime.NumCPU()) {
					nextInterval = 10 * time.Minute
				}
			}
			c.mu.RUnlock()

			// Only recreate ticker if interval changed to avoid unnecessary overhead
			if nextInterval != currentInterval {
				oldTicker := ticker
				oldTicker.Stop()
				ticker = time.NewTicker(nextInterval)
				currentInterval = nextInterval
			}
		}
	}
}

// Stop gracefully stops the collector goroutine.
// Safe to call multiple times.
func (c *Collector) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopChan)
	})
}

func (c *Collector) collect() {
	newState := &SystemState{
		Timestamp: time.Now(),
	}

	if u, err := host.Uptime(); err == nil {
		newState.Uptime = u
	}
	if l, err := load.Avg(); err == nil {
		newState.Load = l
	}
	if m, err := mem.VirtualMemory(); err == nil {
		newState.Memory = m
	}
	newState.DiskPressure = getDiskPressure()

	procs, summary, err := getProcessStats(newState.Uptime, newState.Memory.Total)
	if err == nil {
		newState.ProcessSummary = summary

		// Sort by CPU
		sort.Slice(procs, func(i, j int) bool {
			return procs[i].cpuUsage > procs[j].cpuUsage
		})
		if len(procs) > 5 {
			newState.TopCPU = procs[:5]
		} else {
			newState.TopCPU = procs
		}

		// Sort by Memory (make a copy or re-sort)
		// Since we want two different lists, we'll re-sort the full list
		// but wait, newState.TopCPU holds references or copies? Copies.
		// So we can re-sort the 'procs' slice.
		sort.Slice(procs, func(i, j int) bool {
			return procs[i].memUsage > procs[j].memUsage
		})
		if len(procs) > 5 {
			newState.TopMem = procs[:5]
		} else {
			newState.TopMem = procs
		}
	} else {
		newState.ProcessSummary = fmt.Sprintf("Error collecting process stats: %v", err)
	}

	c.mu.Lock()
	c.state = newState
	c.mu.Unlock()
}

// getDiskPressure reads Pressure Stall Information (PSI) from /proc/pressure/io.
// PSI is a Linux-specific feature available in kernel 4.20+ and requires
// CONFIG_PSI=y in kernel configuration. Returns nil on other platforms,
// older kernels, or if PSI is disabled.
func getDiskPressure() *PressureInfo {
	content, err := os.ReadFile("/proc/pressure/io")
	if err != nil {
		return nil
	}
	// Format example: some avg10=0.00 avg60=0.00 avg300=0.00 total=0
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "some") {
			parts := strings.Fields(line)
			p := &PressureInfo{}
			for _, part := range parts {
				kv := strings.Split(part, "=")
				if len(kv) != 2 {
					continue
				}
				val, _ := strconv.ParseFloat(kv[1], 64)
				switch kv[0] {
				case "avg10":
					p.Avg10 = val
				case "avg60":
					p.Avg60 = val
				case "avg300":
					p.Avg300 = val
				case "total":
					p.Total = val
				}
			}
			return p
		}
	}
	return nil
}

func getProcessStats(uptime uint64, totalMem uint64) ([]ProcessInfo, string, error) {
	fs, err := procfs.NewFS("/proc")
	if err != nil {
		return nil, "", err
	}

	procs, err := fs.AllProcs()
	if err != nil {
		return nil, "", err
	}

	var results []ProcessInfo
	pageSize := os.Getpagesize()
	clkTck := float64(100) // Default to 100Hz on most systems.
	// To be more accurate we should use sysconf(_SC_CLK_TCK) but that requires CGO or more complex syscalls.
	// For this estimation, 100Hz is a reasonable default for x86 Linux.

	for _, p := range procs {
		stat, err := p.Stat()
		if err != nil {
			continue
		}

		cmd, err := p.CmdLine()
		if err != nil || len(cmd) == 0 {
			// Fallback to Comm if CmdLine is empty or error
			comm, err := p.Comm()
			if err == nil {
				cmd = []string{comm}
			} else {
				cmd = []string{"unknown"}
			}
		}
		commandStr := strings.Join(cmd, " ")

		// CPU Usage: (utime + stime) / (uptime - starttime)
		// Times are in jiffies.
		// Uptime is in seconds.

		totalTicks := float64(stat.UTime + stat.STime)
		startTimeSeconds := float64(stat.Starttime) / clkTck

		var cpuUsage float64
		if float64(uptime) > startTimeSeconds {
			secondsActive := float64(uptime) - startTimeSeconds
			cpuUsage = (totalTicks / clkTck) / secondsActive * 100.0
		}

		// Memory Usage
		rssBytes := float64(stat.RSS * pageSize)
		memUsage := 0.0
		if totalMem > 0 {
			memUsage = (rssBytes / float64(totalMem)) * 100.0
		}

		results = append(results, ProcessInfo{
			Pid:      strconv.Itoa(p.PID),
			RSS:      fmt.Sprintf("%.0f", rssBytes), // Bytes
			CPU:      fmt.Sprintf("%.1f", cpuUsage),
			MEM:      fmt.Sprintf("%.1f", memUsage),
			Command:  commandStr,
			cpuUsage: cpuUsage,
			memUsage: memUsage,
		})
	}

	summary := fmt.Sprintf("Total Processes: %d", len(procs))
	return results, summary, nil
}
