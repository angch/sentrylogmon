package monitor

import (
	"context"
	"fmt"
	"io"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// uniqueMockSource overrides the source name for Prometheus registry isolation.
type uniqueMockSource struct {
	name    string
	content string
}

func (s *uniqueMockSource) Name() string { return s.name }
func (s *uniqueMockSource) Stream() (io.Reader, error) {
	return strings.NewReader(s.content), nil
}
func (s *uniqueMockSource) Close() error { return nil }

func TestMonitorLagMetric(t *testing.T) {
	// Generate a log line with a timestamp 5 seconds in the past.
	// Using ISO8601 format to avoid boot time logic for this simple test.
	pastTime := time.Now().Add(-5 * time.Second)
	// We use RFC3339 which truncates to seconds, so the actual parsed time
	// might be slightly different. The memory guideline says to allow up to 1 second variance.
	tsStr := pastTime.Format(time.RFC3339)
	input := fmt.Sprintf("%s Test error message\n", tsStr)

	sourceName := fmt.Sprintf("test_lag_source_%d", time.Now().UnixNano())
	source := &uniqueMockSource{
		name:    sourceName,
		content: input,
	}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Run monitor. It will process the line, extract the timestamp, and record the lag.
	// We run it synchronously for the test by waiting for Start to return (since StopOnEOF=true).

	// Start in a goroutine because we want to wait for it or just run it directly.
	// Since StopOnEOF is true, Start will return.
	mon.Start()

	// Give a little time for Prometheus metric to be recorded
	time.Sleep(100 * time.Millisecond)

	// Check the Prometheus DefaultGatherer
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	found := false
	for _, mf := range mfs {
		if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
			for _, m := range mf.GetMetric() {
				// Check if the label matches our unique source
				for _, label := range m.GetLabel() {
					if label.GetName() == "source" && label.GetValue() == sourceName {
						found = true
						val := m.GetGauge().GetValue()
						// Expected lag is around 5 seconds.
						// Due to truncation in RFC3339 formatting, the logged time could be up to 1 second older than the actual time used to calculate it, plus execution time.
						// The lag should be between 4.5 and 6.5.
						if val < 4.0 || val > 7.0 {
							t.Errorf("Expected monitor lag around 5 seconds, got %f", val)
						} else {
							t.Logf("Got valid monitor lag: %f seconds", val)
						}
					}
				}
			}
		}
	}

	if !found {
		t.Errorf("MonitorLagSeconds metric not found for source %s", sourceName)
	}
}

func TestMonitorLagMetric_NegativeIgnored(t *testing.T) {
	// Generate a log line with a timestamp 5 seconds in the future.
	futureTime := time.Now().Add(5 * time.Second)
	tsStr := futureTime.Format(time.RFC3339)
	input := fmt.Sprintf("%s Test error message\n", tsStr)

	sourceName := fmt.Sprintf("test_lag_source_neg_%d", time.Now().UnixNano())
	source := &uniqueMockSource{
		name:    sourceName,
		content: input,
	}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	mon.Start()

	time.Sleep(100 * time.Millisecond)

	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	for _, mf := range mfs {
		if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
			for _, m := range mf.GetMetric() {
				for _, label := range m.GetLabel() {
					if label.GetName() == "source" && label.GetValue() == sourceName {
						val := m.GetGauge().GetValue()
						// If the value is 0 or doesn't exist, it is ignored/unset as per memory.
						if val != 0 && !math.IsNaN(val) {
							t.Errorf("Expected monitor lag to be 0 or unrecorded for future timestamp, got %f", val)
						}
					}
				}
			}
		}
	}
}
