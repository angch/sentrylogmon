package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/metrics"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestLastActivityMetric(t *testing.T) {
	// Reset metrics to ensure clean state
	metrics.LastActivityTimestamp.Reset()

	input := "line1\nline2\n"
	source := &MockSource{content: input}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Run monitor
	mon.Start()

	// Verify metric
	// We need to look up the metric with label source="mock"
	// Note: MockSource.Name() returns "mock"
	m := metrics.LastActivityTimestamp.With(prometheus.Labels{"source": "mock"})

	// Read metric value
	var metric dto.Metric
	err = m.Write(&metric)
	if err != nil {
		t.Fatalf("Failed to read metric: %v", err)
	}

	val := metric.GetGauge().GetValue()
	now := float64(time.Now().Unix())

	// Check if value is close to now (within 5 seconds)
	// If it wasn't updated, it would be 0 or nil (panic above if nil, but GetValue returns float64)
	if val == 0 {
		t.Errorf("Metric value is 0, expected it to be updated")
	}

	if now-val > 5 {
		t.Errorf("Metric value too old. Got %v, expected ~%v", val, now)
	}
	if val > now+1 {
		t.Errorf("Metric value in future. Got %v, expected ~%v", val, now)
	}
}

func TestMonitorLagMetric(t *testing.T) {
	// Reset metrics
	metrics.MonitorLagSeconds.Reset()

	// Create a log line with a timestamp 10 seconds ago.
	// We use ISO8601 format as it is supported by extractTimestamp.
	now := time.Now()
	past := now.Add(-10 * time.Second)
	// Format: 2006-01-02T15:04:05.000Z07:00
	tsStr := past.Format(time.RFC3339)
	input := tsStr + " log line\n"

	source := &MockSource{content: input}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	mon.Start()

	// Verify metric
	h, err := metrics.MonitorLagSeconds.GetMetricWith(prometheus.Labels{"source": "mock"})
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	var metric dto.Metric
	// h is Observer, need to cast to Metric
	if m, ok := h.(prometheus.Metric); ok {
		err = m.Write(&metric)
		if err != nil {
			t.Fatalf("Failed to write metric: %v", err)
		}
	} else {
		t.Fatalf("Metric is not a prometheus.Metric")
	}

	// Check histogram
	hist := metric.GetHistogram()
	if hist.GetSampleCount() != 1 {
		t.Errorf("Expected 1 sample, got %d", hist.GetSampleCount())
	}

	sum := hist.GetSampleSum()
	// Lag should be around 10 seconds.
	// Allow some margin (e.g. 9 to 11) because of execution time
	if sum < 9.0 || sum > 11.0 {
		t.Errorf("Expected lag around 10s, got %v", sum)
	}
}
