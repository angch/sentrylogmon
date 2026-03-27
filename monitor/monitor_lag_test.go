package monitor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v3/host"
)

type uniqueMockSource struct {
	MockSource
	name string
}

func (s *uniqueMockSource) Name() string { return s.name }

type lagMockDetector struct {
	MockDetector
	timestamp float64
	tsStr     string
	ok        bool
}

func (d *lagMockDetector) ExtractTimestamp(line []byte) (float64, string, bool) {
	return d.timestamp, d.tsStr, d.ok
}

func getMetricValue(t *testing.T, metricName, source string) (float64, bool) {
	metrics, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}
	for _, mf := range metrics {
		if mf.GetName() == metricName {
			for _, m := range mf.GetMetric() {
				hasSourceLabel := false
				for _, label := range m.GetLabel() {
					if label.GetName() == "source" && label.GetValue() == source {
						hasSourceLabel = true
						break
					}
				}
				if hasSourceLabel {
					if m.GetGauge() != nil {
						return m.GetGauge().GetValue(), true
					}
				}
			}
		}
	}
	return 0, false
}

func TestMonitorLagAbsolute(t *testing.T) {
	sourceName := "mock_lag_test_absolute"
	source := &uniqueMockSource{name: sourceName, MockSource: MockSource{content: "log line 1\n"}}

	// Set absolute timestamp to exactly 5 seconds ago
	now := time.Now()
	// Round now down to closest second to avoid precision mismatch during lag test
	nowStr := now.Format(time.RFC3339)
	nowRounded, _ := time.Parse(time.RFC3339, nowStr)

	testTime := nowRounded.Add(-5 * time.Second)
	testTimestamp := float64(testTime.UnixNano()) / 1e9

	// Use our mock detector to force the timestamp
	detector := &lagMockDetector{
		timestamp: testTimestamp,
		tsStr:     testTime.Format(time.RFC3339),
		ok:        true,
	}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	go mon.Start()
	time.Sleep(100 * time.Millisecond) // wait for processing

	lag, found := getMetricValue(t, "sentrylogmon_monitor_lag_seconds", sourceName)
	if !found {
		t.Fatal("sentrylogmon_monitor_lag_seconds metric not found")
	}

	// We expect lag to be exactly 5.0s, allow up to 6.0s for processing
	if lag < 5.0 || lag > 6.0 {
		t.Errorf("Expected lag to be between 5.0 and 6.0, got %f", lag)
	}
}

func TestMonitorLagRelative(t *testing.T) {
	sourceName := "mock_lag_test_relative"
	source := &uniqueMockSource{name: sourceName, MockSource: MockSource{content: "log line 2\n"}}

	bt, err := host.BootTime()
	if err != nil {
		t.Skip("Failed to get boot time, skipping relative test")
	}

	uptime, err := host.Uptime()
	if err != nil {
		t.Skip("Failed to get uptime, skipping relative test")
	}

	// Create relative timestamp 10 seconds ago
	relTimestamp := float64(uptime) - 10.0
	if relTimestamp < 0 {
		relTimestamp = 1.0 // safeguard for very quick test runners
	}

	detector := &lagMockDetector{
		timestamp: relTimestamp,
		tsStr:     fmt.Sprintf("%f", relTimestamp),
		ok:        true,
	}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true
	// Ensure bootTime is captured exactly
	mon.bootTime = float64(bt)

	go mon.Start()
	time.Sleep(100 * time.Millisecond) // wait for processing

	lag, found := getMetricValue(t, "sentrylogmon_monitor_lag_seconds", sourceName)
	if !found {
		t.Fatal("sentrylogmon_monitor_lag_seconds metric not found")
	}

	// Lag should be around 10 seconds
	if lag < 9.0 || lag > 11.0 {
		t.Errorf("Expected lag around 10.0s, got %f", lag)
	}
}

func TestMonitorLagNegativeIgnored(t *testing.T) {
	sourceName := "mock_lag_test_negative"
	source := &uniqueMockSource{name: sourceName, MockSource: MockSource{content: "log line 3\n"}}

	now := time.Now()
	// Set timestamp 1 hour in the future (negative lag)
	futureTime := now.Add(1 * time.Hour)
	testTimestamp := float64(futureTime.UnixNano()) / 1e9

	detector := &lagMockDetector{
		timestamp: testTimestamp,
		tsStr:     futureTime.Format(time.RFC3339),
		ok:        true,
	}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	go mon.Start()
	time.Sleep(100 * time.Millisecond)

	lag, found := getMetricValue(t, "sentrylogmon_monitor_lag_seconds", sourceName)
	// Prometheus might not report gauges that are completely uninitialized if not preset to 0
	if found && lag != 0 {
		t.Errorf("Expected metric to be ignored or 0 for negative lag, got %f", lag)
	}
}
