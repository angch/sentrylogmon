package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/detectors"
	"github.com/angch/sentrylogmon/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func TestMonitorLag(t *testing.T) {
	// Reset metrics
	metrics.MonitorLag.Reset()

	// Create a detector that matches everything
	detector, _ := detectors.NewGenericDetector(".*")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts := Options{
		Verbose: true,
	}

	// Calculate a timestamp 5 seconds ago
	now := time.Now()
	past := now.Add(-5 * time.Second)
	// Format as RFC3339 which is ISO8601 compatible
	tsStr := past.Format(time.RFC3339)
	line := tsStr + " error occurred"

	source := &MockSource{content: line}
	monitor, err := New(ctx, source, detector, nil, opts)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	// Directly call processMatch to avoid async issues with Start()
	monitor.processMatch([]byte(line))

	// Verify metric
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	found := false
	for _, mf := range metricFamilies {
		if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
			for _, m := range mf.GetMetric() {
				foundLabel := false
				for _, label := range m.GetLabel() {
					if label.GetName() == "source" && label.GetValue() == "mock" {
						foundLabel = true
						break
					}
				}
				if foundLabel {
					found = true
					hist := m.GetHistogram()
					if hist.GetSampleCount() != 1 {
						t.Errorf("Expected sample count 1, got %d", hist.GetSampleCount())
					}
					sum := hist.GetSampleSum()
					// Should be around 5.0
					// Allow some jitter
					if sum < 4.5 || sum > 5.5 {
						t.Errorf("Expected lag around 5.0, got %f", sum)
					}
				}
			}
		}
	}

	if !found {
		t.Error("Metric sentrylogmon_monitor_lag_seconds not found for source 'mock'")
	}
}
