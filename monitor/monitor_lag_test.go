package monitor

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// uniqueMockSource overrides the source name for Prometheus registry isolation.
type uniqueMockSource struct {
	name string
}

func (s *uniqueMockSource) Stream() (io.Reader, error) {
	return nil, nil // Not used in this test since we test processMatch directly
}

func (s *uniqueMockSource) Close() error {
	return nil
}

func (s *uniqueMockSource) Name() string {
	return s.name
}

func TestMonitorLagMetrics(t *testing.T) {
	// Gatherer to read metrics
	gatherer := prometheus.DefaultGatherer

	t.Run("PositiveLag", func(t *testing.T) {
		sourceName := "test_positive_lag_source"

		m := &Monitor{
			ctx:       context.Background(),
			Source:    &uniqueMockSource{name: sourceName},
			Detector:  &MockDetector{}, // from monitor_test.go
		}

		// Simulate a log line from 5 seconds ago
		logTime := time.Now().Add(-5 * time.Second)
		// Don't wrap with [] per memory: "When formatting mock log lines with RFC3339 timestamps for tests, avoid wrapping the timestamp in square brackets"
		logLine := fmt.Sprintf("%s Test positive lag message\n", logTime.Format(time.RFC3339))

		m.processMatch([]byte(logLine))

		mfs, err := gatherer.Gather()
		if err != nil {
			t.Fatalf("Failed to gather metrics: %v", err)
		}

		var found bool
		for _, mf := range mfs {
			if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
				for _, metric := range mf.GetMetric() {
					var isSource bool
					for _, label := range metric.GetLabel() {
						if label.GetName() == "source" && label.GetValue() == sourceName {
							isSource = true
						}
					}
					if isSource {
						found = true
						lag := metric.GetGauge().GetValue()
						// Allow for 1 second variance due to RFC3339 truncation and execution time
						if lag < 4.0 || lag > 6.0 {
							t.Errorf("Expected lag between 4.0 and 6.0, got %v", lag)
						}
					}
				}
			}
		}

		if !found {
			t.Errorf("Metric sentrylogmon_monitor_lag_seconds for source %s not found", sourceName)
		}
	})

	t.Run("NegativeLag", func(t *testing.T) {
		sourceName := "test_negative_lag_source"

		m := &Monitor{
			ctx:       context.Background(),
			Source:    &uniqueMockSource{name: sourceName},
			Detector:  &MockDetector{},
		}

		// Simulate a log line from 5 seconds in the future
		logTime := time.Now().Add(5 * time.Second)
		logLine := fmt.Sprintf("%s Test negative lag message\n", logTime.Format(time.RFC3339))

		m.processMatch([]byte(logLine))

		mfs, err := gatherer.Gather()
		if err != nil {
			t.Fatalf("Failed to gather metrics: %v", err)
		}

		for _, mf := range mfs {
			if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
				for _, metric := range mf.GetMetric() {
					var isSource bool
					for _, label := range metric.GetLabel() {
						if label.GetName() == "source" && label.GetValue() == sourceName {
							isSource = true
						}
					}
					if isSource {
						val := metric.GetGauge().GetValue()
						if val != 0 {
							t.Errorf("Expected lag to be 0 or unrecorded for negative lag, got %v", val)
						}
					}
				}
			}
		}
	})
}
