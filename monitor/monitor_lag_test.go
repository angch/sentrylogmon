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
	"github.com/shirou/gopsutil/v3/host"
)

// uniqueMockSource is a mock log source for testing that supports unique names
type uniqueMockSource struct {
	name  string
	lines []string
}

func (m *uniqueMockSource) Stream() (io.Reader, error) {
	return strings.NewReader(strings.Join(m.lines, "\n")), nil
}

func (m *uniqueMockSource) Close() error {
	return nil
}

func (m *uniqueMockSource) Name() string {
	return m.name
}

func getMetricValue(metricVec *prometheus.GaugeVec, labelValue string) (float64, error) {
	m := &dto.Metric{}
	err := metricVec.WithLabelValues(labelValue).Write(m)
	if err != nil {
		return 0, err
	}
	if m.Gauge == nil {
		return 0, fmt.Errorf("metric is not a gauge")
	}
	return m.Gauge.GetValue(), nil
}

func TestMonitorLagSeconds(t *testing.T) {
	t.Run("Absolute_Timestamp", func(t *testing.T) {
		sourceName := "test_absolute_lag"
		src := &uniqueMockSource{
			name: sourceName,
		}

		// Create a timestamp exactly 5 seconds ago
		logTime := time.Now().Add(-5 * time.Second).UTC()
		// Avoid square brackets around RFC3339 as per memory instructions
		logLine := fmt.Sprintf("%s myapp: error message", logTime.Format(time.RFC3339))

		genericDetector, _ := detectors.NewGenericDetector("")
		m, err := New(context.Background(), src, genericDetector, nil, Options{})
		if err != nil {
			t.Fatalf("Failed to create monitor: %v", err)
		}

		m.processMatch([]byte(logLine))

		lag, err := getMetricValue(metrics.MonitorLagSeconds, sourceName)
		if err != nil {
			t.Fatalf("Expected metric to be set, got error: %v", err)
		}

		// Because RFC3339 truncates nanoseconds, the difference can be up to 1 second larger
		// than the exact 5.0 seconds. As stated in memory: "expecting 5.0 to 6.0 instead of ~5.0".
		if lag < 5.0 || lag > 6.0 {
			t.Errorf("Expected lag between 5.0 and 6.0, got %f", lag)
		}
	})

	t.Run("Relative_Timestamp", func(t *testing.T) {
		sourceName := "test_relative_lag"
		src := &uniqueMockSource{
			name: sourceName,
		}

		// We use dmesg-like format for relative timestamps.
		// [ 12345.123456] ...

		// To simulate a lag of 10 seconds:
		// logTime = Now - 10s
		// bootTime = Now - Uptime
		// relativeTime = logTime - bootTime = Uptime - 10s
		uptime, err := host.Uptime()
		if err != nil {
			t.Fatalf("Failed to get uptime: %v", err)
		}

		relativeTimeSecs := float64(uptime) - 10.0
		if relativeTimeSecs < 0 {
			relativeTimeSecs = 1.0 // just in case the system just booted
		}

		logLine := fmt.Sprintf("[ %f] myapp: test error message", relativeTimeSecs)

		m, err := New(context.Background(), src, detectors.NewDmesgDetector(), nil, Options{})
		if err != nil {
			t.Fatalf("Failed to create monitor: %v", err)
		}

		m.processMatch([]byte(logLine))

		lag, err := getMetricValue(metrics.MonitorLagSeconds, sourceName)
		if err != nil {
			t.Fatalf("Expected metric to be set, got error: %v", err)
		}

		// Allow variance since Uptime lacks sub-second precision and might have slight delays
		// Typically, difference can be around 10.0 to 11.0s. Memory says allow up to 1 second variance.
		if lag < 9.0 || lag > 12.0 {
			t.Errorf("Expected lag around 10.0-11.0, got %f", lag)
		}
	})

	t.Run("Negative_Lag", func(t *testing.T) {
		sourceName := "test_negative_lag"
		src := &uniqueMockSource{
			name: sourceName,
		}

		// Timestamp 10 seconds in the future
		logTime := time.Now().Add(10 * time.Second).UTC()
		logLine := fmt.Sprintf("%s myapp: error message", logTime.Format(time.RFC3339))

		genericDetector, _ := detectors.NewGenericDetector("")
		m, err := New(context.Background(), src, genericDetector, nil, Options{})
		if err != nil {
			t.Fatalf("Failed to create monitor: %v", err)
		}

		m.processMatch([]byte(logLine))

		val, err := getMetricValue(metrics.MonitorLagSeconds, sourceName)
		// Usually if not initialized it might return 0, or err. If 0, it means it wasn't modified.
		if err == nil && val != 0 {
			t.Errorf("Expected negative lag to be ignored (metric=0 or not found), but got %f", val)
		}
	})
}
