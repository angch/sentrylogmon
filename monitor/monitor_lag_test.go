package monitor

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus"
)

// NamedMockSource implements sources.LogSource with a custom name
type NamedMockSource struct {
	name    string
	content string
}

func (s *NamedMockSource) Name() string { return s.name }
func (s *NamedMockSource) Stream() (io.Reader, error) {
	return strings.NewReader(s.content), nil
}
func (s *NamedMockSource) Close() error { return nil }

func TestMonitorLagMetric(t *testing.T) {
	// Setup Sentry Mock (needed to avoid errors/network calls)
	transport := &MockTransport{}
	err := sentry.Init(sentry.ClientOptions{
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	// Create a log line with a timestamp 10 seconds in the past
	// We use UTC to match the monitor's parsing assumption (or local depending on impl)
	// detectors.ParseISO8601 uses UTC by default if no timezone specified or handles Z
	now := time.Now().UTC()
	past := now.Add(-10 * time.Second)
	// Using ISO8601 format: 2006-01-02T15:04:05.000Z
	tsStr := past.Format("2006-01-02T15:04:05.000Z")
	// The monitor looks for lines starting with digit for ISO8601/Nginx
	line := fmt.Sprintf("%s Test log line", tsStr)

	sourceName := "test_lag_source"
	source := &NamedMockSource{name: sourceName, content: line}

	// Use MockDetector that detects everything but doesn't implement TimestampExtractor
	// so it falls back to extractTimestamp which handles ISO8601
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Run the monitor
	// Since StopOnEOF is true, Start() will return after processing the content
	mon.Start()

	// Verify metric
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	found := false
	for _, mf := range mfs {
		if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
			for _, m := range mf.GetMetric() {
				hasSource := false
				for _, label := range m.GetLabel() {
					if label.GetName() == "source" && label.GetValue() == sourceName {
						hasSource = true
						break
					}
				}

				if hasSource {
					found = true
					// Check histogram count
					if m.GetHistogram().GetSampleCount() != 1 {
						t.Errorf("Expected sample count 1, got %d", m.GetHistogram().GetSampleCount())
					}
					// Check sum roughly
					// Lag should be roughly 10 seconds plus processing overhead
					sum := m.GetHistogram().GetSampleSum()
					if sum < 9.5 || sum > 15.0 { // Allow some slack for test execution time
						t.Errorf("Expected lag around 10s, got %f", sum)
					}
				}
			}
		}
	}

	if !found {
		t.Errorf("Metric sentrylogmon_monitor_lag_seconds not found for source %s", sourceName)
	}
}
