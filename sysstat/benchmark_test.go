package sysstat

import (
	"math/rand"
	"sort"
	"strconv"
	"testing"
)

func generateProcs(n int) []ProcessInfo {
	procs := make([]ProcessInfo, n)
	for i := 0; i < n; i++ {
		procs[i] = ProcessInfo{
			Pid:      strconv.Itoa(i),
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
