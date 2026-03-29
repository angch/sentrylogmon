package monitor

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
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
	// Create a unique source name to avoid collisions in the global registry
	sourceName := "mock_lag_test"

	// Generate a log line with a timestamp 10 seconds ago
	// Format: [timestamp] message (Dmesg format is supported and easy)
	now := time.Now()
	lagSeconds := 10.0
	pastTime := float64(now.UnixNano())/1e9 - lagSeconds

	input := fmt.Sprintf("[%f] Test Log Line", pastTime)
	source := &NamedMockSource{name: sourceName, content: input}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	go mon.Start()

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Verify metric
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	var metricFamily *dto.MetricFamily
	for _, mf := range mfs {
		if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
			metricFamily = mf
			break
		}
	}

	if metricFamily == nil {
		t.Fatalf("Metric sentrylogmon_monitor_lag_seconds not found")
	}

	var metric *dto.Metric
	for _, m := range metricFamily.Metric {
		for _, label := range m.Label {
			if label.GetName() == "source" && label.GetValue() == sourceName {
				metric = m
				break
			}
		}
	}

	if metric == nil {
		t.Fatalf("Metric for source %s not found", sourceName)
	}

	histogram := metric.GetHistogram()
	if histogram.GetSampleCount() != 1 {
		t.Errorf("Expected sample count 1, got %d", histogram.GetSampleCount())
	}

	sum := histogram.GetSampleSum()
	// Should be close to 10.0
	// Allow 0.5s margin for execution time
	if sum < 9.5 || sum > 10.5 {
		t.Errorf("Expected lag around 10.0, got %f", sum)
	}
}
