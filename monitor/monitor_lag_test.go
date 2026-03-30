package monitor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_model/go"
	"github.com/shirou/gopsutil/v3/host"
)

// uniqueMockSource wraps MockSource to provide a unique name for Prometheus registry isolation
type uniqueMockSource struct {
	MockSource
	name string
}

func (s *uniqueMockSource) Name() string { return s.name }

func getMetricValue(metricFamily string, labelName, labelValue string) (float64, error) {
	gatherer := prometheus.DefaultGatherer
	mfs, err := gatherer.Gather()
	if err != nil {
		return 0, err
	}

	for _, mf := range mfs {
		if mf.GetName() == metricFamily {
			for _, m := range mf.GetMetric() {
				for _, lp := range m.GetLabel() {
					if lp.GetName() == labelName && lp.GetValue() == labelValue {
						if mf.GetType() == io_prometheus_client.MetricType_GAUGE {
							return m.GetGauge().GetValue(), nil
						}
					}
				}
			}
		}
	}
	return 0, fmt.Errorf("metric %s not found for %s=%s", metricFamily, labelName, labelValue)
}

func TestMonitorLagAbsolute(t *testing.T) {
	sourceName := "mock_lag_absolute_test"

	// Create a log line with a timestamp exactly 5 seconds ago
	now := time.Now()
	past := now.Add(-5 * time.Second)
	// Using RFC3339 format which ISO8601 parser can handle
	tsStr := past.Format(time.RFC3339)
	input := fmt.Sprintf("%s Test message\n", tsStr)

	source := &uniqueMockSource{
		MockSource: MockSource{content: input},
		name:       sourceName,
	}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Explicitly set bootTime to 0 to ensure absolute path is taken
	mon.bootTime = 0

	// Start monitor and wait for processing
	mon.Start()

	// Verify the metric
	lag, err := getMetricValue("sentrylogmon_monitor_lag_seconds", "source", sourceName)
	if err != nil {
		t.Fatalf("Failed to get lag metric: %v", err)
	}

	// The lag should be approximately 5 seconds.
	// Allow a small delta for processing time.
	if lag < 4.0 || lag > 6.0 {
		t.Errorf("Expected lag around 5s, got %fs", lag)
	}
}

func TestMonitorLagRelative(t *testing.T) {
	sourceName := "mock_lag_relative_test"

	// Get current uptime to construct a realistic dmesg timestamp
	uptime, err := host.Uptime()
	if err != nil {
		t.Skipf("Skipping test because host.Uptime failed: %v", err)
	}

	// Create a dmesg log line with an uptime 5 seconds ago
	pastUptime := float64(uptime) - 5.0
	if pastUptime < 0 {
		pastUptime = 0.1
	}

	input := fmt.Sprintf("[ %f] Test dmesg message\n", pastUptime)

	source := &uniqueMockSource{
		MockSource: MockSource{content: input},
		name:       sourceName,
	}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Start monitor and wait for processing
	mon.Start()

	// Verify the metric
	lag, err := getMetricValue("sentrylogmon_monitor_lag_seconds", "source", sourceName)
	if err != nil {
		t.Fatalf("Failed to get lag metric: %v", err)
	}

	// The lag should be approximately 5 seconds.
	// Allow a small delta for processing time.
	if lag < 4.0 || lag > 6.0 {
		t.Errorf("Expected lag around 5s, got %fs", lag)
	}
}
