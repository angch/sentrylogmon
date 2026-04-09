package monitor

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/metrics"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// uniqueMockSource is a wrapper around MockSource with a configurable name
// to avoid Prometheus metric collisions during parallel testing.
type uniqueMockSource struct {
	content string
	name    string
}

func (s *uniqueMockSource) Name() string { return s.name }
func (s *uniqueMockSource) Stream() (io.Reader, error) {
	return strings.NewReader(s.content), nil
}
func (s *uniqueMockSource) Close() error { return nil }

func TestMonitorLagCalculation(t *testing.T) {
	metrics.MonitorLagSeconds.Reset()
	sourceName := "mock_lag_test"

	now := time.Now()
	// Create a log entry from exactly 5 seconds ago
	logTime := now.Add(-5 * time.Second)
	// Syslog format: "Oct 11 22:14:15"
	logLine := fmt.Sprintf("%s server process: Test error message\n", logTime.Format("Jan _2 15:04:05"))

	source := &uniqueMockSource{content: logLine, name: sourceName}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Run monitor
	mon.Start()

	m := metrics.MonitorLagSeconds.With(prometheus.Labels{"source": sourceName})

	var metric dto.Metric
	err = m.Write(&metric)
	if err != nil {
		t.Fatalf("Failed to read metric: %v", err)
	}

	if metric.Gauge == nil {
		t.Fatalf("Metric is not a gauge or not set")
	}

	val := metric.GetGauge().GetValue()

	// The lag should be approximately 5 seconds
	if val < 4.0 || val > 7.0 {
		t.Errorf("Metric value out of expected range. Got %v, expected ~5.0", val)
	}
}

func TestMonitorLagNegative(t *testing.T) {
	metrics.MonitorLagSeconds.Reset()
	sourceName := "mock_lag_test_negative"

	now := time.Now()
	// Create a log entry from 5 seconds in the future
	logTime := now.Add(5 * time.Second)
	logLine := fmt.Sprintf("%s server process: Test error message\n", logTime.Format("Jan _2 15:04:05"))

	source := &uniqueMockSource{content: logLine, name: sourceName}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Run monitor
	mon.Start()

	m := metrics.MonitorLagSeconds.With(prometheus.Labels{"source": sourceName})

	var metric dto.Metric
	err = m.Write(&metric)
	if err != nil {
		t.Fatalf("Failed to read metric: %v", err)
	}

	// For an uninitialized Gauge or a Gauge that wasn't updated with a negative value,
	// the default value is 0. If it recorded the negative value, it would be < 0.
	val := metric.GetGauge().GetValue()
	if val != 0 {
		t.Errorf("Expected negative lag to be ignored (metric value 0), but got %v", val)
	}
}
