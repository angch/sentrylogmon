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

	input := "[12345.67890] Error occurred\n"
	source := &MockSource{content: input}

	// Create a detector that returns true for the input and also extracts a mock timestamp
	detector := &MockLagDetector{
		detectReturns: true,
		timestamp:     float64(time.Now().UnixNano())/1e9 - 2.5, // simulate 2.5 seconds lag
	}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	mon.Start()

	// Use GetMetricWith to fetch the metric from the vector
	observer, err := metrics.MonitorLag.GetMetricWith(prometheus.Labels{"source": "mock"})
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}
	m := observer.(prometheus.Metric)

	var metric dto.Metric
	err = m.Write(&metric)
	if err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	histogram := metric.GetHistogram()
	if histogram.GetSampleCount() != 1 {
		t.Errorf("Expected 1 sample, got %v", histogram.GetSampleCount())
	}

	sum := histogram.GetSampleSum()
	if sum < 2.4 || sum > 2.6 {
		t.Errorf("Expected lag around 2.5s, got %v", sum)
	}
}

type MockLagDetector struct {
	detectReturns bool
	timestamp     float64
}

func (d *MockLagDetector) Detect(line []byte) bool {
	return d.detectReturns
}

func (d *MockLagDetector) Name() string {
	return "MockLagDetector"
}

func (d *MockLagDetector) ExtractTimestamp(line []byte) (float64, string, bool) {
	return d.timestamp, "mock_ts", true
}
