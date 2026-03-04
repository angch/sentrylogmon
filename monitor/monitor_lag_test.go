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
	"github.com/shirou/gopsutil/v3/host"
)

type uniqueMockSource struct {
	MockSource
	name string
}

func (s *uniqueMockSource) Name() string { return s.name }
func (s *uniqueMockSource) Stream() (io.Reader, error) {
	return strings.NewReader(s.content), nil
}
func (s *uniqueMockSource) Close() error { return nil }

func TestMonitorLagAbsolute(t *testing.T) {
	metrics.MonitorLagSeconds.Reset()

	now := time.Now()
	// Create a log line 5 seconds in the past using RFC3339 layout
	logTime := now.Add(-5 * time.Second)
	// Using RFC3339 which truncates nanoseconds
	input := fmt.Sprintf("%s Test line\n", logTime.Format(time.RFC3339))

	sourceName := "mock_lag_abs"
	source := &uniqueMockSource{name: sourceName, MockSource: MockSource{content: input}}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	mon.Start()

	m := metrics.MonitorLagSeconds.With(prometheus.Labels{"source": sourceName})
	var metric dto.Metric
	if err := m.Write(&metric); err != nil {
		t.Fatalf("Failed to read metric: %v", err)
	}

	val := metric.GetGauge().GetValue()

	// Because time.RFC3339 truncates nanoseconds, the difference can be slightly more than 5 seconds
	if val < 5.0 || val > 6.0 {
		t.Errorf("Expected lag to be between 5.0 and 6.0 seconds, got %v", val)
	}
}

func TestMonitorLagRelative(t *testing.T) {
	metrics.MonitorLagSeconds.Reset()

	bootTime, err := host.BootTime()
	if err != nil {
		t.Skipf("Failed to get boot time: %v", err)
	}

	now := float64(time.Now().UnixNano()) / 1e9
	// Create a log line 10 seconds in the past, formatted as relative uptime (dmesg)
	logUptime := now - float64(bootTime) - 10.0
	input := fmt.Sprintf("[ %.6f] Test dmesg line\n", logUptime)

	sourceName := "mock_lag_rel"
	source := &uniqueMockSource{name: sourceName, MockSource: MockSource{content: input}}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	mon.Start()

	m := metrics.MonitorLagSeconds.With(prometheus.Labels{"source": sourceName})
	var metric dto.Metric
	if err := m.Write(&metric); err != nil {
		t.Fatalf("Failed to read metric: %v", err)
	}

	val := metric.GetGauge().GetValue()

	if val < 10.0 || val > 11.0 {
		t.Errorf("Expected lag to be between 10.0 and 11.0 seconds, got %v", val)
	}
}

func TestMonitorLagNegative(t *testing.T) {
	metrics.MonitorLagSeconds.Reset()

	now := time.Now()
	// Create a log line 5 seconds in the future
	logTime := now.Add(5 * time.Second)
	input := fmt.Sprintf("%s Test future line\n", logTime.Format(time.RFC3339))

	sourceName := "mock_lag_neg"
	source := &uniqueMockSource{name: sourceName, MockSource: MockSource{content: input}}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	mon.Start()

	m := metrics.MonitorLagSeconds.With(prometheus.Labels{"source": sourceName})
	var metric dto.Metric
	if err := m.Write(&metric); err != nil {
		t.Fatalf("Failed to read metric: %v", err)
	}

	val := metric.GetGauge().GetValue()

	if val != 0 {
		t.Errorf("Expected negative lag to be ignored (metric should remain 0), got %v", val)
	}
}
