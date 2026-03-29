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
	startTime := time.Now().Add(-1 * time.Hour)
	instances := []ipc.StatusResponse{
		{
			PID:         1234,
			StartTime:   startTime,
			Version:     "v1.0.0",
			MemoryAlloc: 1024 * 1024 * 50, // 50 MiB
			Config: &config.Config{
				Monitors: []config.MonitorConfig{
					{Name: "test-monitor", Type: "file"},
				},
			},
		},
	}

	var buf bytes.Buffer
	printInstanceTable(&buf, instances)

	output := buf.String()

	// Check headers
	if !strings.Contains(output, "PID") || !strings.Contains(output, "Status") || !strings.Contains(output, "Started") {
		t.Errorf("Output missing expected headers: %s", output)
	}

	// Check content
	if !strings.Contains(output, "1234") {
		t.Errorf("Output missing PID: %s", output)
	}
	if !strings.Contains(output, "🟢 Running") {
		t.Errorf("Output missing status indicator: %s", output)
	}
	if !strings.Contains(output, "50.0 MiB") {
		t.Errorf("Output missing memory usage: %s", output)
	}
	if !strings.Contains(output, "test-monitor(file)") {
		t.Errorf("Output missing monitor details: %s", output)
	}
}

func TestPrintInstanceTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	printInstanceTable(&buf, []ipc.StatusResponse{})
	output := buf.String()
	if !strings.Contains(output, "No running instances found") {
		t.Errorf("Output missing empty state message: %s", output)
	}
}
