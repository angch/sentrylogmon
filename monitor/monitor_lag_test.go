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
	"github.com/shirou/gopsutil/v3/host"
)

type uniqueMockSource struct {
	name    string
	content string
}

func (s *uniqueMockSource) Name() string { return s.name }
func (s *uniqueMockSource) Stream() (io.Reader, error) {
	return strings.NewReader(s.content), nil
}
func (s *uniqueMockSource) Close() error { return nil }

func getGaugeValue(vec *prometheus.GaugeVec, labelValue string) (float64, error) {
	metrics, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return 0, err
	}
	for _, mf := range metrics {
		if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
			for _, m := range mf.GetMetric() {
				for _, lp := range m.GetLabel() {
					if lp.GetName() == "source" && lp.GetValue() == labelValue {
						return m.GetGauge().GetValue(), nil
					}
				}
			}
		}
	}
	return 0, fmt.Errorf("metric not found")
}

func TestMonitorLagAbsolute(t *testing.T) {
	sourceName := "test_lag_absolute"

	// Create a log line from 5 seconds ago
	tLog := time.Now().Add(-5 * time.Second)
	// We use an ISO8601 format to ensure absolute timestamp parsing
	tsStr := tLog.Format("2006-01-02T15:04:05Z07:00")
	line := fmt.Sprintf("%s [error] some message", tsStr)

	source := &uniqueMockSource{
		name:    sourceName,
		content: line,
	}

	opts := Options{
		Verbose: false,
	}

	m, err := New(context.Background(), source, &MockDetector{}, nil, opts)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	m.StopOnEOF = true
	m.Start()

	// In Go tests, formatting a time.Time object with time.RFC3339 truncates nanoseconds.
	// When computing durations (like lag) against this truncated time format, the result
	// can be up to 1 second larger than the exact duration; test assertions must allow
	// for this variance (e.g., expecting 5.0 to 6.0 instead of ~5.0).
	val, err := getGaugeValue(metrics.MonitorLagSeconds, sourceName)
	if err != nil {
		t.Fatalf("Failed to get metric value: %v", err)
	}

	if val < 5.0 || val > 6.0 {
		t.Errorf("Expected lag to be between 5.0 and 6.0, got %f", val)
	}
}

func TestMonitorLagRelative(t *testing.T) {
	sourceName := "test_lag_relative"

	uptime, err := host.Uptime()
	if err != nil {
		t.Skipf("Failed to get host uptime, skipping test: %v", err)
	}

	// Calculate a relative timestamp (e.g. 5 seconds ago)
	var relTS float64
	if uptime > 5 {
		relTS = float64(uptime) - 5.0
	} else {
		// If system booted less than 5 seconds ago (unlikely but possible), use 0.1
		relTS = 0.1
	}

	// Dmesg format [   12.345678] message
	line := fmt.Sprintf("[ %11.6f] error message", relTS)

	source := &uniqueMockSource{
		name:    sourceName,
		content: line,
	}

	opts := Options{
		Verbose: false,
	}

	m, err := New(context.Background(), source, &MockDetector{}, nil, opts)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	m.StopOnEOF = true
	m.Start()

	val, err := getGaugeValue(metrics.MonitorLagSeconds, sourceName)
	if err != nil {
		t.Fatalf("Failed to get metric value: %v", err)
	}

	// Assuming the lag is ~5 seconds. Due to `host.BootTime()` and `host.Uptime()`
	// not returning sub-second precision, there can be ~1s variance.
	if uptime > 5 {
		if val < 4.0 || val > 6.0 {
			t.Errorf("Expected relative lag to be between 4.0 and 6.0, got %f", val)
		}
	} else {
		if val < 0 {
			t.Errorf("Expected lag to be non-negative, got %f", val)
		}
	}
}

func TestMonitorLagNegativeIgnored(t *testing.T) {
	sourceName := "test_lag_negative"

	// Log from 1 hour in the FUTURE
	tLog := time.Now().Add(1 * time.Hour)
	tsStr := tLog.Format("2006-01-02T15:04:05Z07:00")
	line := fmt.Sprintf("%s [error] some message from the future", tsStr)

	source := &uniqueMockSource{
		name:    sourceName,
		content: line,
	}

	opts := Options{
		Verbose: false,
	}

	m, err := New(context.Background(), source, &MockDetector{}, nil, opts)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	m.StopOnEOF = true
	m.Start()

	// When verifying that a Prometheus Gauge metric is ignored or unset in Go tests
	// (e.g., negative monitor lag), check that the value is `0` or the metric returns
	// a 'not found' error, as uninitialized Gauges in the DefaultGatherer may evaluate to 0.
	val, err := getGaugeValue(metrics.MonitorLagSeconds, sourceName)
	if err != nil && err.Error() != "metric not found" {
		t.Fatalf("Unexpected error getting metric: %v", err)
	}

	if err == nil && val != 0 {
		t.Errorf("Expected negative lag to be ignored (0 or not found), got %f", val)
	}
}
