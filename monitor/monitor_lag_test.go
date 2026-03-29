package monitor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus"
)

// NamedMockSource wraps MockSource to provide a custom name
type NamedMockSource struct {
	MockSource
	name string
}

func (s *NamedMockSource) Name() string { return s.name }

func TestMonitorLagMetric(t *testing.T) {
	// Initialize Sentry Mock (needed for New)
	transport := &MockTransport{}
	_ = sentry.Init(sentry.ClientOptions{
		Transport: transport,
	})

	// Create a log line with a timestamp 1 hour ago
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	// Format: 2006-01-02T15:04:05Z (RFC3339)
	tsStr := oneHourAgo.Format(time.RFC3339)
	input := fmt.Sprintf("%s Line 1\n", tsStr)

	sourceName := "mock_lag_test"
	baseSource := &MockSource{content: input}

	detector := &MockDetector{}

	// We need to use a custom source name to isolate the metric
	namedSource := &NamedMockSource{MockSource: *baseSource, name: sourceName}

	mon, err := New(context.Background(), namedSource, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	go mon.Start()

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Verify Metric
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	var found bool
	for _, mf := range mfs {
		if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
			for _, m := range mf.GetMetric() {
				// Check label
				for _, label := range m.GetLabel() {
					if label.GetName() == "source" && label.GetValue() == sourceName {
						found = true
						// Check if count > 0
						if m.GetHistogram().GetSampleCount() != 1 {
							t.Errorf("Expected sample count 1, got %d", m.GetHistogram().GetSampleCount())
						}
						// Check sum. Should be approx 3600
						sum := m.GetHistogram().GetSampleSum()
						if sum < 3500 || sum > 3700 { // Allow some buffer for execution time
							t.Errorf("Expected lag around 3600s, got %f", sum)
						}
					}
				}
			}
		}
	}

	if !found {
		t.Errorf("Metric sentrylogmon_monitor_lag_seconds not found for source %s", sourceName)
	}
}
