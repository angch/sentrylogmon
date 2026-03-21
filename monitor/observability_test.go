package monitor

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/detectors"
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

// MockSourceLag is needed to test source with different label name
type MockSourceLag struct {
	content string
}

func (s *MockSourceLag) Name() string { return "mock_lag" }
func (s *MockSourceLag) Stream() (io.Reader, error) {
	return strings.NewReader(s.content), nil
}
func (s *MockSourceLag) Close() error { return nil }

func TestMonitorLagMetric(t *testing.T) {
	// Reset metrics to ensure clean state
	metrics.MonitorLagSeconds.Reset()

	// Setup 100 seconds ago absolute timestamp using ISO8601 format
	now := time.Now()
	timestamp := now.Add(-100 * time.Second)

	// Create a log entry with an ISO8601 absolute timestamp
	input := fmt.Sprintf("%s ERROR log line\n", timestamp.Format(time.RFC3339))
	source := &MockSourceLag{content: input}

	detector, _ := detectors.NewGenericDetector("ERROR")

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Run monitor
	mon.Start()

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	m := metrics.MonitorLagSeconds.With(prometheus.Labels{"source": "mock_lag"})
	metric, ok := m.(prometheus.Metric)
	if !ok {
		t.Fatalf("Failed to cast to prometheus.Metric")
	}

	var dtoMetric dto.Metric
	err = metric.Write(&dtoMetric)
	if err != nil {
		t.Fatalf("Failed to read metric: %v", err)
	}

	if dtoMetric.Histogram == nil {
		t.Fatalf("Histogram is nil")
	}

	if *dtoMetric.Histogram.SampleCount == 0 {
		t.Errorf("SampleCount is 0, expected it to be updated")
	}
}
