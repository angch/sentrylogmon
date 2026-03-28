package sysstat

import (
	"testing"
)

func TestSanitizeCommandWithSplitter(t *testing.T) {
	// This test simulates how SplitCommand and SanitizeCommand are used together
	// to solve the original vulnerability where quoted secrets were leaked.

	tests := []struct {
		name     string
		rawInput string
		expected string
	}{
		{
			name:     "Quoted password",
			rawInput: "--password \"my secret\"",
			expected: "--password [REDACTED]",
		},
		{
			name:     "Quoted password with single quotes",
			rawInput: "--password 'my secret'",
			expected: "--password [REDACTED]",
		},
		{
			name:     "Quoted token",
			rawInput: "--token=\"secret token\"",
			expected: "--token=[REDACTED]",
		},
		{
			name:     "Quoted DSN (new protection)",
			rawInput: "--dsn \"https://key@sentry.io/123\"",
			expected: "--dsn [REDACTED]",
		},
		{
			name:     "Quoted Sentry DSN (new protection)",
			rawInput: "--sentry-dsn \"https://key@sentry.io/123\"",
			expected: "--sentry-dsn [REDACTED]",
		},
		{
			name:     "Multiple args with quotes",
			rawInput: "--user admin --password \"complex password\" --verbose",
			expected: "--user admin --password [REDACTED] --verbose",
		},
		{
			name:     "Escaped quotes in secret",
			rawInput: "--secret \"pass\\\"word\"",
			expected: "--secret [REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Split using the new splitter
			parts, err := SplitCommand(tt.rawInput)
			if err != nil {
				t.Fatalf("SplitCommand failed: %v", err)
			}

			// 2. Sanitize using the sanitizer
			sanitized := SanitizeCommand(parts)

			// 3. Verify
			if sanitized != tt.expected {
				t.Errorf("SanitizeCommand(SplitCommand(%q)) = %q, want %q", tt.rawInput, sanitized, tt.expected)
			}
		})
	}
}
