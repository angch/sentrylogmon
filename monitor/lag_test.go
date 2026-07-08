package monitor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/metrics"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"golang.org/x/sys/unix"
)

type MockLagTimestampDetector struct {
	ts float64
}

func (m *MockLagTimestampDetector) Detect(line []byte) bool { return true }
func (m *MockLagTimestampDetector) ExtractTimestamp(line []byte) (float64, string, bool) {
	return m.ts, fmt.Sprintf("%f", m.ts), true
}

func TestMonitorLagMetric(t *testing.T) {
	now := float64(time.Now().UnixNano()) / 1e9
	absTs := now - 2.0

	source := &MockSource{content: "dummy"}
	monitor, _ := New(context.Background(), source, &MockLagTimestampDetector{ts: absTs}, nil, Options{Verbose: true})

	m, _ := metrics.MonitorLagSeconds.GetMetricWith(prometheus.Labels{"source": "mock"})
	var beforeMetric dto.Metric
	m.(prometheus.Metric).Write(&beforeMetric)
	beforeCount := beforeMetric.GetHistogram().GetSampleCount()

	monitor.processMatch([]byte("dummy"))

	var afterMetric dto.Metric
	m.(prometheus.Metric).Write(&afterMetric)
	afterCount := afterMetric.GetHistogram().GetSampleCount()

	if afterCount != beforeCount+1 {
		t.Errorf("Expected count to increase by 1, before %d after %d", beforeCount, afterCount)
	}
}

func TestMonitorLagUptime(t *testing.T) {
	var ts unix.Timespec
	unix.ClockGettime(unix.CLOCK_BOOTTIME, &ts)
	uptime := float64(ts.Sec) + float64(ts.Nsec)/1e9
	mockTs := uptime - 2.0

	source := &MockSource{content: "dummy"}
	monitor, _ := New(context.Background(), source, &MockLagTimestampDetector{ts: mockTs}, nil, Options{Verbose: true})

	m, _ := metrics.MonitorLagSeconds.GetMetricWith(prometheus.Labels{"source": "mock"})
	var beforeMetric dto.Metric
	m.(prometheus.Metric).Write(&beforeMetric)
	beforeCount := beforeMetric.GetHistogram().GetSampleCount()

	monitor.processMatch([]byte("dummy"))

	var afterMetric dto.Metric
	m.(prometheus.Metric).Write(&afterMetric)
	afterCount := afterMetric.GetHistogram().GetSampleCount()

	if afterCount != beforeCount+1 {
		t.Errorf("Expected count to increase by 1")
	}
}
