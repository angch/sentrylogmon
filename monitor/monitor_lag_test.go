package monitor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v3/host"
)

// uniqueMockSource wraps MockSource to provide a distinct name for Prometheus metrics.
type uniqueMockSource struct {
	*MockSource
	name string
}

func (s *uniqueMockSource) Name() string { return s.name }

func getGaugeValue(metricName string, sourceName string) (float64, error) {
	metrics, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return 0, err
	}

	for _, m := range metrics {
		if m.GetName() == metricName {
			for _, metric := range m.GetMetric() {
				for _, label := range metric.GetLabel() {
					if label.GetName() == "source" && label.GetValue() == sourceName {
						if metric.Gauge != nil && metric.Gauge.Value != nil {
							return *metric.Gauge.Value, nil
						}
					}
				}
			}
		}
	}
	return 0, fmt.Errorf("metric not found")
}

func TestMonitorLagAbsolute(t *testing.T) {
	// Setup
	err := sentry.Init(sentry.ClientOptions{
		Transport: &MockTransport{},
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	sourceName := "test_lag_absolute"

	// Create a log line with a timestamp 10 seconds ago
	now := time.Now()
	logTime := now.Add(-10 * time.Second)
	// ISO8601 (RFC3339) timestamp format
	tsStr := logTime.Format(time.RFC3339)
	input := fmt.Sprintf("%s Test error message\n", tsStr)

	baseSource := &MockSource{content: input}
	source := &uniqueMockSource{MockSource: baseSource, name: sourceName}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Run
	mon.Start()

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verification
	lag, err := getGaugeValue("sentrylogmon_monitor_lag_seconds", sourceName)
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	// The lag should be approximately 10 seconds.
	// Allow a small margin of error (e.g., 1 second) for test execution time.
	if lag < 9.0 || lag > 11.0 {
		t.Errorf("Expected lag around 10.0, got %f", lag)
	}
}

func TestMonitorLagRelative(t *testing.T) {
	// Setup
	err := sentry.Init(sentry.ClientOptions{
		Transport: &MockTransport{},
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	sourceName := "test_lag_relative"

	uptime, err := host.Uptime()
	if err != nil {
		t.Skip("host.Uptime() not available, skipping relative lag test")
	}

	// Simulate a dmesg log line from 5 seconds ago
	logTime := float64(uptime) - 5.0
	input := fmt.Sprintf("[%f] Test kernel error\n", logTime)

	baseSource := &MockSource{content: input}
	source := &uniqueMockSource{MockSource: baseSource, name: sourceName}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Run
	mon.Start()

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verification
	lag, err := getGaugeValue("sentrylogmon_monitor_lag_seconds", sourceName)
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	// The lag should be approximately 5 seconds.
	// Allow a small margin of error (e.g., 1 second) for test execution time.
	if lag < 4.0 || lag > 6.0 {
		t.Errorf("Expected lag around 5.0, got %f", lag)
	}
}
