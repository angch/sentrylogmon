package monitor

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/detectors"
	dto "github.com/prometheus/client_model/go"
	"io"
	"github.com/shirou/gopsutil/v3/host"
)

type uniqueMockSource struct {
	name  string
	lines []string
}

func (m *uniqueMockSource) Stream() (io.Reader, error) {
	return nil, nil // Not used in this test since we test processMatch directly
}

func (m *uniqueMockSource) Name() string {
	return m.name
}

func (m *uniqueMockSource) Close() error {
	return nil
}

// Minimal implementation of sources.LogSource for the test
type mockReader struct {
	*strings.Reader
}

func (m *mockReader) Close() error { return nil }

func (m *uniqueMockSource) StreamMock() (*mockReader, error) {
	return &mockReader{strings.NewReader(strings.Join(m.lines, "\n"))}, nil
}


func TestMonitorLagSeconds(t *testing.T) {
	ctx := context.Background()

	setupMonitor := func(srcName string) (*Monitor, func() (float64, error)) {
		src := &uniqueMockSource{name: srcName}
		detector, _ := detectors.NewGenericDetector("error")

		mon, err := New(ctx, src, detector, nil, Options{})
		if err != nil {
			t.Fatalf("Failed to create monitor: %v", err)
		}

		// Helper to get metric value
		getLagMetric := func() (float64, error) {
			metric := &dto.Metric{}
			err := mon.metricMonitorLag.Write(metric)
			if err != nil {
				return 0, err
			}
			if metric.Gauge == nil {
				return 0, fmt.Errorf("gauge is nil")
			}
			if metric.Gauge.Value == nil {
				return 0, nil
			}
			return *metric.Gauge.Value, nil
		}
		return mon, getLagMetric
	}

	t.Run("Absolute Timestamp", func(t *testing.T) {
		mon, getLagMetric := setupMonitor("lag_test_source_abs")

		// Log from exactly 5 seconds ago
		logTime := time.Now().Add(-5 * time.Second)
		logLine := fmt.Sprintf("%s error: something happened", logTime.Format(time.RFC3339))

		// Set internal time to now so we control it
		mon.lastActivityTime = time.Now()

		mon.processMatch([]byte(logLine))

		val, err := getLagMetric()
		if err != nil {
			t.Fatalf("Failed to get metric: %v", err)
		}

		// Due to precision loss in time formatting, allow up to 1s variance
		if val < 5.0 || val > 6.0 {
			t.Errorf("Expected lag ~5.0s, got %fs", val)
		}
	})

	t.Run("Relative Timestamp (dmesg)", func(t *testing.T) {
		mon, getLagMetric := setupMonitor("lag_test_source_rel")

		up, err := host.Uptime()
		if err != nil {
			t.Skip("host.Uptime() failed:", err)
		}

		// Log from 5 seconds ago using dmesg format
		relTime := float64(up) - 5.0
		if relTime < 0 {
			relTime = 0.1 // fallback
		}

		logLine := fmt.Sprintf("[ %f] error: kernel oops", relTime)
		mon.lastActivityTime = time.Now()
		mon.processMatch([]byte(logLine))

		val, err := getLagMetric()
		if err != nil {
			t.Fatalf("Failed to get metric: %v", err)
		}

		// Allow up to 1 second variance due to Uptime() int cast
		if val < 4.0 || val > 6.0 {
			t.Errorf("Expected lag ~5.0s, got %fs", val)
		}
	})

	t.Run("Negative Lag Ignored", func(t *testing.T) {
		mon, getLagMetric := setupMonitor("lag_test_source_neg")

		// Log from 5 seconds in the future
		futureTime := time.Now().Add(5 * time.Second)
		logLine := fmt.Sprintf("%s error: time machine", futureTime.Format(time.RFC3339))

		mon.lastActivityTime = time.Now()
		mon.processMatch([]byte(logLine))

		val, err := getLagMetric()
		if err != nil {
			t.Fatalf("Failed to get metric: %v", err)
		}

		if val != 0 {
			t.Errorf("Expected lag to remain 0 for future timestamps, got %f", val)
		}
	})
}
