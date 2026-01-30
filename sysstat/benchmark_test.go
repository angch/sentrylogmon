package sysstat

import (
	"encoding/json"
	"math/rand"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

func generateProcs(n int) []ProcessInfo {
	procs := make([]ProcessInfo, n)
	for i := 0; i < n; i++ {
		procs[i] = ProcessInfo{
			Pid:      strconv.Itoa(i),
			RSS:      "123456",
			CPU:      "10.5",
			MEM:      "5.2",
			Command:  "some-command --flag",
			cpuUsage: rand.Float64() * 100,
			memUsage: rand.Float64() * 100,
		}
	}
	return procs
}

func BenchmarkProcessSort_Baseline(b *testing.B) {
	input := generateProcs(10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Need copy because sort modifies in place
		procs := make([]ProcessInfo, len(input))
		copy(procs, input)

		var topCPU []ProcessInfo
		var topMem []ProcessInfo

		// Old Logic (mimicked)
		sort.Slice(procs, func(i, j int) bool {
			return procs[i].cpuUsage > procs[j].cpuUsage
		})
		if len(procs) > 5 {
			topCPU = procs[:5]
		} else {
			topCPU = procs
		}

		sort.Slice(procs, func(i, j int) bool {
			return procs[i].memUsage > procs[j].memUsage
		})
		if len(procs) > 5 {
			topMem = procs[:5]
		} else {
			topMem = procs
		}

		_ = topCPU
		_ = topMem
	}
}

func BenchmarkProcessSort_Optimized(b *testing.B) {
	input := generateProcs(10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// No copy needed as getTopKProcesses is non-destructive
		procs := input

		topCPU := getTopKProcesses(procs, 5, func(a, b ProcessInfo) bool {
			return a.cpuUsage > b.cpuUsage
		})

		topMem := getTopKProcesses(procs, 5, func(a, b ProcessInfo) bool {
			return a.memUsage > b.memUsage
		})

		_ = topCPU
		_ = topMem
	}
}

func BenchmarkSerialization_AntiPattern(b *testing.B) {
	state := &SystemState{
		Timestamp: time.Now(),
		Uptime:    12345,
		Load: &load.AvgStat{
			Load1:  1.5,
			Load5:  1.2,
			Load15: 1.0,
		},
		Memory: &mem.VirtualMemoryStat{
			Total:       16000000000,
			Available:   8000000000,
			Used:        8000000000,
			UsedPercent: 50.0,
		},
		TopCPU:         generateProcs(5),
		TopMem:         generateProcs(5),
		ProcessSummary: "Total Processes: 100",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Anti-pattern: Marshal -> Unmarshal -> map
		data, _ := json.Marshal(state)
		var stateMap map[string]interface{}
		_ = json.Unmarshal(data, &stateMap)

		// Simulate Sentry usage: SetContext uses map, which eventually gets Marshaled
		_, _ = json.Marshal(stateMap)
	}
}

func BenchmarkSerialization_Current(b *testing.B) {
	state := &SystemState{
		Timestamp: time.Now(),
		Uptime:    12345,
		Load: &load.AvgStat{
			Load1:  1.5,
			Load5:  1.2,
			Load15: 1.0,
		},
		Memory: &mem.VirtualMemoryStat{
			Total:       16000000000,
			Available:   8000000000,
			Used:        8000000000,
			UsedPercent: 50.0,
		},
		TopCPU:         generateProcs(5),
		TopMem:         generateProcs(5),
		ProcessSummary: "Total Processes: 100",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Current: ToMap (returns map with structs)
		stateMap := state.ToMap()

		// Simulate Sentry usage: SetContext uses map, which eventually gets Marshaled
		_, _ = json.Marshal(stateMap)
	}
}
