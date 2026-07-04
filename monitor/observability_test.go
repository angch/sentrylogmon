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

type mockTimestampDetector struct {
	MockDetector
}

func (d *mockTimestampDetector) ExtractTimestamp(line []byte) (float64, string, bool) {
	return 31536000.0, "1971-01-01T00:00:00Z", true
}

func TestMonitorLagMetric(t *testing.T) {
	metrics.MonitorLag.Reset()
	input := "1971-01-01T00:00:00Z line1\n1971-01-01T00:00:00Z line2\n"
	source := &MockSource{content: input}
	detector := &mockTimestampDetector{}
	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true
	mon.Start()

	m := metrics.MonitorLag.With(prometheus.Labels{"source": "mock"})
	if m == nil {
		t.Fatalf("Metric is nil")
	}
	if mMetric, ok := m.(prometheus.Metric); ok {
		var metric dto.Metric
		err = mMetric.Write(&metric)
		if err != nil {
			t.Fatalf("Failed to read metric: %v", err)
		}
		count := metric.GetHistogram().GetSampleCount()
		if count == 0 {
			t.Errorf("Metric sample count is 0, expected it to be updated")
		}
	} else {
		t.Fatalf("Could not cast to prometheus.Metric")
	}
}
