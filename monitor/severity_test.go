package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

func TestMonitorSyslogSeverity(t *testing.T) {
	// Setup Sentry Mock
	transport := &MockTransport{}
	err := sentry.Init(sentry.ClientOptions{
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	// Case 1: Severity 1 (Alert) -> Fatal
	// Case 2: Severity 3 (Error) -> Error
	// Case 3: Severity 6 (Info) -> Info
	// Note: We use distinct timestamps to prevent grouping if buffer logic triggers.
	// But FlushInterval is 5s. We can force wait.
	// Or we can just run separate tests. Let's run separate subtests or just one flow.

	testCases := []struct {
		name          string
		input         string
		expectedLevel sentry.Level
	}{
		{
			name:          "Severity Alert (1) -> Fatal",
			input:         "<9>Oct 11 22:14:15 myhost myprogram[123]: Alert message", // Facility 1 (8), Severity 1 -> 8+1=9
			expectedLevel: sentry.LevelFatal,
		},
		{
			name:          "Severity Error (3) -> Error",
			input:         "<11>Oct 11 22:14:16 myhost myprogram[123]: Error message", // Facility 1 (8), Severity 3 -> 8+3=11
			expectedLevel: sentry.LevelError,
		},
		{
			name:          "Severity Info (6) -> Info",
			input:         "<14>Oct 11 22:14:17 myhost myprogram[123]: Info message", // Facility 1 (8), Severity 6 -> 8+6=14
			expectedLevel: sentry.LevelInfo,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset transport events
			transport.mu.Lock()
			transport.events = nil
			transport.mu.Unlock()

			source := &MockSource{content: tc.input}
			detector := &MockDetector{}

			mon, err := New(context.Background(), source, detector, nil, Options{})
			if err != nil {
				t.Fatalf("Failed to create monitor: %v", err)
			}
			mon.StopOnEOF = true

			go mon.Start()

			// Wait for processing
			// We can poll transport
			start := time.Now()
			found := false
			for time.Since(start) < 2*time.Second {
				transport.mu.Lock()
				count := len(transport.events)
				transport.mu.Unlock()
				if count > 0 {
					found = true
					break
				}
				time.Sleep(10 * time.Millisecond)
			}

			if !found {
				t.Fatalf("Timeout waiting for event")
			}

			// Flush sentry
			sentry.Flush(time.Second)

			transport.mu.Lock()
			defer transport.mu.Unlock()

			if len(transport.events) != 1 {
				t.Fatalf("Expected 1 event, got %d", len(transport.events))
			}

			event := transport.events[0]
			if event.Level != tc.expectedLevel {
				t.Errorf("Expected level %s, got %s", tc.expectedLevel, event.Level)
			}
		})
	}
}
