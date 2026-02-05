package detectors

import (
	"sync"
	"testing"
)

func TestJsonDetector_Detect(t *testing.T) {
	d, err := NewJsonDetector("level:error")
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Match",
			input:    `{"level":"error", "msg":"test"}`,
			expected: true,
		},
		{
			name:     "No Match (Value)",
			input:    `{"level":"info", "msg":"test"}`,
			expected: false,
		},
		{
			name:     "No Match (Field missing)",
			input:    `{"msg":"test"}`,
			expected: false,
		},
		{
			name:     "Invalid JSON",
			input:    `{level:"error"}`,
			expected: false,
		},
		{
			name:     "Nested Match",
			input:    `{"level":"error", "details":{"code":500}}`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := d.Detect([]byte(tt.input)); got != tt.expected {
				t.Errorf("Detect() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestJsonDetector_GetContext(t *testing.T) {
	d, _ := NewJsonDetector("level:error")
	input := []byte(`{"level":"error", "foo":"bar"}`)

	// 1. Without Detect (Should parse fresh)
	ctx := d.GetContext(input)
	if ctx == nil {
		t.Fatal("Expected context, got nil")
	}
	if ctx["foo"] != "bar" {
		t.Errorf("Expected foo=bar, got %v", ctx["foo"])
	}

	// 2. With Detect (Should use cache)
	if !d.Detect(input) {
		t.Fatal("Expected Detect to return true")
	}
	ctx = d.GetContext(input)
	if ctx == nil {
		t.Fatal("Expected context, got nil")
	}
	if ctx["foo"] != "bar" {
		t.Errorf("Expected foo=bar, got %v", ctx["foo"])
	}
}

func TestJsonDetector_ExtractTimestamp(t *testing.T) {
	d, _ := NewJsonDetector("level:error")

	tests := []struct {
		name       string
		input      string
		expectedOK bool
		expectedTS float64 // Approximate
	}{
		{
			name:       "RFC3339",
			input:      `{"time":"2023-10-27T10:00:00Z"}`,
			expectedOK: true,
			expectedTS: 1698400800,
		},
		{
			name:       "Unix Seconds",
			input:      `{"ts":1698400800}`,
			expectedOK: true,
			expectedTS: 1698400800,
		},
		{
			name:       "Unix Milliseconds",
			input:      `{"timestamp":1698400800000}`,
			expectedOK: true,
			expectedTS: 1698400800,
		},
		{
			name:       "No Timestamp",
			input:      `{"msg":"hello"}`,
			expectedOK: false,
		},
		{
			name:       "Invalid Format",
			input:      `{"time":"invalid"}`,
			expectedOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pre-populate cache via Detect if possible, or just call ExtractTimestamp directly
			// ExtractTimestamp handles both cached and non-cached.
			// Let's test non-cached first
			ts, _, ok := d.ExtractTimestamp([]byte(tt.input))
			if ok != tt.expectedOK {
				t.Errorf("ExtractTimestamp() ok = %v, want %v", ok, tt.expectedOK)
			}
			if ok && tt.expectedOK {
				if ts != tt.expectedTS {
					t.Errorf("ExtractTimestamp() ts = %f, want %f", ts, tt.expectedTS)
				}
			}
		})
	}
}

func TestJsonDetector_CacheConsistency(t *testing.T) {
	d, _ := NewJsonDetector("level:error")

	line1 := []byte(`{"level":"error", "id":1}`)
	line2 := []byte(`{"level":"error", "id":2}`)

	// Detect line 1
	if !d.Detect(line1) {
		t.Fatal("Line 1 should match")
	}

	// Get Context (should be line 1)
	ctx1 := d.GetContext(line1)
	if id, ok := ctx1["id"].(float64); !ok || id != 1 {
		t.Errorf("Expected id=1, got %v", ctx1["id"])
	}

	// Detect line 2
	if !d.Detect(line2) {
		t.Fatal("Line 2 should match")
	}

	// Get Context (should be line 2)
	ctx2 := d.GetContext(line2)
	if id, ok := ctx2["id"].(float64); !ok || id != 2 {
		t.Errorf("Expected id=2, got %v", ctx2["id"])
	}

	// Cross-check: Get Context for line 1 again (should be fresh parse, not stale line 2 cache)
	ctx1Again := d.GetContext(line1)
	if id, ok := ctx1Again["id"].(float64); !ok || id != 1 {
		t.Errorf("Expected id=1, got %v", ctx1Again["id"])
	}
}

func TestJsonDetector_Concurrency(t *testing.T) {
	d, _ := NewJsonDetector("level:error")
	line := []byte(`{"level":"error", "id":1}`)

	var wg sync.WaitGroup
	// Run Detect and GetContext concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if d.Detect(line) {
				ctx := d.GetContext(line)
				if ctx == nil {
					t.Error("Got nil context")
				}
			}
		}()
	}
	wg.Wait()
}
