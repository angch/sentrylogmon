package detectors

import (
	"testing"
)

func TestDmesgTransformMessage(t *testing.T) {
	d := NewDmesgDetector()

	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "[787739.009559] ata1: SError: { PHYRdyChg CommWake 10B8B DevExch }",
			expected: "ata1: SError: { PHYRdyChg CommWake 10B8B DevExch }",
		},
		{
			input:    "[ 123.456] Simple message",
			expected: "Simple message",
		},
		{
			input:    "No timestamp message",
			expected: "No timestamp message",
		},
		{
			input:    "[invalid] timestamp",
			expected: "[invalid] timestamp",
		},
	}

	for _, tt := range tests {
		got := string(d.TransformMessage([]byte(tt.input)))
		if got != tt.expected {
			t.Errorf("TransformMessage(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
