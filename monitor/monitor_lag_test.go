package monitor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus"
	// The client_model package is needed to inspect metric values
	// It is usually available via prometheus/client_model/go
	// However, client_golang might expose DTOs directly.
	// client_golang/prometheus imports "github.com/prometheus/client_model/go" as dto.
	// But we can't access internal imports.
	// Checking go.mod, it requires github.com/prometheus/client_model v0.6.2
)

// Define NamedMockSource locally since we can't modify MockSource in monitor_test.go easily
// and we want to ensure unique source name for metrics.
type NamedMockSource struct {
	MockSource
	nameOverride string
}

func (s *NamedMockSource) Name() string {
	return s.nameOverride
}

func TestMonitorLagMetric(t *testing.T) {
	// Setup Sentry Mock
	transport := &MockTransport{}
	err := sentry.Init(sentry.ClientOptions{
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	sourceName := "mock_lag_test"

	// Current time - 10 seconds
	// We use UTC to ensure consistent formatting/parsing
	pastTime := time.Now().UTC().Add(-10 * time.Second)

	// Format as RFC3339 which is supported by extractTimestamp -> detectors.ParseISO8601
	tsStr := pastTime.Format(time.RFC3339)
	input := fmt.Sprintf("%s Test Log Line", tsStr)

	// Initialize NamedMockSource
	// MockSource is defined in monitor_test.go in the same package
	source := &NamedMockSource{
		MockSource: MockSource{content: input},
		nameOverride: sourceName,
	}

	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Run monitor
	mon.Start()

	// Verify metric
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	var found bool
	var count uint64
	var sum float64

	for _, mf := range mfs {
		if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
			for _, m := range mf.GetMetric() {
				// Check labels
				for _, label := range m.GetLabel() {
					if label.GetName() == "source" && label.GetValue() == sourceName {
						found = true
						if h := m.GetHistogram(); h != nil {
							count = h.GetSampleCount()
							sum = h.GetSampleSum()
						}
					}
				}
			}
		}
	}

	if !found {
		t.Fatalf("Metric sentrylogmon_monitor_lag_seconds for source %s not found", sourceName)
	}

	if count != 1 {
		t.Errorf("Expected 1 observation, got %d", count)
	}

	// Lag should be around 10 seconds.
	if sum < 9.0 {
		t.Errorf("Expected lag >= 9.0, got %f", sum)
	}
	if sum > 15.0 {
		t.Errorf("Expected lag < 15.0, got %f", sum)
	}
}
