package monitor

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v3/host"
)

type UniqueMockSource struct {
	MockSource
	name string
}

func (s *UniqueMockSource) Name() string { return s.name }

type MockTimestampDetector struct {
	Timestamp float64
}

func (d *MockTimestampDetector) Detect(line []byte) bool { return true }
func (d *MockTimestampDetector) ExtractTimestamp(line []byte) (float64, string, bool) {
	return d.Timestamp, "", true
}

func getMetricValue(metricName string, labelName string, labelValue string) (float64, bool) {
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return 0, false
	}
	for _, mf := range mfs {
		if mf.GetName() == metricName {
			for _, m := range mf.GetMetric() {
				for _, label := range m.GetLabel() {
					if label.GetName() == labelName && label.GetValue() == labelValue {
						return m.GetGauge().GetValue(), true
					}
				}
			}
		}
	}
	return 0, false
}

func TestMonitorLagAbsolute(t *testing.T) {
	sourceName := "test_lag_absolute"
	input := "some log line"
	source := &UniqueMockSource{MockSource: MockSource{content: input}, name: sourceName}

	// 10 seconds ago
	// Use float64(time.Now().Unix()) to match precision of input
	now := float64(time.Now().UnixNano()) / 1e9
	ts := now - 10.0
	detector := &MockTimestampDetector{Timestamp: ts}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	mon.Start()

	val, found := getMetricValue("sentrylogmon_monitor_lag_seconds", "source", sourceName)
	if !found {
		t.Fatalf("Metric sentrylogmon_monitor_lag_seconds not found for source %s", sourceName)
	}

	// Should be around 10 seconds.
	// Allow some margin for execution time.
	if math.Abs(val-10.0) > 1.0 {
		t.Errorf("Expected lag around 10.0, got %f", val)
	}
}

func TestMonitorLagRelative(t *testing.T) {
	sourceName := "test_lag_relative"
	input := "some log line"
	source := &UniqueMockSource{MockSource: MockSource{content: input}, name: sourceName}

	uptime, err := host.Uptime()
	if err != nil {
		t.Skip("Skipping relative lag test: failed to get uptime")
	}

	// Set timestamp to 10 seconds ago relative to boot
	// If uptime is less than 10s, this test might be flaky or fail, but that's rare for CI envs.
	ts := float64(uptime) - 10.0
	if ts < 0 {
		ts = 0
	}

	detector := &MockTimestampDetector{Timestamp: ts}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	mon.Start()

	val, found := getMetricValue("sentrylogmon_monitor_lag_seconds", "source", sourceName)
	if !found {
		t.Fatalf("Metric sentrylogmon_monitor_lag_seconds not found for source %s", sourceName)
	}

	// Calculate expected lag
	// lag = now - (bootTime + ts)
	// ts = uptime - 10
	// lag = now - bootTime - uptime + 10
	// now - bootTime approx uptime
	// lag approx 10
	// However, precise calculation:

	bootTime, err := host.BootTime()
	if err != nil {
		t.Fatalf("Failed to get boot time: %v", err)
	}

	now := float64(time.Now().UnixNano()) / 1e9
	expectedLag := now - (float64(bootTime) + ts)

	// Allow margin
	if math.Abs(val-expectedLag) > 2.0 {
		t.Errorf("Expected lag around %f, got %f (diff: %f)", expectedLag, val, val-expectedLag)
	}
}
