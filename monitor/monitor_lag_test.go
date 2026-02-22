package monitor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestMonitorLagMetric(t *testing.T) {
	// Setup Sentry Mock
	transport := &MockTransport{}
	err := sentry.Init(sentry.ClientOptions{
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	// Create a unique source name to avoid collision in global registry
	sourceName := "mock_lag_test"

	// Create a log line with a timestamp 2 seconds in the past.
	// We use ISO8601/RFC3339 format to ensure it's treated as a Unix timestamp.
	now := time.Now()
	past := now.Add(-2 * time.Second)
	inputTimestamp := past.Format(time.RFC3339Nano)
	input := fmt.Sprintf("%s Test message", inputTimestamp)

	// We need to override the Name() method of MockSource?
	// MockSource.Name() returns "mock".
	// We need "mock_lag_test".
	// MockSource definition:
	// func (s *MockSource) Name() string { return "mock" }
	// I cannot override it easily unless I define a new type or struct embedding.

	sourceUnique := &UniqueMockSource{
		MockSource: MockSource{content: input},
		name:       sourceName,
	}

	detector := &MockDetector{}

	mon, err := New(context.Background(), sourceUnique, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// Run monitor synchronously as MockSource yields immediately
	mon.Start()

	// Verify metric
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	var metricFamily *dto.MetricFamily
	for _, mf := range mfs {
		if mf.GetName() == "sentrylogmon_monitor_lag_seconds" {
			metricFamily = mf
			break
		}
	}

	if metricFamily == nil {
		t.Fatalf("Metric sentrylogmon_monitor_lag_seconds not found")
	}

	var metric *dto.Metric
	for _, m := range metricFamily.GetMetric() {
		for _, label := range m.GetLabel() {
			if label.GetName() == "source" && label.GetValue() == sourceName {
				metric = m
				break
			}
		}
		if metric != nil {
			break
		}
	}

	if metric == nil {
		t.Fatalf("Metric for source %s not found", sourceName)
	}

	// Verify Histogram
	hist := metric.GetHistogram()
	if hist.GetSampleCount() != 1 {
		t.Errorf("Expected sample count 1, got %d", hist.GetSampleCount())
	}

	sum := hist.GetSampleSum()
	// Expected lag is ~2.0 seconds. Allow some delta for execution time.
	// It should be at least 2.0.
	if sum < 2.0 {
		t.Errorf("Expected lag >= 2.0, got %f", sum)
	}
	if sum > 3.0 { // Allow 1s overhead
		t.Errorf("Expected lag < 3.0, got %f", sum)
	}
}

type UniqueMockSource struct {
	MockSource
	name string
}

func (s *UniqueMockSource) Name() string {
	return s.name
}
