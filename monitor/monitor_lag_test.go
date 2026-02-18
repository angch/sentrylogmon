package monitor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
)

type NamedMockSource struct {
	MockSource
	name string
}

func (s *NamedMockSource) Name() string { return s.name }

func TestMonitorLagMetric(t *testing.T) {
	// Create a log line with a timestamp 10 seconds ago
	now := time.Now()
	past := now.Add(-10 * time.Second)
	// Format as ISO8601/RFC3339 which ExtractTimestamp supports
	timestampStr := past.Format(time.RFC3339)
	input := fmt.Sprintf("%s Test Log Line\n", timestampStr)

	sourceName := fmt.Sprintf("test_lag_source_%d", time.Now().UnixNano())
	source := &NamedMockSource{
		MockSource: MockSource{content: input},
		name:       sourceName,
	}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Run monitor synchronously as StopOnEOF is true
	mon.Start()

	// Verify metric
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	var lagMetric *io_prometheus_client.MetricFamily
	for _, mf := range mfs {
		if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
			lagMetric = mf
			break
		}
	}

	if lagMetric == nil {
		t.Fatal("sentrylogmon_monitor_lag_seconds metric not found")
	}

	var metric *io_prometheus_client.Metric
	for _, m := range lagMetric.Metric {
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

	if metric.Histogram == nil {
		t.Fatal("Metric is not a histogram")
	}

	count := metric.Histogram.GetSampleCount()
	if count != 1 {
		t.Errorf("Expected sample count 1, got %d", count)
	}

	sum := metric.Histogram.GetSampleSum()
	// Lag should be around 10 seconds. Allow some margin (e.g., 9.5 to 11.5)
	if sum < 9.5 || sum > 11.5 {
		t.Errorf("Expected lag sum around 10.0, got %f", sum)
	}
}
