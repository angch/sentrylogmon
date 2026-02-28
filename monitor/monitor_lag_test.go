package monitor

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v3/host"
)

// uniqueMockSource wraps MockSource but allows setting a distinct name
// so that each test gets its own metric series under 'source'.
type uniqueMockSource struct {
	MockSource
	name string
}

func (s *uniqueMockSource) Name() string { return s.name }

// MockLagDetector implements detectors.Detector and TimestampExtractor
type MockLagDetector struct {
	timestamp float64
}

func (d *MockLagDetector) Detect(line []byte) bool { return true }
func (d *MockLagDetector) ExtractTimestamp(line []byte) (float64, string, bool) {
	return d.timestamp, "", true
}

func getMetricValue(gatherer prometheus.Gatherer, metricName string, labelName string, labelValue string) (float64, error) {
	mfs, err := gatherer.Gather()
	if err != nil {
		return 0, err
	}
	for _, mf := range mfs {
		if mf.GetName() == metricName {
			for _, m := range mf.GetMetric() {
				// Check labels
				matched := false
				for _, lp := range m.GetLabel() {
					if lp.GetName() == labelName && lp.GetValue() == labelValue {
						matched = true
						break
					}
				}
				if matched {
					if m.GetGauge() != nil {
						return m.GetGauge().GetValue(), nil
					}
				}
			}
		}
	}
	return 0, fmt.Errorf("metric %s with %s=%s not found", metricName, labelName, labelValue)
}

func TestMonitorLagAbsolute(t *testing.T) {
	// e.g., an ISO8601 absolute timestamp in the past
	now := time.Now()
	// Simulate log was written 5.5 seconds ago
	logTime := now.Add(-5500 * time.Millisecond)
	logTimeUnix := float64(logTime.UnixNano()) / 1e9

	// Provide a unique source name for this test
	sourceName := "mock_lag_test_abs"
	source := &uniqueMockSource{MockSource: MockSource{content: fmt.Sprintf("[%f] absolute log line\n", logTimeUnix)}, name: sourceName}
	detector := &MockLagDetector{timestamp: logTimeUnix}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true
	// We do not start the full monitor loop, just inject a matched line directly.
	// We need to set lastActivityTime to simulate typical operation
	mon.lastActivityTime = now

	mon.processMatch([]byte("fake line"))

	val, err := getMetricValue(prometheus.DefaultGatherer, "sentrylogmon_monitor_lag_seconds", "source", sourceName)
	if err != nil {
		t.Fatalf("Error getting metric: %v", err)
	}

	// The lag should be approximately 5.5 seconds
	if val < 5.4 || val > 5.6 {
		t.Errorf("Expected lag around 5.5s, got %f", val)
	}
}

func TestMonitorLagRelative(t *testing.T) {
	// e.g., dmesg/uptime-based timestamp
	bt, err := host.BootTime()
	if err != nil {
		t.Skipf("Skipping relative lag test because BootTime could not be retrieved: %v", err)
	}

	now := time.Now()
	nowUnix := float64(now.UnixNano()) / 1e9
	uptimeSeconds := nowUnix - float64(bt)

	// Simulate a kernel log emitted 10 seconds ago
	logUptime := uptimeSeconds - 10.0
	// Ensure logUptime is positive and < 1e9 (heuristic for uptime vs absolute)
	if logUptime < 0 || logUptime >= 1e9 {
		t.Skipf("Calculated uptime %f is outside expected bounds [0, 1e9)", logUptime)
	}

	sourceName := "mock_lag_test_rel"
	source := &uniqueMockSource{MockSource: MockSource{content: fmt.Sprintf("[%f] relative log line\n", logUptime)}, name: sourceName}
	detector := &MockLagDetector{timestamp: logUptime}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true
	mon.lastActivityTime = now

	mon.processMatch([]byte("fake line"))

	val, err := getMetricValue(prometheus.DefaultGatherer, "sentrylogmon_monitor_lag_seconds", "source", sourceName)
	if err != nil {
		t.Fatalf("Error getting metric: %v", err)
	}

	// The lag should be approximately 10.0 seconds
	if val < 9.9 || val > 10.1 {
		t.Errorf("Expected lag around 10.0s, got %f", val)
	}
}

func TestMonitorLagNegativeIgnored(t *testing.T) {
	// A timestamp in the future should result in a negative lag, which we want to ignore (don't set metric)
	now := time.Now()
	futureTime := now.Add(10 * time.Second)
	futureTimeUnix := float64(futureTime.UnixNano()) / 1e9

	sourceName := "mock_lag_test_neg"
	source := &uniqueMockSource{MockSource: MockSource{content: "fake"}, name: sourceName}
	detector := &MockLagDetector{timestamp: futureTimeUnix}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// mon.processMatch sets lastActivityTime = time.Now(), which will be close to `now`
	// and futureTime is 10s ahead of `now`, so lag will be negative.
	mon.processMatch([]byte("fake line"))

	val, err := getMetricValue(prometheus.DefaultGatherer, "sentrylogmon_monitor_lag_seconds", "source", sourceName)
	// In Prometheus, a Gauge is initialized to 0. Since we don't set it if lag < 0, it should remain 0.
	if err == nil && val != 0 {
		t.Errorf("Expected metric to be 0 or missing because lag is negative, but it was %v", val)
	} else if err != nil && !strings.Contains(err.Error(), "not found") {
		t.Errorf("Unexpected error: %v", err)
	}
}
