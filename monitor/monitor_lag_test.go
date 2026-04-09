package monitor

import (
	"bytes"
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

type uniqueMockSource struct {
	name   string
	reader io.Reader
}

func (m *uniqueMockSource) Name() string { return m.name }
func (m *uniqueMockSource) Stream() (io.Reader, error) {
	return m.reader, nil
}
func (m *uniqueMockSource) Close() error { return nil }

func TestMonitorLag(t *testing.T) {
	// 1. Setup mock source with a unique name to isolate the Prometheus metric state
	sourceName := "test_lag_source_1"

	// Create a log line from 5 seconds ago
	pastTime := time.Now().Add(-5 * time.Second)
	// Truncate to match what extractors often do or what's simpler for exact math
	pastTime = pastTime.Truncate(time.Second)
	logLine := fmt.Sprintf("%s Some error message\n", pastTime.Format(time.RFC3339))

	source := &uniqueMockSource{
		name:   sourceName,
		reader: bytes.NewReader([]byte(logLine)),
	}

	d, _ := detectors.NewGenericDetector("error")

	// 2. Initialize monitor
	opts := Options{Verbose: true}
	m, err := New(context.Background(), source, d, nil, opts)
	m.StopOnEOF = true
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	// 3. Process logs
	m.Start()

	// 4. Verify metric
	gaugeVec := metrics.MonitorLagSeconds
	metric, err := gaugeVec.GetMetricWithLabelValues(sourceName)
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}
	val := testutil.ToFloat64(metric)

	// Since the log was created 5 seconds ago, the lag should be approximately 5.
	// We allow a small tolerance (e.g. 5.0 to 6.0) as noted in the memory due to RFC3339 truncation and test execution time.
	if val < 5.0 || val > 6.0 {
		t.Errorf("Expected MonitorLagSeconds to be between 5.0 and 6.0, got %f", val)
	}
}

func TestMonitorLag_NegativeLagIgnored(t *testing.T) {
	// 1. Setup mock source with a unique name to isolate the Prometheus metric state
	sourceName := "test_lag_source_2"

	// Create a log line from 5 seconds in the FUTURE
	futureTime := time.Now().Add(5 * time.Second)
	logLine := fmt.Sprintf("%s Another error message\n", futureTime.Format(time.RFC3339))

	source := &uniqueMockSource{
		name:   sourceName,
		reader: strings.NewReader(logLine),
	}

	d, _ := detectors.NewGenericDetector("error")

	// 2. Initialize monitor
	opts := Options{}
	m, err := New(context.Background(), source, d, nil, opts)
	m.StopOnEOF = true
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	// 3. Process logs
	m.Start()

	// 4. Verify metric
	gaugeVec := metrics.MonitorLagSeconds
	metric, err := gaugeVec.GetMetricWithLabelValues(sourceName)

	// Since negative lag is ignored, the gauge shouldn't be initialized (not found) or should be 0.
	if err == nil {
		val := testutil.ToFloat64(metric)
		if val != 0 {
			t.Errorf("Expected MonitorLagSeconds to be ignored (uninitialized or 0) for negative lag, got %f", val)
		}
	}
}
