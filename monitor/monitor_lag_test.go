package monitor

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

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
	// Setup
	now := time.Now()
	// Create a log line with a timestamp 10 seconds ago
	past := now.Add(-10 * time.Second)
	// Using ISO8601 format: 2006-01-02T15:04:05.000Z
	tsStr := past.UTC().Format("2006-01-02T15:04:05.000Z")
	input := fmt.Sprintf("%s Test log line", tsStr)

	sourceName := "mock_lag_test"
	source := &NamedMockSource{name: sourceName, content: input}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Run monitor
	mon.Start()

	// Verify metrics
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	found := false
	for _, mf := range mfs {
		if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
			for _, m := range mf.GetMetric() {
				// Check label
				for _, label := range m.GetLabel() {
					if label.GetName() == "source" && label.GetValue() == sourceName {
						found = true
						hist := m.GetHistogram()
						if hist.GetSampleCount() != 1 {
							t.Errorf("Expected sample count 1, got %d", hist.GetSampleCount())
						}
						// Check sum (should be around 10.0)
						// Allow some jitter (e.g. +/- 2s due to test execution time)
						if hist.GetSampleSum() < 8.0 || hist.GetSampleSum() > 12.0 {
							t.Errorf("Expected lag around 10s, got %f", hist.GetSampleSum())
						}
					}
				}
			}
		}
	}

	if !found {
		t.Errorf("Metric sentrylogmon_monitor_lag_seconds not found for source=%s", sourceName)
	}
}
