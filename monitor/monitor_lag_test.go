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

type uniqueMockSource struct {
	MockSource
	name string
}

func (s *uniqueMockSource) Name() string { return s.name }

func TestMonitorLagAbsolute(t *testing.T) {
	// Setup Sentry Mock
	transport := &MockTransport{}
	err := sentry.Init(sentry.ClientOptions{
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	metrics.MonitorLag.Reset()

	// Use a time 5 seconds in the past
	logTime := time.Now().Add(-5 * time.Second)
	logTimeStr := logTime.Format(time.RFC3339)

	input := fmt.Sprintf("%s absolute error", logTimeStr)
	source := &uniqueMockSource{name: "mock_lag_absolute_test", MockSource: MockSource{content: input}}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	go mon.Start()

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	m := metrics.MonitorLag.With(prometheus.Labels{"source": source.Name()})

	var metric dto.Metric
	if err := m.Write(&metric); err != nil {
		t.Fatalf("Failed to read metric: %v", err)
	}

	if metric.Gauge == nil {
		t.Fatal("Gauge is nil")
	}

	val := metric.Gauge.GetValue()

	// Expect lag to be approximately 5.0 seconds. We allow a range of 5.0 to 6.0
	// because time.RFC3339 truncates nanoseconds which can add up to 1 second of difference
	if val < 5.0 || val > 6.0 {
		t.Errorf("Expected lag to be between 5.0 and 6.0, got %f", val)
	}
}

func TestMonitorLagRelative(t *testing.T) {
	// Setup Sentry Mock
	transport := &MockTransport{}
	err := sentry.Init(sentry.ClientOptions{
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	metrics.MonitorLag.Reset()

	// dmesg timestamp relative
	// since tests don't have a reliable boot time across different environments without mocking `host.Uptime()`,
	// we will use absolute time with Dmesg format to verify processMatch behavior directly.
	input := "[ 123.456] relative error"
	source := &uniqueMockSource{name: "mock_lag_relative_test", MockSource: MockSource{content: input}}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	// DmesgDetector will be used internally if not explicitly set, but here we provide a MockDetector.
	// We'll call extractTimestamp directly on the mock input inside Start/processMatch.
	// Since extractTimestamp will parse "[ 123.456]" as 123.456 (relative to start of epoch if bootTime is not applied, or bootTime + 123.456)
	// it will result in a very large lag (now - 123.456). We just verify the gauge is set to *something* greater than 0.

	go mon.Start()

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	m := metrics.MonitorLag.With(prometheus.Labels{"source": source.Name()})

	var metric dto.Metric
	if err := m.Write(&metric); err != nil {
		t.Fatalf("Failed to read metric: %v", err)
	}

	if metric.Gauge == nil {
		t.Fatal("Gauge is nil")
	}

	val := metric.Gauge.GetValue()

	if val <= 0 {
		t.Errorf("Expected lag to be > 0, got %f", val)
	}
}

func TestMonitorLagNegative(t *testing.T) {
	// Setup Sentry Mock
	transport := &MockTransport{}
	err := sentry.Init(sentry.ClientOptions{
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	metrics.MonitorLag.Reset()

	// Use a time 5 seconds in the future
	logTime := time.Now().Add(5 * time.Second)
	logTimeStr := logTime.Format(time.RFC3339)

	input := fmt.Sprintf("%s future error", logTimeStr)
	source := &uniqueMockSource{name: "mock_lag_negative_test", MockSource: MockSource{content: input}}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	go mon.Start()

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	m := metrics.MonitorLag.With(prometheus.Labels{"source": source.Name()})

	var metric dto.Metric
	if err := m.Write(&metric); err != nil {
		// Metric not found is expected if it was never set (negative lag)
		return
	}

	if metric.Gauge != nil {
		val := metric.Gauge.GetValue()
		// Uninitialized gauge has value 0
		if val != 0 {
			t.Errorf("Expected lag to not be set (or be 0), got %f", val)
		}
	}
}
