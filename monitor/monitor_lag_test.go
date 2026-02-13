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

func TestMonitorLagMetric(t *testing.T) {
	// Reset metrics
	metrics.MonitorLagSeconds.Reset()

	// Create a log line with a timestamp 10 seconds ago
	now := time.Now()
	tenSecondsAgo := now.Add(-10 * time.Second)
	// Format as ISO8601 which is supported by default extractTimestamp
	timestampStr := tenSecondsAgo.Format(time.RFC3339)
	line := fmt.Sprintf("%s Some log message", timestampStr)

	source := &MockSource{content: line + "\n"} // Name() returns "mock"
	// Use a detector that returns true for everything but doesn't implement TimestampExtractor
	// so it falls back to extractTimestamp function which supports RFC3339
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

	// Cast Observer to Metric to access Write
	metricObserver := m
	metric, ok := metricObserver.(prometheus.Metric)
	if !ok {
		t.Fatalf("Failed to cast Observer to Metric")
	}

	var metricDto dto.Metric
	err = metric.Write(&metricDto)
	if err != nil {
		t.Fatalf("Failed to read metric: %v", err)
	}

	hist := metricDto.GetHistogram()
	if hist.GetSampleCount() != 1 {
		t.Errorf("Expected 1 sample, got %d", hist.GetSampleCount())
	}

	sum := hist.GetSampleSum()
	// Should be around 10 seconds.
	// Since we are mocking, the processing time (time.Now() inside processMatch)
	// will be slightly after our `now` variable.
	// So lag = processMatchNow - tenSecondsAgo
	// processMatchNow >= now
	// So lag >= 10s.

	// Allow small margin
	if sum < 10.0 || sum > 15.0 {
		t.Errorf("Expected lag around 10s, got %f", sum)
	}
}
