package monitor

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/detectors"
	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus"
)

// NamedMockSource implements sources.LogSource with a custom name
type NamedMockSource struct {
	name    string
	content string
}

func (s *NamedMockSource) Name() string { return s.name }
func (s *NamedMockSource) Stream() (io.Reader, error) {
	return strings.NewReader(s.content), nil
}
func (s *NamedMockSource) Close() error { return nil }

func TestMonitorLagMetric(t *testing.T) {
	// Initialize Sentry with mock transport to avoid errors
	err := sentry.Init(sentry.ClientOptions{
		Transport: &MockTransport{},
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	// Create a log line with a timestamp 5 seconds ago
	// Using ISO8601 for easier absolute time control.

	now := time.Now()
	past := now.Add(-5 * time.Second)
	// Format: 2006-01-02T15:04:05.000Z07:00
	tsStr := past.Format("2006-01-02T15:04:05.000Z07:00")
	line := fmt.Sprintf("%s Test Message", tsStr)

	sourceName := "test_lag_source"
	source := &NamedMockSource{name: sourceName, content: line}

	// Use GenericDetector to ensure it matches
	detector, _ := detectors.NewGenericDetector("Test Message")

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Run monitor
	mon.Start()

	// Check metrics
	gatherer := prometheus.DefaultGatherer
	mfs, err := gatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	var found bool
	for _, mf := range mfs {
		if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
			for _, m := range mf.GetMetric() {
				// Check labels
				var hasSource bool
				for _, label := range m.GetLabel() {
					if label.GetName() == "source" && label.GetValue() == sourceName {
						hasSource = true
						break
					}
				}
				if !hasSource {
					continue
				}

				found = true
				hist := m.GetHistogram()
				if hist.GetSampleCount() != 1 {
					t.Errorf("Expected 1 sample, got %d", hist.GetSampleCount())
				}

				// Verify sum (approx lag)
				// Lag should be around 5.0 seconds.
				// Since execution takes time, it will be slightly more than 5.0.
				// Let's say between 4.5 and 10.0 to be safe.
				sum := hist.GetSampleSum()
				if sum < 4.5 || sum > 10.0 {
					t.Errorf("Expected lag around 5.0s, got %f", sum)
				}
			}
		}
	}

	if !found {
		t.Error("Metric sentrylogmon_monitor_lag_seconds not found for source")
	}
}
