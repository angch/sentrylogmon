package monitor

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/metrics"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// uniqueMockSource wraps MockSource to provide a unique name
// We rely on MockSource being defined in monitor_test.go in the same package
type uniqueMockSource struct {
	MockSource
	name string
}

func (s *uniqueMockSource) Name() string {
	return s.name
}

// Delegate Stream explicitly to avoid ambiguity or embedding issues
func (s *uniqueMockSource) Stream() (io.Reader, error) {
	return s.MockSource.Stream()
}

func (s *uniqueMockSource) Close() error {
	return s.MockSource.Close()
}

func TestMonitorLag_Absolute(t *testing.T) {
	sourceName := "mock_lag_absolute"

	now := time.Now()
	// 5 seconds lag
	lagDuration := 5 * time.Second
	logTime := now.Add(-lagDuration)

	// RFC3339 format: 2006-01-02T15:04:05Z07:00
	logLine := logTime.Format(time.RFC3339) + " Test log message"

	source := &uniqueMockSource{
		MockSource: MockSource{content: logLine},
		name:       sourceName,
	}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Run Monitor synchronously as it will stop on EOF
	mon.Start()

	// Verify Metric
	gauge, err := metrics.MonitorLagSeconds.GetMetricWith(prometheus.Labels{"source": sourceName})
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	var m dto.Metric
	if err := gauge.Write(&m); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	val := m.GetGauge().GetValue()

	t.Logf("Measured lag: %f", val)

	// We expect roughly 5 seconds.
	// Allow 0.5s margin for test execution slowness.
	if val < 4.5 || val > 6.0 {
		t.Errorf("Expected lag around 5.0s, got %f", val)
	}
}
