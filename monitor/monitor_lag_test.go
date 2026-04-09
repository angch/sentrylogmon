package monitor

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_model/go"
)

// uniqueMockSource overrides MockSource's name for Prometheus isolation per subtest
type uniqueMockSource struct {
	content string
	name    string
}

func (s *uniqueMockSource) Name() string { return s.name }
func (s *uniqueMockSource) Stream() (io.Reader, error) {
	return strings.NewReader(s.content), nil
}
func (s *uniqueMockSource) Close() error { return nil }

func getGaugeValue(vec *prometheus.GaugeVec, labelName, labelValue string) (float64, error) {
	metric, err := vec.GetMetricWith(prometheus.Labels{labelName: labelValue})
	if err != nil {
		return 0, err
	}
	m := &io_prometheus_client.Metric{}
	err = metric.Write(m)
	if err != nil {
		return 0, err
	}
	if m.Gauge == nil {
		return 0, fmt.Errorf("metric is not a gauge")
	}
	return m.Gauge.GetValue(), nil
}

func TestMonitorLagSecondsMetric(t *testing.T) {
	now := time.Now()
	// Create an ISO8601 string from 5 seconds ago
	pastTime := now.Add(-5 * time.Second)
	// Truncate to match what parsing typically sees
	pastTimeStr := pastTime.UTC().Format("2006-01-02T15:04:05Z")
	// Make sure we format the line correctly (without square brackets for ISO8601 extractor)
	line := fmt.Sprintf("%s Error something went wrong\n", pastTimeStr)

	// Since prometheus uses a global registry by default, use a unique source name
	sourceName := "test_lag_source_1"
	source := &uniqueMockSource{
		content: line,
		name:    sourceName,
	}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	mon.Start() // this will process the log line

	// Metric should have been updated.
	val, err := getGaugeValue(metrics.MonitorLagSeconds, "source", sourceName)
	if err != nil {
		t.Fatalf("Failed to get metric value: %v", err)
	}

	// We expect the lag to be around 5.0 seconds. Allow some variance due to sub-second precision and execution time.
	// Since pastTimeStr formatting truncates fractional seconds, val can be slightly more than 5.
	if val < 4.9 || val > 6.0 {
		t.Errorf("Expected lag to be around 5.0 seconds, got %v", val)
	}
}

func TestMonitorLagSecondsMetricNegative(t *testing.T) {
	now := time.Now()
	// Create an ISO8601 string from 5 seconds in the future
	futureTime := now.Add(5 * time.Second)
	futureTimeStr := futureTime.UTC().Format("2006-01-02T15:04:05Z")
	line := fmt.Sprintf("%s Error future event\n", futureTimeStr)

	sourceName := "test_lag_source_2"
	source := &uniqueMockSource{
		content: line,
		name:    sourceName,
	}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	mon.Start()

	val, err := getGaugeValue(metrics.MonitorLagSeconds, "source", sourceName)
	if err != nil {
		t.Fatalf("Failed to get metric value: %v", err)
	}

	// Negative lag should be ignored, meaning the gauge remains 0
	if val != 0 {
		t.Errorf("Expected negative lag to be ignored (gauge=0), got %v", val)
	}
}
