package config

import (
	"testing"
)

func TestConfigRedacted(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Flag with equals",
			input:    "--password=secret",
			expected: "--password=[REDACTED]",
		},
		{
			name:     "Flag with space",
			input:    "--password secret",
			expected: "--password [REDACTED]",
		},
		{
			name:     "Quoted secret with spaces",
			input:    "--password \"super secret\"",
			expected: "--password [REDACTED]",
		},
		{
			name:     "Multiple args",
			input:    "--user admin --password \"secret code\" --verbose",
			expected: "--user admin --password [REDACTED] --verbose",
		},
		{
			name:     "Fallback on invalid quotes (unclosed)",
			input:    "--password \"unclosed",
			// SplitCommand fails. Fallback to strings.Fields -> ["--password", "\"unclosed"]
			// SanitizeCommand sees --password, redacts next arg.
			expected: "--password [REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Monitors: []MonitorConfig{{Args: tt.input}},
			}
			redactedCfg := cfg.Redacted()
			got := redactedCfg.Monitors[0].Args

			if got != tt.expected {
				t.Errorf("Redacted() = %q, want %q", got, tt.expected)
			}
		})
	}
}
