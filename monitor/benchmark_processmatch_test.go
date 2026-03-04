package monitor

import (
	"context"
	"strings"
	"testing"
	"github.com/angch/sentrylogmon/detectors"
	"github.com/getsentry/sentry-go"
)

func BenchmarkProcessMatch_Group(b *testing.B) {
	// Setup Sentry Mock
	transport := &MockTransport{}
	client, _ := sentry.NewClient(sentry.ClientOptions{
		Transport: transport,
	})
	hub := sentry.NewHub(client, sentry.NewScope())

	det, _ := detectors.NewGenericDetector("error")
	mon, err := New(context.Background(), &MockSource{content: ""}, det, nil, Options{})
	if err != nil {
		b.Fatalf("Failed to create monitor: %v", err)
	}
	mon.Hub = hub

	// Use a long line
	line := "[100.0] " + strings.Repeat("a", 100)
	lineBytes := []byte(line)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mon.processMatch(lineBytes)
	}
}
