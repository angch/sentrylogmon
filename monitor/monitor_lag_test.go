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
	// Create a log line with a timestamp 10 seconds in the past
	now := time.Now()
	past := now.Add(-10 * time.Second)
	// Format: [1234567890.123456] ...
	// Dmesg timestamp is just float seconds.
	tsStr := fmt.Sprintf("%.6f", float64(past.UnixMicro())/1e6)
	input := fmt.Sprintf("[%s] Some log message\n", tsStr)

	uniqueName := fmt.Sprintf("test_lag_%d", time.Now().UnixNano())
	source := &UniqueMockSource{content: input, name: uniqueName}
	detector := &MockDetector{} // Detects everything

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Run monitor
	mon.Start()

	// Check metrics
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	var lagMetric *dto.MetricFamily
	for _, mf := range mfs {
		if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
			lagMetric = mf
			break
		}
	}

	if lagMetric == nil {
		t.Fatalf("Metric sentrylogmon_monitor_lag_seconds not found")
	}

	// Find the metric for our source
	var metric *dto.Metric
	for _, m := range lagMetric.Metric {
		found := false
		for _, label := range m.Label {
			if label.GetName() == "source" && label.GetValue() == uniqueName {
				found = true
				break
			}
		}
		if found {
			metric = m
			break
		}
	}

	if metric == nil {
		t.Fatalf("Metric for source '%s' not found. Available: %v", uniqueName, lagMetric.Metric)
	}

	hist := metric.Histogram
	if hist == nil {
		t.Fatalf("Metric is not a histogram")
	}

	if hist.GetSampleCount() == 0 {
		t.Errorf("Expected sample count > 0, got 0")
	}

	if hist.GetSampleSum() == 0 {
		t.Errorf("Expected sample sum > 0, got 0")
	}

	// Lag should be around 10 seconds
	// We can't be precise because of execution time, but it should be > 9 and < 20
	avgLag := hist.GetSampleSum() / float64(hist.GetSampleCount())
	if avgLag < 9.0 || avgLag > 20.0 {
		t.Errorf("Expected lag around 10s, got %f", avgLag)
	}
}
