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

// MockTimestampDetector implements detectors.Detector and detectors.TimestampExtractor
type MockTimestampDetector struct{}

func (d *MockTimestampDetector) Detect(line []byte) bool { return true }
func (d *MockTimestampDetector) ExtractTimestamp(line []byte) (float64, string, bool) {
	now := float64(time.Now().UnixNano()) / 1e9
	return now - 1.5, "test-time", true // 1.5 seconds ago
}

func TestMonitorLagMetric(t *testing.T) {
	metrics.MonitorLag.Reset()

	input := "line1\n"
	source := &MockSource{content: input}
	detector := &MockTimestampDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	mon.Start()

	m, err := metrics.MonitorLag.GetMetricWith(prometheus.Labels{"source": "mock"})
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	var metric dto.Metric
	err = m.(prometheus.Metric).Write(&metric)
	if err != nil {
		t.Fatalf("Failed to read metric: %v", err)
	}

	if metric.GetHistogram() == nil {
		t.Fatalf("Histogram is nil")
	}

	count := metric.GetHistogram().GetSampleCount()
	if count != 1 {
		t.Errorf("Expected 1 observation, got %d", count)
	}

	sum := metric.GetHistogram().GetSampleSum()
	if sum < 1.4 || sum > 2.0 {
		t.Errorf("Expected sum to be around 1.5, got %f", sum)
	}
}
