package monitor

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/metrics"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestLastActivityMetric(t *testing.T) {
	// Reset metrics to ensure clean state
	metrics.LastActivityTimestamp.Reset()

	input := "line1\nline2\n"
	source := &MockSource{content: input}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Run monitor
	mon.Start()

	// Verify metric
	// We need to look up the metric with label source="mock"
	// Note: MockSource.Name() returns "mock"
	m := metrics.LastActivityTimestamp.With(prometheus.Labels{"source": "mock"})

	// Read metric value
	var metric dto.Metric
	err = m.Write(&metric)
	if err != nil {
		t.Fatalf("Failed to read metric: %v", err)
	}

	val := metric.GetGauge().GetValue()
	now := float64(time.Now().Unix())

	// Check if value is close to now (within 5 seconds)
	// If it wasn't updated, it would be 0 or nil (panic above if nil, but GetValue returns float64)
	if val == 0 {
		t.Errorf("Metric value is 0, expected it to be updated")
	}

	if now-val > 5 {
		t.Errorf("Metric value too old. Got %v, expected ~%v", val, now)
	}
	if val > now+1 {
		t.Errorf("Metric value in future. Got %v, expected ~%v", val, now)
	}
}

func TestMonitorLagMetric(t *testing.T) {
	// Reset metrics to ensure clean state
	metrics.MonitorLag.Reset()

	// Log with ISO8601 timestamp 10 seconds in the past
	pastTime := time.Now().Add(-10 * time.Second)
	// We use space-separated format that ParseISO8601 supports or just RFC3339
	pastTimeStr := pastTime.Format(time.RFC3339)
	input := pastTimeStr + " error occurred\n"

	detector := &MockDetector{}

	// Create a quick local implementation
	localSource := &LocalMockSource{content: input}

	mon, err := New(context.Background(), localSource, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Run monitor
	mon.Start()

	// Verify metric
	m := metrics.MonitorLag.With(prometheus.Labels{"source": "local_mock"})

	var metric dto.Metric
	err = m.Write(&metric)
	if err != nil {
		t.Fatalf("Failed to read metric: %v", err)
	}

	val := metric.GetGauge().GetValue()

	// Lag should be around 10 seconds. We'll check if it's between 9.0 and 11.0 to account for test execution time and truncated RFC3339 nanos.
	if val < 9.0 || val > 12.0 {
		t.Errorf("Metric value for lag is incorrect. Got %v, expected ~10.0", val)
	}
}

// LocalMockSource implements sources.LogSource for TestMonitorLagMetric
type LocalMockSource struct {
	content string
}
func (s *LocalMockSource) Name() string { return "local_mock" }
func (s *LocalMockSource) Stream() (io.Reader, error) {
	return strings.NewReader(s.content), nil
}
func (s *LocalMockSource) Close() error { return nil }
