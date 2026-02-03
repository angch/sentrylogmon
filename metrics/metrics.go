package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ProcessedLinesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentrylogmon_processed_lines_total",
			Help: "Total number of log lines processed.",
		},
		[]string{"source"},
	)

	IssuesDetectedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentrylogmon_issues_detected_total",
			Help: "Total number of issues detected by the monitor.",
		},
		[]string{"source"},
	)

	SentryEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentrylogmon_sentry_events_total",
			Help: "Total number of events sent to Sentry.",
		},
		[]string{"source", "status"},
	)

	LastActivityTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sentrylogmon_last_activity_timestamp_seconds",
			Help: "Unix timestamp of the last activity (read line) for the source.",
		},
		[]string{"source"},
	)
)

func init() {
	prometheus.MustRegister(ProcessedLinesTotal)
	prometheus.MustRegister(IssuesDetectedTotal)
	prometheus.MustRegister(SentryEventsTotal)
	prometheus.MustRegister(LastActivityTimestamp)
}
