package monitor

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

// MockPipeSource implements sources.LogSource using io.Pipe
type MockPipeSource struct {
	reader *io.PipeReader
	writer *io.PipeWriter
}

func NewMockPipeSource() *MockPipeSource {
	pr, pw := io.Pipe()
	return &MockPipeSource{
		reader: pr,
		writer: pw,
	}
}

func (s *MockPipeSource) Name() string { return "mock_pipe" }
func (s *MockPipeSource) Stream() (io.Reader, error) {
	return s.reader, nil
}
func (s *MockPipeSource) Close() error {
	return s.writer.Close()
}
func (s *MockPipeSource) Write(p []byte) (n int, err error) {
	return s.writer.Write(p)
}

func TestInactivityAlert(t *testing.T) {
	// Setup Sentry Mock
	transport := &MockTransport{}
	err := sentry.Init(sentry.ClientOptions{
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	source := NewMockPipeSource()
	detector := &MockDetector{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Short inactivity duration for testing
	mon, err := New(ctx, source, detector, nil, Options{
		MaxInactivity: "200ms",
		Verbose:       true, // Enable verbose to see logs if needed
	})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	go mon.Start()

	// 1. Initial Activity
	source.Write([]byte("Line 1\n"))

	// Wait a bit, but less than timeout
	time.Sleep(50 * time.Millisecond)

	// Verify no alerts yet
	transport.mu.Lock()
	if len(transport.events) != 0 {
		t.Errorf("Expected 0 events, got %d", len(transport.events))
	}
	transport.mu.Unlock()

	// 2. Wait for Inactivity Alert
	// Timeout is 200ms. Ticker is 100ms.
	// We need to wait enough for Ticker to fire AFTER 200ms of silence.
	time.Sleep(300 * time.Millisecond)

	transport.mu.Lock()
	foundAlert := false
	for _, e := range transport.events {
		if val, ok := e.Tags["alert_type"]; ok && val == "inactivity" {
			if e.Level == sentry.LevelWarning {
				foundAlert = true
				break
			}
		}
	}
	if !foundAlert {
		t.Errorf("Expected inactivity alert (Warning), got %d events", len(transport.events))
		for i, e := range transport.events {
			t.Logf("Event %d: Level=%s Tags=%v Msg=%s", i, e.Level, e.Tags, e.Message)
		}
	}
	transport.mu.Unlock()

	// 3. Resume Activity
	source.Write([]byte("Line 2\n"))

	// Wait for watchdog to pick up activity (next tick)
	time.Sleep(200 * time.Millisecond)

	transport.mu.Lock()
	foundRecovery := false
	for _, e := range transport.events {
		if val, ok := e.Tags["alert_type"]; ok && val == "inactivity" {
			if e.Level == sentry.LevelInfo {
				foundRecovery = true
				break
			}
		}
	}
	if !foundRecovery {
		t.Errorf("Expected recovery alert (Info), got %d events", len(transport.events))
		for i, e := range transport.events {
			t.Logf("Event %d: Level=%s Tags=%v Msg=%s", i, e.Level, e.Tags, e.Message)
		}
	}
	transport.mu.Unlock()

	source.Close()
}
