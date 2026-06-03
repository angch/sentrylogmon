package monitor

import (
	"context"
	"testing"

	"github.com/angch/sentrylogmon/metrics"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

type MockLagDetector struct {
	extractTs   float64
	extractTsOk bool
}

func (d *MockLagDetector) Detect(line []byte) bool { return true }
func (d *MockLagDetector) ExtractTimestamp(line []byte) (float64, string, bool) {
	return d.extractTs, "", d.extractTsOk
}

func TestMonitorLagMetric(t *testing.T) {
	metrics.MonitorLag.Reset()

	input := "2023-10-27T10:00:00Z error log\n"
	source := &MockSource{content: input}
	detector := &MockLagDetector{
		extractTs:   float64(1698400800),
		extractTsOk: true,
	}

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
		t.Fatalf("Failed to read metric: %v", err)
	}

	if metric.Histogram == nil {
		t.Fatalf("Histogram is nil, expected values")
	}
	if metric.Histogram.GetSampleCount() == 0 {
		t.Errorf("Expected sample count > 0")
	}
}
