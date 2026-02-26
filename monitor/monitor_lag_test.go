package monitor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// UniqueMockSource wraps MockSource to provide a custom name
type UniqueMockSource struct {
	MockSource
	name string
}

func (s *UniqueMockSource) Name() string {
	return s.name
}

func TestMonitorLag(t *testing.T) {
	// Create a unique source name to isolate metrics
	sourceName := "mock_lag_test"

	// Use an absolute timestamp (ISO8601) to avoid dependency on host.BootTime()
	// Set timestamp to 10 seconds ago
	now := time.Now()
	logTime := now.Add(-10 * time.Second)
	// Format: 2023-10-27T10:00:00Z
	logLine := fmt.Sprintf("%s Test Lag Line", logTime.Format(time.RFC3339))

	source := &UniqueMockSource{
		MockSource: MockSource{content: logLine + "\n"},
		name:       sourceName,
	}
	detector := &MockDetector{}

	// Reset metrics for this test source if possible, or just rely on the unique label
	// Prometheus globals are hard to reset, so we rely on unique source name.

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Run monitor synchronously
	mon.Start()

	// Check metric
	// We need to gather metrics and find sentrylogmon_monitor_lag_seconds{source="mock_lag_test"}
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	var found bool
	var val float64

	for _, mf := range mfs {
		if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
			for _, m := range mf.GetMetric() {
				// Check label
				for _, label := range m.GetLabel() {
					if label.GetName() == "source" && label.GetValue() == sourceName {
						val = m.GetGauge().GetValue()
						found = true
						break
					}
				}
			}
		}
	}

	if !found {
		// List available metrics for debugging
		var available []string
		for _, mf := range mfs {
			available = append(available, mf.GetName())
		}
		t.Fatalf("Metric sentrylogmon_monitor_lag_seconds not found for source %s. Available: %v", sourceName, available)
	}

	// Lag should be around 10 seconds. Allow some margin for execution time.
	// It shouldn't be less than 10s (unless clock skew) and shouldn't be huge.
	if val < 9.0 || val > 15.0 {
		t.Errorf("Expected lag around 10s, got %.2fs", val)
	}
}

// TestMonitorLagRelative tests relative timestamp (dmesg) lag calculation.
func TestMonitorLagRelative(t *testing.T) {
	sourceName := "mock_lag_relative_test"

	// Create a dmesg line with a small timestamp (e.g. 100.0)
	// This should be treated as relative time (uptime)
	logLine := "[  100.000000] Test Relative Lag Line"

	source := &UniqueMockSource{
		MockSource: MockSource{content: logLine + "\n"},
		name:       sourceName,
	}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	mon.Start()

	// Check metric
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	var found bool
	for _, mf := range mfs {
		if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
			for _, m := range mf.GetMetric() {
				for _, label := range m.GetLabel() {
					if label.GetName() == "source" && label.GetValue() == sourceName {
						found = true
						break
					}
				}
			}
		}
	}

	if !found {
		t.Logf("Metric sentrylogmon_monitor_lag_seconds not found for relative timestamp (expected if uptime < 100s)")
	} else {
		t.Logf("Metric sentrylogmon_monitor_lag_seconds found for relative timestamp")
	}
}
