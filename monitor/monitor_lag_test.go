package monitor

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
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
	// Construct a log line with a timestamp 1 second in the past
	now := time.Now()
	oneSecondAgo := now.Add(-1 * time.Second)
	// Using RFC3339 format which is supported by extractTimestamp
	timestampStr := oneSecondAgo.Format(time.RFC3339)
	line := fmt.Sprintf("%s This is a test log line", timestampStr)

	uniqueSourceName := fmt.Sprintf("mock_lag_test_%d", time.Now().UnixNano())

	source := &NamedMockSource{name: uniqueSourceName, content: line}
	detector := &MockDetector{}

	// Create monitor
	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Run monitor
	mon.Start()

	// Verify metrics
	// We need to gather metrics from the default registry
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	var lagHistogram *io_prometheus_client.MetricFamily
	for _, mf := range mfs {
		if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
			lagHistogram = mf
			break
		}
	}

	if lagHistogram == nil {
		t.Fatal("sentrylogmon_monitor_lag_seconds metric not found")
	}

	// Find the metric for our unique source
	var metric *io_prometheus_client.Metric
	for _, m := range lagHistogram.GetMetric() {
		for _, label := range m.GetLabel() {
			if label.GetName() == "source" && label.GetValue() == uniqueSourceName {
				metric = m
				break
			}
		}
		if metric != nil {
			break
		}
	}

	if metric == nil {
		t.Fatal("Metric for source 'mock' not found")
	}

	if metric.GetHistogram() == nil {
		t.Fatal("Metric is not a histogram")
	}

	count := metric.GetHistogram().GetSampleCount()
	if count < 1 {
		t.Errorf("Expected sample count >= 1, got %d", count)
	}

	sum := metric.GetHistogram().GetSampleSum()

	// Lag should be around 1.0 seconds.
	if sum < 0.5 || sum > 5.0 {
		t.Errorf("Expected lag around 1.0s, got %f", sum)
	}
}
