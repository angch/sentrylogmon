package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/detectors"
	"github.com/getsentry/sentry-go"
)

func TestJsonSeverity(t *testing.T) {
	// Setup Sentry Mock
	transport := &MockTransport{}
	err := sentry.Init(sentry.ClientOptions{
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	testCases := []struct {
		name          string
		input         string
		expectedLevel sentry.Level
	}{
		{
			name:          "JSON Error",
			input:         `{"level": "error", "msg": "failed task"}`,
			expectedLevel: sentry.LevelError,
		},
		{
			name:          "JSON Warning",
			input:         `{"severity": "warning", "msg": "disk space low"}`,
			expectedLevel: sentry.LevelWarning,
		},
		{
			name:          "JSON Info",
			input:         `{"log_level": "info", "msg": "service started"}`,
			expectedLevel: sentry.LevelInfo,
		},
		{
			name:          "JSON Debug",
			input:         `{"level": "debug", "msg": "variable x=1"}`,
			expectedLevel: sentry.LevelDebug,
		},
		{
			name:          "JSON Fatal",
			input:         `{"level": "fatal", "msg": "system crash"}`,
			expectedLevel: sentry.LevelFatal,
		},
		{
			name:          "JSON Critical",
			input:         `{"level": "critical", "msg": "db down"}`,
			expectedLevel: sentry.LevelFatal,
		},
		{
			name:          "JSON Unknown Level",
			input:         `{"level": "unknown", "msg": "what is this"}`,
			expectedLevel: sentry.LevelInfo, // Default fallback (or empty which implies Info/Default)
		},
		{
			name:          "JSON No Level",
			input:         `{"msg": "just a message"}`,
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
			// Use JsonDetector. Pattern doesn't matter much for basic detection if we feed valid JSON,
			// but we need a valid pattern to construct it.
			// The detector checks if the field exists and matches the regex.
			// Let's use a pattern that matches everything in "msg" to ensure detection.
			detector, err := detectors.NewJsonDetector("msg:.*")
			if err != nil {
				t.Fatalf("Failed to create detector: %v", err)
			}

			mon, err := New(context.Background(), source, detector, nil, Options{})
			if err != nil {
				t.Fatalf("Failed to create monitor: %v", err)
			}
			mon.StopOnEOF = true

			go mon.Start()

			// Wait for processing
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
			// Sentry Go SDK default level is info if not set.
			// If we expect Info, it matches the default.
			// If we expect Error but get Info, it fails.

			// Normalize empty level to Info for comparison
			actualLevel := event.Level
			if actualLevel == "" {
				actualLevel = sentry.LevelInfo
			}

			if actualLevel != tc.expectedLevel {
				t.Errorf("Expected level %s, got %s", tc.expectedLevel, actualLevel)
			}
		})
	}
}
