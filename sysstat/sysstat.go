package sysstat

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

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

	// Top CPU
	newState.TopCPU = getTopProcesses("-%cpu")
	// Top Mem
	newState.TopMem = getTopProcesses("-%mem")

	// Process Summary
	out, err := exec.Command("sh", "-c", "ps -e | wc -l").Output()
	if err == nil {
		count := strings.TrimSpace(string(out))
		newState.ProcessSummary = fmt.Sprintf("Total Processes: %s", count)
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

func getTopProcesses(sortFlag string) []ProcessInfo {
	// ps -eo pid,rss,pcpu,pmem,args --sort=-%cpu
	cmd := exec.Command("ps", "-eo", "pid,rss,pcpu,pmem,args", "--sort="+sortFlag)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(string(out), "\n")
	var res []ProcessInfo
	// Skip header (line 0)
	if len(lines) < 2 {
		return nil
	}

	// We want top 5.
	count := 0
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 5 {
			continue
		}

		res = append(res, ProcessInfo{
			Pid:     parts[0],
			RSS:     parts[1],
			CPU:     parts[2],
			MEM:     parts[3],
			Command: strings.Join(parts[4:], " "),
		})
		count++
		if count >= 5 {
			break
		}
	}
	return res
}
