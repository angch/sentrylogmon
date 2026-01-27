package sysstat

import (
	"testing"
	"time"
)

func TestCollect(t *testing.T) {
	c := New()
	c.collect()

	state := c.GetState()
	if state == nil {
		t.Fatal("State should not be nil")
	}

	if state.Uptime == 0 {
		t.Log("Uptime is 0, possibly expected in some environments or very fresh boot")
	}

	if state.Load == nil {
		t.Error("Load should not be nil")
	}

	if state.Memory == nil {
		t.Error("Memory should not be nil")
	}

	if len(state.TopCPU) == 0 {
		t.Log("TopCPU is empty, possibly due to missing ps command or permissions")
	} else {
		if len(state.TopCPU) > 5 {
			t.Error("TopCPU has too many entries")
		}
	}

	if len(state.TopMem) == 0 {
		t.Log("TopMem is empty, possibly due to missing ps command or permissions")
	} else {
		if len(state.TopMem) > 5 {
			t.Error("TopMem has too many entries")
		}
	}

	if time.Since(state.Timestamp) > 1*time.Minute {
		t.Error("Timestamp is too old")
	}
}

func TestToMap(t *testing.T) {
	c := New()
	// Populate state manually to avoid relying on system stats
	c.mu.Lock()
	c.state = &SystemState{
		Timestamp:      time.Now(),
		Uptime:         12345,
		ProcessSummary: "Summary",
		TopCPU: []ProcessInfo{
			{Pid: "1", Command: "init", CPU: "0.1"},
		},
	}
	c.mu.Unlock()

	state := c.GetState()
	m := state.ToMap()

	if m["uptime"] != uint64(12345) {
		t.Errorf("Expected uptime 12345, got %v", m["uptime"])
	}
	if m["process_summary"] != "Summary" {
		t.Errorf("Expected process_summary 'Summary', got %v", m["process_summary"])
	}

	topCPU, ok := m["top_cpu"].([]map[string]interface{})
	if !ok {
		t.Fatal("top_cpu is not []map[string]interface{}")
	}
	if len(topCPU) != 1 {
		t.Errorf("Expected 1 top_cpu entry, got %d", len(topCPU))
	}
	if topCPU[0]["pid"] != "1" {
		t.Errorf("Expected pid 1, got %v", topCPU[0]["pid"])
	}
}
