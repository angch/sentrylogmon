package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/config"
	"github.com/angch/sentrylogmon/ipc"
)

func TestPrintInstanceTable(t *testing.T) {
	// Setup test data
	startTime := time.Date(2023, 10, 27, 10, 0, 0, 0, time.UTC)
	instances := []ipc.StatusResponse{
		{
			PID:         1234,
			StartTime:   startTime,
			Version:     "v1.0.0",
			MemoryAlloc: 1572864, // 1.5 MiB
			Config: &config.Config{
				Monitors: []config.MonitorConfig{
					{Name: "nginx", Type: "file"},
					{Name: "dmesg", Type: "dmesg"},
					{Name: "custom", Type: "command"},
				},
			},
		},
		{
			PID:         5678,
			StartTime:   startTime.Add(1 * time.Hour),
			Version:     "v1.1.0",
			MemoryAlloc: 1024, // 1 KiB
			Config: &config.Config{
				Monitors: []config.MonitorConfig{
					{Name: "solo", Type: "solo"},
				},
			},
		},
	}

	var buf bytes.Buffer
	printInstanceTable(&buf, instances)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Check header
	if !strings.Contains(lines[0], "PID") || !strings.Contains(lines[0], "STARTED") || !strings.Contains(lines[0], "MONITORS") {
		t.Errorf("Header missing expected columns. Got: %s", lines[0])
	}

	// Check instance 1
	// PID: 1234
	// STARTED: 2023-10-27 10:00 (16 chars)
	// MEM: 1.5 MiB
	// MONITORS: nginx [file], dmesg, custom [command]
	if !strings.Contains(output, "1234") {
		t.Error("Instance 1 PID not found")
	}
	if !strings.Contains(output, "2023-10-27 10:00") {
		t.Error("Instance 1 StartTime format incorrect")
	}
	if !strings.Contains(output, "1.5 MiB") {
		t.Error("Instance 1 Memory format incorrect")
	}
	// Check monitor formatting
	if !strings.Contains(output, "nginx [file]") {
		t.Error("Instance 1 monitor 'nginx [file]' format incorrect")
	}
	if !strings.Contains(output, "dmesg") {
		t.Error("Instance 1 monitor 'dmesg' format incorrect (should not have [dmesg])")
	}
	if strings.Contains(output, "dmesg [dmesg]") {
		t.Error("Instance 1 monitor 'dmesg' has redundant type info")
	}
	if !strings.Contains(output, "custom [command]") {
		t.Error("Instance 1 monitor 'custom [command]' format incorrect")
	}

	// Check instance 2
	if !strings.Contains(output, "5678") {
		t.Error("Instance 2 PID not found")
	}
	if !strings.Contains(output, "solo") {
		t.Error("Instance 2 monitor 'solo' format incorrect")
	}
	if strings.Contains(output, "solo [solo]") {
		t.Error("Instance 2 monitor 'solo' has redundant type info")
	}

	// Check footer
	if !strings.Contains(output, "Total instances: 2") {
		t.Error("Footer missing or incorrect")
	}
}

func TestPrintInstanceTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	printInstanceTable(&buf, []ipc.StatusResponse{})
	output := buf.String()

	if !strings.Contains(output, "No running instances found") {
		t.Errorf("Expected 'No running instances found', got: %s", output)
	}
}
