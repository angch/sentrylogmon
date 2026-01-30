package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

func TestMonitorGrouping_ISO8601(t *testing.T) {
	// Setup Sentry Mock
	transport := &MockTransport{}
	err := sentry.Init(sentry.ClientOptions{
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	// Input lines with ISO8601 timestamps
	// Group 1: 10:00:00, 10:00:01 (diff 1s <= 5s)
	// Group 2: 10:00:06 (diff 6s from 10:00:00 > 5s) -> New group
	// Group 2: 10:00:07 (diff 1s from 10:00:06 <= 5s)
	input := `2023-10-27T10:00:00Z Line 1
2023-10-27T10:00:01Z Line 2
2023-10-27T10:00:06Z Line 3
2023-10-27T10:00:07Z Line 4
`
	source := &MockSource{content: input}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	go mon.Start()

	// Wait for processing with timeout
	start := time.Now()
	for {
		transport.mu.Lock()
		count := len(transport.events)
		transport.mu.Unlock()
		if count >= 2 {
			break
		}
		if time.Since(start) > 2*time.Second {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Flush sentry to ensure events are sent to mock transport
	sentry.Flush(time.Second)

	transport.mu.Lock()
	defer transport.mu.Unlock()

	if len(transport.events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(transport.events))
		for i, e := range transport.events {
			t.Logf("Event %d: %s", i, e.Message)
		}
	} else {
		// Verify content
		msg1 := transport.events[0].Message
		expected1 := "2023-10-27T10:00:00Z Line 1\n2023-10-27T10:00:01Z Line 2"
		if msg1 != expected1 {
			t.Errorf("Event 1 content mismatch.\nExpected:\n%s\nGot:\n%s", expected1, msg1)
		}

		msg2 := transport.events[1].Message
		expected2 := "2023-10-27T10:00:06Z Line 3\n2023-10-27T10:00:07Z Line 4"
		if msg2 != expected2 {
			t.Errorf("Event 2 content mismatch.\nExpected:\n%s\nGot:\n%s", expected2, msg2)
		}
	}
}
