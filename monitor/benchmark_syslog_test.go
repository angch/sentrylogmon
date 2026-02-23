package monitor

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/detectors"
	"github.com/getsentry/sentry-go"
)

func BenchmarkProcessMatchSyslog(b *testing.B) {
	// Setup Sentry Mock
	transport := &MockTransport{}
	client, _ := sentry.NewClient(sentry.ClientOptions{
		Transport: transport,
	})
	hub := sentry.NewHub(client, sentry.NewScope())

	// Generic detector so it falls back to extractTimestamp
	det, _ := detectors.NewGenericDetector("error")
	mon, err := New(context.Background(), &MockSource{content: ""}, det, nil, Options{})
	if err != nil {
		b.Fatalf("Failed to create monitor: %v", err)
	}
	mon.Hub = hub

	// Syslog line (RFC 3164) format.
	// Starts with Oct 27 10:00:00
	line := "Oct 27 10:00:00 myhost myprocess: error message " + strings.Repeat("a", 100)
	lineBytes := []byte(line)

	b.ResetTimer()
	b.ReportAllocs()
	now := time.Now()
	for i := 0; i < b.N; i++ {
		mon.processMatch(lineBytes, now)
	}
}
