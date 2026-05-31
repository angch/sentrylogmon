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
	metrics.MonitorLag.Reset()

	// timestamp ~100 seconds ago
	now := float64(time.Now().UnixNano())/1e9
	ts := now - 100.0
	input := "[100.0] Line 1\n"
	source := &MockSource{content: input}

	// Create a detector that implements TimestampExtractor
	// so it can return our specific timestamp
	detector := &mockTimestampDetector{ts: ts}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	mon.Start()

	o, err := metrics.MonitorLag.GetMetricWith(prometheus.Labels{"source": "mock"})
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}
	m := o.(prometheus.Metric)

	var metric dto.Metric
	err = m.Write(&metric)
	if err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	hist := metric.GetHistogram()
	if hist.GetSampleCount() != 1 {
		t.Errorf("Expected 1 sample, got %v", hist.GetSampleCount())
	}

	// sum of lag should be roughly 100 seconds
	if hist.GetSampleSum() < 99 || hist.GetSampleSum() > 102 {
		t.Errorf("Expected sum to be around 100, got %v", hist.GetSampleSum())
	}
}

type mockTimestampDetector struct {
	ts float64
}

func (d *mockTimestampDetector) Detect(line []byte) bool { return true }
func (d *mockTimestampDetector) ExtractTimestamp(line []byte) (float64, string, bool) {
	return d.ts, "mock_ts", true
}
