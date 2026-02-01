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

func (d *MockDetector) Detect(line []byte) bool {
	return true // Detect everything
}

// MockContextDetector implements detectors.Detector and detectors.ContextExtractor
type MockContextDetector struct{}

func (d *MockContextDetector) Detect(line []byte) bool { return true }
func (d *MockContextDetector) GetContext(line []byte) map[string]interface{} {
	return map[string]interface{}{"extracted_key": "extracted_value"}
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

func TestContextExtraction(t *testing.T) {
	// Setup Sentry Mock
	transport := &MockTransport{}
	err := sentry.Init(sentry.ClientOptions{
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	input := `[100.0] Line 1`
	source := &MockSource{content: input}
	detector := &MockContextDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	go mon.Start()

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Flush sentry
	sentry.Flush(time.Second)

	transport.mu.Lock()
	defer transport.mu.Unlock()

	if len(transport.events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(transport.events))
	} else {
		event := transport.events[0]
		// Check contexts
		if event.Contexts == nil {
			t.Errorf("Expected contexts, got nil")
		} else {
			if logData, ok := event.Contexts["Log Data"]; ok {
				if val, ok := logData["extracted_key"]; ok {
					if val != "extracted_value" {
						t.Errorf("Expected extracted_value, got %v", val)
					}
				} else {
					t.Errorf("Context missing extracted_key")
				}
			} else {
				t.Errorf("Context missing 'Log Data'")
			}
		}
	}
}

func TestMonitorExclusion(t *testing.T) {
	// Setup Sentry Mock
	transport := &MockTransport{}
	err := sentry.Init(sentry.ClientOptions{
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	// Input lines
	// Line 1: Should be excluded
	// Line 2: Should be kept
	input := `[100.0] Line 1 - ignore me
[101.0] Line 2 - keep me
[102.0] Line 3 - ignore me too
`
	source := &MockSource{content: input}
	detector := &MockDetector{} // Detects everything

	// Create monitor with exclude pattern
	mon, err := New(context.Background(), source, detector, nil, Options{ExcludePattern: "ignore me"})
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
		if count >= 1 {
			break
		}
		if time.Since(start) > 2*time.Second {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Flush sentry
	sentry.Flush(time.Second)

	transport.mu.Lock()
	defer transport.mu.Unlock()

	if len(transport.events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(transport.events))
		for i, e := range transport.events {
			t.Logf("Event %d: %s", i, e.Message)
		}
	} else {
		msg := transport.events[0].Message
		expected := "[101.0] Line 2 - keep me"
		if msg != expected {
			t.Errorf("Event content mismatch.\nExpected:\n%s\nGot:\n%s", expected, msg)
		}
	}
}

func TestRateLimiting(t *testing.T) {
	// Setup Sentry Mock
	transport := &MockTransport{}
	err := sentry.Init(sentry.ClientOptions{
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	// Input lines with large gaps to force flushing separate events
	// Burst 2, Window 1s.
	// 3 events generated quickly.
	input := `[100.0] Line 1
[110.0] Line 2
[120.0] Line 3
`
	source := &MockSource{content: input}
	detector := &MockDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{
		RateLimitBurst:  2,
		RateLimitWindow: "1s",
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

	// Expect exactly 2 events (Line 1 and Line 2), Line 3 should be dropped
	if len(transport.events) != 2 {
		t.Errorf("Expected 2 events (rate limited), got %d", len(transport.events))
		for i, e := range transport.events {
			t.Logf("Event %d: %s", i, e.Message)
		}
	}
}

// MockTransformerDetector implements detectors.Detector and detectors.MessageTransformer
type MockTransformerDetector struct {
	MockDetector
}

func (d *MockTransformerDetector) TransformMessage(line []byte) []byte {
	return []byte(strings.ReplaceAll(string(line), "foo", "bar"))
}

func TestMessageTransformation(t *testing.T) {
	// Setup Sentry Mock
	transport := &MockTransport{}
	err := sentry.Init(sentry.ClientOptions{
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("Failed to init sentry: %v", err)
	}

	input := "[100.0] foo something bar"
	source := &MockSource{content: input}
	detector := &MockTransformerDetector{}

	mon, err := New(context.Background(), source, detector, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	mon.StopOnEOF = true

	go mon.Start()

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Flush sentry
	sentry.Flush(time.Second)

	transport.mu.Lock()
	defer transport.mu.Unlock()

	if len(transport.events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(transport.events))
	} else {
		msg := transport.events[0].Message
		// "foo" -> "bar"
		expected := "[100.0] bar something bar"
		if msg != expected {
			t.Errorf("Event content mismatch.\nExpected:\n%s\nGot:\n%s", expected, msg)
		}
	}
}

func TestMonitorMultiTenancy(t *testing.T) {
	// Setup global Sentry to verify separation
	err := sentry.Init(sentry.ClientOptions{
		Dsn: "https://global@sentry.io/1",
	})
	if err != nil {
		t.Fatalf("Failed to init global sentry: %v", err)
	}

	// 1. Monitor with global config
	source1 := &MockSource{content: "line1"}
	det1 := &MockDetector{}
	mon1, err := New(context.Background(), source1, det1, nil, Options{})
	if err != nil {
		t.Fatalf("Failed to create mon1: %v", err)
	}

	// Verify mon1 uses global hub
	if mon1.Hub != sentry.CurrentHub() {
		t.Error("mon1.Hub should be global hub")
	}
	if mon1.Hub.Client().Options().Dsn != "https://global@sentry.io/1" {
		t.Errorf("mon1 DSN mismatch. Got %s", mon1.Hub.Client().Options().Dsn)
	}

	// 2. Monitor with custom DSN
	customDSN := "https://custom@sentry.io/2"
	source2 := &MockSource{content: "line2"}
	det2 := &MockDetector{}
	mon2, err := New(context.Background(), source2, det2, nil, Options{
		SentryDSN: customDSN,
	})
	if err != nil {
		t.Fatalf("Failed to create mon2: %v", err)
	}

	// Verify mon2 uses distinct hub
	if mon2.Hub == sentry.CurrentHub() {
		t.Error("mon2.Hub should NOT be global hub")
	}
	if mon2.Hub == mon1.Hub {
		t.Error("mon2.Hub should NOT be same as mon1.Hub")
	}

	// Verify mon2 DSN
	if mon2.Hub.Client().Options().Dsn != customDSN {
		t.Errorf("mon2 DSN mismatch. Expected %s, got %s", customDSN, mon2.Hub.Client().Options().Dsn)
	}
}
