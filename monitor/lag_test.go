package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/metrics"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// MockTimestampExtractor implements detectors.Detector and detectors.TimestampExtractor
type MockTimestampExtractor struct {
	MockDetector
	ts float64
}

func (d *MockTimestampExtractor) ExtractTimestamp(line []byte) (float64, string, bool) {
	return d.ts, "", true
}

func TestMonitorLagMetric(t *testing.T) {
	metrics.MonitorLag.Reset()

	input := "line1\n"
	source := &MockSource{content: input}

	// Set timestamp to exactly 2 seconds ago
	now := float64(time.Now().UnixNano()) / 1e9
	detector := &MockTimestampExtractor{ts: now - 2.0}

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
	om := m.(prometheus.Metric)
	err = om.Write(&metric)
	if err != nil {
		t.Fatalf("Failed to read metric: %v", err)
	}

	hist := metric.GetHistogram()

	if hist.GetSampleCount() != 1 {
		t.Errorf("Expected 1 sample, got %v", hist.GetSampleCount())
	}

	sum := hist.GetSampleSum()
	// Should be around 2.0 (allow small variance for processing time)
	if sum < 1.9 || sum > 2.5 {
		t.Errorf("Expected sum ~2.0, got %v", sum)
	}
}
