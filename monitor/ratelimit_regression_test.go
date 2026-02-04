package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

func TestRateLimitingDefaultWindow(t *testing.T) {
	// Setup Sentry Mock
	transport := &MockTransport{}
	err := sentry.Init(sentry.ClientOptions{
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	// Burst 2, NO Window set (should default to 1s)
	// Input lines come in fast.
	input := `[100.0] Line 1
[110.0] Line 2
[120.0] Line 3
[130.0] Line 4
[140.0] Line 5
`
	source := &MockSource{content: input}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{
		RateLimitBurst:  2,
		RateLimitWindow: "", // Intentionally empty to trigger default
	})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	go mon.Start()

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Flush sentry
	sentry.Flush(time.Second)

	transport.mu.Lock()
	defer transport.mu.Unlock()

	// With default window of 1s and burst 2, we expect only the first 2 events.
	if len(transport.events) != 2 {
		t.Errorf("Expected 2 events (rate limited), got %d", len(transport.events))
	}
}
