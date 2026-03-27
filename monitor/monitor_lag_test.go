package monitor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// NamedMockSource wraps MockSource to provide a custom name
type NamedMockSource struct {
	MockSource
	name string
}

func (s *NamedMockSource) Name() string {
	return s.name
}

func TestMonitorLagMetric(t *testing.T) {
	// 1. Create a source with a timestamp from 10 seconds ago.
	// We use ISO8601 format: 2006-01-02T15:04:05Z07:00
	// Use UTC to avoid timezone issues in testing
	now := time.Now().UTC()
	past := now.Add(-10 * time.Second)
	tsStr := past.Format(time.RFC3339)
	line := fmt.Sprintf("%s Some log message\n", tsStr)

	namedSource := &NamedMockSource{MockSource: MockSource{content: line}, name: "mock_lag_test"}
	detector := &MockDetector{}

	// 2. Create Monitor
	mon, err := New(context.Background(), namedSource, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// 3. Start Monitor
	// Monitor.Start() handles the loop and stops on EOF.
	mon.Start()

	// 4. Check Metrics
	// We need to gather metrics from DefaultGatherer
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	var found bool
	var lagCount uint64
	var lagSum float64

	for _, mf := range mfs {
		if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
			for _, m := range mf.GetMetric() {
				// Check label "source" == "mock_lag_test"
				for _, label := range m.GetLabel() {
					if label.GetName() == "source" && label.GetValue() == "mock_lag_test" {
						found = true
						h := m.GetHistogram()
						lagCount = h.GetSampleCount()
						lagSum = h.GetSampleSum()
					}
				}
			}
		}
	}

	if !found {
		t.Fatalf("Metric sentrylogmon_monitor_lag_seconds not found for source mock_lag_test")
	}

	if lagCount != 1 {
		t.Errorf("Expected count 1, got %d", lagCount)
	}

	// Lag should be around 10 seconds.
	// Allow some margin (e.g., 9.0 to 15.0) - sometimes CI/test environment can be slow
	// or `time.Now` diffs might be slightly off.
	if lagSum < 9.0 || lagSum > 15.0 {
		t.Errorf("Expected lag around 10s, got %f", lagSum)
	}
}
