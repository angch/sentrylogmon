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

// UniqueMockSource wraps MockSource to provide a unique name
type UniqueMockSource struct {
	content string
	name    string
}

func (s *UniqueMockSource) Name() string { return s.name }
func (s *UniqueMockSource) Stream() (io.Reader, error) {
	return strings.NewReader(s.content), nil
}
func (s *UniqueMockSource) Close() error { return nil }

func TestMonitorLag(t *testing.T) {
	// Create a mock source with a unique name to isolate metrics
	sourceName := "mock_lag_test"

	// Create a log line with timestamp 60 seconds in the past
	// Format: ISO8601 2006-01-02T15:04:05Z
	pastTime := time.Now().Add(-60 * time.Second).UTC()
	tsStr := pastTime.Format("2006-01-02T15:04:05Z")
	input := fmt.Sprintf("%s Test Log Line", tsStr)

	source := &UniqueMockSource{content: input, name: sourceName}
	detector := &MockDetector{} // From monitor_test.go

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	go mon.Start()

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Verify metrics
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	var found bool
	var count uint64
	var sum float64

	for _, mf := range mfs {
		if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
			for _, m := range mf.GetMetric() {
				// Check label
				for _, label := range m.GetLabel() {
					if label.GetName() == "source" && label.GetValue() == sourceName {
						hist := m.GetHistogram()
						count = hist.GetSampleCount()
						sum = hist.GetSampleSum()
						found = true
						break
					}
				}
			}
		}
	}

	if !found {
		t.Errorf("Metric sentrylogmon_monitor_lag_seconds not found for source %s", sourceName)
	}

	if count != 1 {
		t.Errorf("Expected sample count 1, got %d", count)
	}

	// Lag should be around 60 seconds. Allow +/- 2 seconds.
	if sum < 58.0 || sum > 62.0 {
		t.Errorf("Expected lag around 60s, got %f", sum)
	}
}
