package monitor

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

// MockSource implements sources.LogSource
type MockSource struct {
	content string
}

func (s *MockSource) Name() string { return "mock" }
func (s *MockSource) Stream() (io.Reader, error) {
	return strings.NewReader(s.content), nil
}
func (s *MockSource) Close() error { return nil }

// MockDetector implements detectors.Detector (implicitly)
type MockDetector struct{}

func (d *MockDetector) Detect(line string) bool {
	return true // Detect everything
}

// MockTransport captures Sentry events
type MockTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (t *MockTransport) Configure(options sentry.ClientOptions) {}
func (t *MockTransport) SendEvent(event *sentry.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, event)
}
func (t *MockTransport) Flush(timeout time.Duration) bool          { return true }
func (t *MockTransport) FlushWithContext(ctx context.Context) bool { return true }
func (t *MockTransport) Close()                                    {}

func TestMonitorGrouping(t *testing.T) {
	// Setup Sentry Mock
	transport := &MockTransport{}
	err := sentry.Init(sentry.ClientOptions{
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	// Input lines
	// Group 1: 100.0, 101.0 (diff 1.0 < 5.0)
	// Group 2: 106.0 (diff 6.0 from 100.0 > 5.0) -> New group
	// Group 2: 107.0 (diff 1.0 from 106.0 < 5.0)
	input := `[100.0] Line 1
[101.0] Line 2
[106.0] Line 3
[107.0] Line 4
`
	source := &MockSource{content: input}
	detector := &MockDetector{}

	mon, err := New(source, detector, nil, false)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	mon.Start()

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
		expected1 := "[100.0] Line 1\n[101.0] Line 2"
		if msg1 != expected1 {
			t.Errorf("Event 1 content mismatch.\nExpected:\n%s\nGot:\n%s", expected1, msg1)
		}

		msg2 := transport.events[1].Message
		expected2 := "[106.0] Line 3\n[107.0] Line 4"
		if msg2 != expected2 {
			t.Errorf("Event 2 content mismatch.\nExpected:\n%s\nGot:\n%s", expected2, msg2)
		}
	}
}
