package monitor

import (
	"context"
	"fmt"
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
	// Reset metrics to ensure clean state
	metrics.MonitorLagSeconds.Reset()

	// Use a dmesg-style timestamp from ~1.5 second ago
	nowFloat := float64(time.Now().UnixNano()) / 1e9
	ts := nowFloat - 1.5
	// Dmesg format is [seconds.microseconds]
	input := fmt.Sprintf("[%f] mock error\n", ts)

	// Create mock source and detector that will match
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
	m := metrics.MonitorLagSeconds.With(prometheus.Labels{"source": "mock"})

	var metric dto.Metric
	// We need to type assert the metric since it's an Observer
	if hist, ok := m.(prometheus.Metric); ok {
		err = hist.Write(&metric)
		if err != nil {
			t.Fatalf("Failed to read metric: %v", err)
		}

		val := metric.GetHistogram().GetSampleCount()
		if val == 0 {
			t.Errorf("Metric sample count is 0, expected it to be updated")
		}

		sum := metric.GetHistogram().GetSampleSum()
		if sum < 1.0 || sum > 2.0 {
			t.Errorf("Metric sum unexpected. Got %v, expected ~1.5", sum)
		}
	} else {
		t.Fatalf("Metric is not of expected type")
	}
}
