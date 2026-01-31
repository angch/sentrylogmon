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
	"github.com/tklauser/go-sysconf"
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
	proc     procfs.Proc
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

type Collector struct {
	mu    sync.RWMutex
	state *SystemState
}

func New() *Collector {
	return &Collector{
		state: &SystemState{},
	}
}

// ToMap converts the SystemState to a map[string]interface{}.
// This is optimized to avoid double JSON marshaling (struct -> json -> map -> json)
// when sending context to Sentry.
func (s *SystemState) ToMap() map[string]interface{} {
	if s == nil {
		return nil
	}
	m := map[string]interface{}{
		"timestamp":       s.Timestamp,
		"uptime":          s.Uptime,
		"load":            s.Load,
		"memory":          s.Memory,
		"top_cpu":         s.TopCPU,
		"top_mem":         s.TopMem,
		"process_summary": s.ProcessSummary,
	}
	if s.DiskPressure != nil {
		m["disk_pressure"] = s.DiskPressure
	}
	return m
}

func (c *Collector) GetState() *SystemState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

func (c *Collector) Run() {
	// Initial collection
	c.collect()

	for {
		sleepDuration := 1 * time.Minute

		c.mu.RLock()
		if c.state.Load != nil {
			// If Load1 > NumCPU, consider it high load and back off
			if c.state.Load.Load1 > float64(runtime.NumCPU()) {
				sleepDuration = 10 * time.Minute
			}
		}
		c.mu.RUnlock()

		time.Sleep(sleepDuration)
		c.collect()
	}
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

		// Get Top CPU
		newState.TopCPU = getTopKProcesses(procs, 5, func(i, j ProcessInfo) bool {
			return i.cpuUsage > j.cpuUsage
		})
		for i := range newState.TopCPU {
			fetchCommand(&newState.TopCPU[i])
		}

		// Get Top Memory
		newState.TopMem = getTopKProcesses(procs, 5, func(i, j ProcessInfo) bool {
			return i.memUsage > j.memUsage
		})
		for i := range newState.TopMem {
			fetchCommand(&newState.TopMem[i])
		}
	} else {
		newState.ProcessSummary = fmt.Sprintf("Error collecting process stats: %v", err)
	}

	c.mu.Lock()
	c.state = newState
	c.mu.Unlock()
}

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

func fetchCommand(p *ProcessInfo) {
	if p.Command != "" {
		return
	}
	cmd, err := p.proc.CmdLine()
	if err != nil || len(cmd) == 0 {
		// Fallback to Comm if CmdLine is empty or error
		comm, err := p.proc.Comm()
		if err == nil {
			cmd = []string{comm}
		} else {
			cmd = []string{"unknown"}
		}
	}
	p.Command = SanitizeCommand(cmd)
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
	clkTck := float64(100) // Default fallback
	if val, err := sysconf.Sysconf(sysconf.SC_CLK_TCK); err == nil {
		clkTck = float64(val)
	}

	for _, p := range procs {
		stat, err := p.Stat()
		if err != nil {
			continue
		}

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
			Command:  "",
			cpuUsage: cpuUsage,
			memUsage: memUsage,
			proc:     p,
		})
	}

	summary := fmt.Sprintf("Total Processes: %d", len(procs))
	return results, summary, nil
}

func getTopKProcesses(procs []ProcessInfo, k int, more func(i, j ProcessInfo) bool) []ProcessInfo {
	if len(procs) <= k {
		result := make([]ProcessInfo, len(procs))
		copy(result, procs)
		sort.Slice(result, func(i, j int) bool {
			return more(result[i], result[j])
		})
		return result
	}

	result := make([]ProcessInfo, 0, k)

	for _, p := range procs {
		if len(result) < k {
			result = append(result, p)
			if len(result) == k {
				sort.Slice(result, func(i, j int) bool {
					return more(result[i], result[j])
				})
			}
		} else {
			if more(p, result[k-1]) {
				pos := k - 1
				for pos > 0 && more(p, result[pos-1]) {
					result[pos] = result[pos-1]
					pos--
				}
				result[pos] = p
			}
		}
	}
	return result
}
