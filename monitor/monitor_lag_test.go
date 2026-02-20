package monitor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/metrics"
	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
)

func TestMonitorLag(t *testing.T) {
	// 1. Create a log line with a timestamp 10 seconds in the past
	lagDuration := 10 * time.Second
	logTime := time.Now().Add(-lagDuration)
	// Format: ISO8601 which is supported by extractTimestamp
	tsStr := logTime.Format(time.RFC3339)
	line := fmt.Sprintf("%s This is a delayed log line", tsStr)

	source := &MockSource{content: line}
	detector := &MockDetector{}

	// 2. Reset metrics to ensure clean state BEFORE creating monitor
	metrics.MonitorLagSeconds.Reset()

	// 3. Initialize Monitor
	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// 4. Run Monitor
	// The first line processed should trigger the lag check because lastMetricUpdateTime starts at 0.
	mon.Start()

	// 5. Verify Metric
	// We need to gather metrics from the default registry (where MonitorLagSeconds is registered)
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	var metricFamily *io_prometheus_client.MetricFamily
	for _, mf := range mfs {
		if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
			metricFamily = mf
			break
		}
	}

	if metricFamily == nil {
		t.Fatal("Metric sentrylogmon_monitor_lag_seconds not found")
	}

	// Find the metric for our source "mock"
	var metric *io_prometheus_client.Metric
	for _, m := range metricFamily.GetMetric() {
		for _, label := range m.GetLabel() {
			if label.GetName() == "source" && label.GetValue() == "mock" {
				metric = m
				break
			}
		}
	}

	if metric == nil {
		t.Fatal("Metric for source 'mock' not found")
	}

	// Check histogram
	hist := metric.GetHistogram()
	if hist.GetSampleCount() == 0 {
		t.Error("Histogram sample count is 0, expected at least 1")
	}

	sum := hist.GetSampleSum()
	// Lag should be around 10 seconds. Allow some buffer (9.0 to 15.0)
	if sum < 9.0 || sum > 15.0 {
		t.Errorf("Expected lag around 10s, got sum %f (count %d)", sum, hist.GetSampleCount())
	}
}
