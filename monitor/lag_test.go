package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/metrics"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestMonitorLagMetric(t *testing.T) {
	metrics.MonitorLag.Reset()

	input := "[100.0] Line 1\n"
	source := &MockSource{content: input}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	mon.Start()

	m := metrics.MonitorLag.With(prometheus.Labels{"source": "mock"})

	var metric dto.Metric
	err = m.Write(&metric)
	if err != nil {
		t.Fatalf("Failed to read metric: %v", err)
	}

	val := metric.GetGauge().GetValue()
	now := float64(time.Now().UnixNano()) / 1e9

	if val <= 0 {
		t.Errorf("Metric value should be greater than 0, got %v", val)
	}

	expectedLag := now - 100.0
	if val < expectedLag-5 || val > expectedLag+5 {
		t.Errorf("Metric value seems incorrect. Got %v, expected approx %v", val, expectedLag)
	}
}
