package monitor

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/detectors"
	"github.com/angch/sentrylogmon/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// uniqueMockSource wraps strings.Reader and gives it a unique name
// for Prometheus registry isolation.
type uniqueMockSource struct {
	*strings.Reader
	name string
}

func (m *uniqueMockSource) Stream() (io.Reader, error) {
	return m.Reader, nil
}

func (m *uniqueMockSource) Close() error {
	return nil
}

func (m *uniqueMockSource) Name() string {
	return m.name
}

func TestMonitorLagMetric(t *testing.T) {
	tests := []struct {
		name         string
		logTimeFunc  func() time.Time
		logFormatter func(t time.Time) string
		expectedMin  float64
		expectedMax  float64
		shouldBeSet  bool
	}{
		{
			name: "Recent log",
			logTimeFunc: func() time.Time {
				return time.Now().Add(-5 * time.Second)
			},
			logFormatter: func(t time.Time) string {
				// We don't use square brackets here to avoid misidentification as Nginx format.
				// Format as: YYYY-MM-DDTHH:mm:ssZ
				return t.Format(time.RFC3339) + " error something went wrong"
			},
			expectedMin: 5.0,
			expectedMax: 6.0, // Allow up to 1 second variance due to RFC3339 truncation
			shouldBeSet: true,
		},
		{
			name: "Future log (negative lag)",
			logTimeFunc: func() time.Time {
				return time.Now().Add(5 * time.Second)
			},
			logFormatter: func(t time.Time) string {
				return t.Format(time.RFC3339) + " error something went wrong"
			},
			expectedMin: 0,
			expectedMax: 0,
			shouldBeSet: false, // Lag should not be updated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logTime := tt.logTimeFunc()
			logLine := tt.logFormatter(logTime)

			sourceName := fmt.Sprintf("test_source_%s_%d", strings.ReplaceAll(tt.name, " ", "_"), time.Now().UnixNano())
			source := &uniqueMockSource{
				Reader: strings.NewReader(logLine + "\n"),
				name:   sourceName,
			}

			// Use the GenericDetector to catch "error"
			detector, err := detectors.NewGenericDetector("(?i)(error)")
			if err != nil {
				t.Fatalf("Failed to create detector: %v", err)
			}

			mon, err := New(context.Background(), source, detector, nil, Options{})
			if err != nil {
				t.Fatalf("Failed to create monitor: %v", err)
			}

			// Process the log line
			mon.processMatch([]byte(logLine))

			// Check the metric
			val := testutil.ToFloat64(metrics.MonitorLagSeconds.WithLabelValues(sourceName))

			if tt.shouldBeSet {
				if val < tt.expectedMin || val > tt.expectedMax {
					t.Errorf("Expected lag between %v and %v, got %v", tt.expectedMin, tt.expectedMax, val)
				}
			} else {
				if val != 0 {
					t.Errorf("Expected lag metric to be 0 (unset/ignored), got %v", val)
				}
			}
		})
	}
}
