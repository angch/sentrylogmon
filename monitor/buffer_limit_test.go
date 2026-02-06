package monitor

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

func TestBufferByteLimit(t *testing.T) {
	// Setup Sentry Mock
	transport := &MockTransport{}
	err := sentry.Init(sentry.ClientOptions{
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	// MaxBufferBytes is 256KB.
	// We want to verify that we split events if they exceed this.

	// Create a line that is ~100KB.
	// 2 lines = ~200KB.
	// 3 lines = ~300KB -> Should trigger flush before 3rd line.

	lineSize := 100 * 1024 // 100KB
	lineContent := strings.Repeat("a", lineSize)

	// Input: 3 lines with same timestamp.
	// Without size limit, they would be grouped into one event (diff < 5.0s).
	// With size limit, they should be split.
	input := "[100.0] " + lineContent + "\n" +
			 "[100.0] " + lineContent + "\n" +
			 "[100.0] " + lineContent + "\n"

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
		if count >= 2 { // We expect split
			break
		}
		if time.Since(start) > 5*time.Second {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Flush sentry
	sentry.Flush(time.Second)

	transport.mu.Lock()
	defer transport.mu.Unlock()

	if len(transport.events) < 2 {
		t.Errorf("Expected at least 2 events (split due to size), got %d. Total size likely exceeded limit if grouped.", len(transport.events))
		if len(transport.events) == 1 {
			t.Logf("Event 1 size: %d", len(transport.events[0].Message))
		}
	} else {
		// Verify approximate sizes
		msg1 := transport.events[0].Message
		msg2 := transport.events[1].Message

		// Event 1 should have 2 lines (~200KB)
		// Check length. 2 lines + timestamps ~ 200KB + overhead
		if len(msg1) < 200000 {
			t.Errorf("Event 1 too small: %d (expected ~200KB)", len(msg1))
		}
		// Should roughly be 2 * (100KB + header)
		// It shouldn't contain 3 lines
		if len(msg1) > 250000 {
			t.Errorf("Event 1 too large: %d (might contain 3 lines?)", len(msg1))
		}

		// Event 2 should have 1 line (~100KB)
		if len(msg2) < 100000 {
			t.Errorf("Event 2 too small: %d", len(msg2))
		}
	}
}
