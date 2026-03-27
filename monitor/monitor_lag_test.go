package monitor

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/detectors"
	"github.com/angch/sentrylogmon/metrics"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/shirou/gopsutil/v3/host"
)

type uniqueMockSource struct {
	name    string
	content string
}

func (s *uniqueMockSource) Name() string {
	return s.name
}

func (s *uniqueMockSource) Stream() (io.Reader, error) {
	return nil, nil // Not used for this manual test
}

func (s *uniqueMockSource) Close() error {
	return nil
}

func getMetricValue(sourceName string) (float64, error) {
	m := metrics.MonitorLagSeconds.With(prometheus.Labels{"source": sourceName})
	var metric dto.Metric
	err := m.Write(&metric)
	if err != nil {
		return 0, err
	}
	if metric.Gauge == nil {
		return 0, fmt.Errorf("metric not found or not a gauge")
	}
	return metric.GetGauge().GetValue(), nil
}

func TestMonitorLagAbsolute(t *testing.T) {
	sourceName := "mock_lag_absolute"
	source := &uniqueMockSource{name: sourceName, content: ""}
	// We use ISO8601 to ensure it's treated as absolute time
	detector, _ := detectors.NewGenericDetector("(?i)error")

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	// Calculate a log timestamp that is 5 seconds in the past
	now := time.Now()
	logTime := now.Add(-5 * time.Second)
	logTimeStr := logTime.Format(time.RFC3339)
	line := []byte(fmt.Sprintf("%s Error something went wrong", logTimeStr))

	mon.processMatch(line, now)

	// Fetch lag metric
	val, err := getMetricValue(sourceName)
	if err != nil {
		t.Fatalf("Failed to read lag metric: %v", err)
	}

	// Because time.RFC3339 truncates nanoseconds, the lag might be slightly more than 5.0 (up to 6.0)
	if val < 5.0 || val > 6.0 {
		t.Errorf("Expected lag between 5.0s and 6.0s, got %f", val)
	}
}

func TestMonitorLagRelative(t *testing.T) {
	sourceName := "mock_lag_relative"
	source := &uniqueMockSource{name: sourceName, content: ""}
	detector := detectors.NewDmesgDetector()

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	// We calculate input timestamp dynamically using host.Uptime() minus a logical offset
	uptime, err := host.Uptime()
	if err != nil || uptime == 0 {
		t.Skip("Skipping relative test because host uptime could not be retrieved")
	}

	// Let's pretend the log was generated 10 seconds ago
	logUptime := float64(uptime) - 10.0
	if logUptime < 0 {
		logUptime = 1.0 // Just in case system just booted
	}

	line := []byte(fmt.Sprintf("[%12.6f] Error from kernel", logUptime))

	now := time.Now()
	mon.processMatch(line, now)

	val, err := getMetricValue(sourceName)
	if err != nil {
		t.Fatalf("Failed to read lag metric: %v", err)
	}

	// Depending on time elapsed since uptime fetch, it should be approximately 10 seconds
	if val < 9.0 || val > 11.0 {
		t.Errorf("Expected relative lag ~10.0s, got %f", val)
	}
}

func TestMonitorLagNegative(t *testing.T) {
	sourceName := "mock_lag_negative"
	source := &uniqueMockSource{name: sourceName, content: ""}
	detector, _ := detectors.NewGenericDetector("(?i)error")

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	// Log time is 5 seconds in the FUTURE (clock skew scenario)
	now := time.Now()
	logTime := now.Add(5 * time.Second)
	logTimeStr := logTime.Format(time.RFC3339)
	line := []byte(fmt.Sprintf("%s Error something went wrong", logTimeStr))

	mon.processMatch(line, now)

	// Fetch lag metric
	val, err := getMetricValue(sourceName)
	if err != nil {
		t.Fatalf("Failed to read lag metric: %v", err)
	}

	// The lag is negative (-5s), so metricMonitorLag.Set(lag) should NOT be called.
	// Since gauge starts uninitialized (or we expect it to be 0 if uninitialized),
	// check that the value is 0. Uninitialized Gauges in DefaultGatherer evaluate to 0
	if val != 0 {
		t.Errorf("Expected negative lag to be ignored (metric=0), got %f", val)
	}
}
