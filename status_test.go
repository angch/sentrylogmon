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
	// Mock data
	startTime := time.Date(2023, 10, 27, 10, 0, 0, 0, time.UTC)
	instances := []ipc.StatusResponse{
		{
			PID:         1234,
			StartTime:   startTime,
			Version:     "v1.0.0",
			MemoryAlloc: 10 * 1024 * 1024, // 10 MiB
			Config: &config.Config{
				Monitors: []config.MonitorConfig{
					{Name: "nginx", Type: "file"},
				},
			},
		},
	}

	var buf bytes.Buffer
	printInstanceTable(&buf, instances)

	output := buf.String()
	t.Logf("Output:\n%s", output)

	// Verify headers
	requiredStrings := []string{
		"STATUS", "PID", "Started", "Uptime", "Mem", "Version", "Monitors",
		"🟢 Running", "1234", "10.0 MiB", "v1.0.0", "nginx(file)",
	}

	for _, s := range requiredStrings {
		if !strings.Contains(output, s) {
			t.Errorf("Output missing expected string: %q", s)
		}
	}
}
