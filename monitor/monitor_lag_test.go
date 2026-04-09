package monitor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/metrics"
	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// uniqueMockSource is a wrapper around MockSource to provide a unique name per test
type uniqueMockSource struct {
	MockSource
	name string
}

func (s *uniqueMockSource) Name() string {
	return s.name
}

func TestMonitorLag(t *testing.T) {
	err := sentry.Init(sentry.ClientOptions{
		Transport: &MockTransport{},
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	tests := []struct {
		name          string
		inputLine     string
		expectedCount int
		// Since we use time.Now() in the actual code, we can't test exact lag values easily
		// without mocking time.Now(). However, we can test that *a* value was recorded
		// for normal lag, and *no* value was recorded for negative lag (future timestamps).
	}{
		{
			name: "NormalLag",
			// Timestamp is exactly 10 seconds in the past from time.Now()
			// We format it as RFC3339 which is supported by extractTimestamp.
			// It should start with a digit so the parser picks it up as ISO8601.
			inputLine:     fmt.Sprintf("%s Error occurred", time.Now().Add(-10*time.Second).Format(time.RFC3339)),
			expectedCount: 1,
		},
		{
			name: "NegativeLag",
			// Timestamp is 10 seconds in the future
			inputLine:     fmt.Sprintf("%s Error occurred", time.Now().Add(10*time.Second).Format(time.RFC3339)),
			expectedCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Use unique source name to avoid metric contamination across tests
			sourceName := fmt.Sprintf("mock_%s", tc.name)
			source := &uniqueMockSource{
				MockSource: MockSource{content: tc.inputLine + "\n"},
				name:       sourceName,
			}
			detector := &MockDetector{}

			mon, err := New(context.Background(), source, detector, nil, Options{})
			if err != nil {
				t.Fatalf("Failed to create monitor: %v", err)
			}
			mon.StopOnEOF = true

			// Run monitor. This will process the line and update metrics.
			mon.Start()

			// Check metric count
			metric := metrics.MonitorLagSeconds.WithLabelValues(sourceName)

			// Extract the value from the histogram
			ch := make(chan prometheus.Metric, 1)
			metric.(prometheus.Collector).Collect(ch)
			m := <-ch

			pb := &dto.Metric{}
			m.Write(pb)

			var actualCount int
			if pb.Histogram != nil && pb.Histogram.SampleCount != nil {
				actualCount = int(*pb.Histogram.SampleCount)
			}

			if actualCount != tc.expectedCount {
				t.Errorf("Expected lag observation count %d, got %v", tc.expectedCount, actualCount)
			}
		})
	}
}
