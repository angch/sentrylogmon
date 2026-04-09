package monitor

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// uniqueMockSource allows us to override the source name to prevent metric collisions
// across tests in the same package.
type uniqueMockSource struct {
	name    string
	content string
}

func (s *uniqueMockSource) Name() string               { return s.name }
func (s *uniqueMockSource) Stream() (io.Reader, error) { return strings.NewReader(s.content), nil }
func (s *uniqueMockSource) Close() error               { return nil }

func TestMonitorLagCalculation(t *testing.T) {
	// Let's create a log with a specific timestamp.
	// For instance: ISO8601: "2023-10-27T10:00:00Z" (Unix: 1698393600)
	// We'll calculate the expected lag based on when the test runs.
	logTimeStr := "2023-10-27T10:00:00Z"
	logTime, _ := time.Parse(time.RFC3339, logTimeStr)

	input := fmt.Sprintf("%s Error something happened", logTimeStr)
	sourceName := "test_lag_source_1"
	source := &uniqueMockSource{name: sourceName, content: input}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Record start time to calculate expected range
	startProcessTime := float64(time.Now().UnixNano()) / 1e9

	// Run monitor directly, no goroutine, to ensure it processes
	mon.Start()

	endProcessTime := float64(time.Now().UnixNano()) / 1e9

	expectedLagMin := startProcessTime - float64(logTime.UnixNano())/1e9
	expectedLagMax := endProcessTime - float64(logTime.UnixNano())/1e9

	// Wait for any async stuff (though in this case Start should block until StopOnEOF)
	// Let's get the metric value.
	metricValue := testutil.ToFloat64(metrics.MonitorLagSeconds.WithLabelValues(sourceName))

	if metricValue < expectedLagMin || metricValue > expectedLagMax {
		t.Errorf("Lag calculation incorrect. Expected between %f and %f, got %f", expectedLagMin, expectedLagMax, metricValue)
	}
}

func TestMonitorLagIgnoreNegative(t *testing.T) {
	// Let's create a log with a future timestamp.
	futureTime := time.Now().Add(10 * time.Minute)
	logTimeStr := futureTime.Format(time.RFC3339)

	input := fmt.Sprintf("%s Error something happened", logTimeStr)
	sourceName := "test_lag_source_2"
	source := &uniqueMockSource{name: sourceName, content: input}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Ensure the metric starts at 0 or doesn't exist.
	// We'll set it to 0 explicitly to track changes.
	metrics.MonitorLagSeconds.WithLabelValues(sourceName).Set(0)

	// Run monitor directly
	mon.Start()

	// The lag is negative, so the metric should NOT be updated. It should remain 0.
	metricValue := testutil.ToFloat64(metrics.MonitorLagSeconds.WithLabelValues(sourceName))

	if metricValue != 0 {
		t.Errorf("Expected lag metric to remain 0 for future timestamps, but got %f", metricValue)
	}
}
