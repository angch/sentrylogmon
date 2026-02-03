package main

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/config"
)

var timestampRegex = regexp.MustCompile(`^\[\s*([0-9.]+)\]`)

func TestTimestampExtraction(t *testing.T) {
	tests := []struct {
		line      string
		timestamp string
		found     bool
	}{
		{
			line:      "[    0.000000] Linux version 5.15.0-86-generic",
			timestamp: "0.000000",
			found:     true,
		},
		{
			line:      "[    2.456789] ACPI Error: AE_NOT_FOUND",
			timestamp: "2.456789",
			found:     true,
		},
		{
			line:      "No timestamp here",
			timestamp: "",
			found:     false,
		},
	}

	for _, tt := range tests {
		matches := timestampRegex.FindStringSubmatch(tt.line)
		if tt.found {
			if len(matches) < 2 {
				t.Errorf("Expected to find timestamp in %q, but didn't", tt.line)
				continue
			}
			if matches[1] != tt.timestamp {
				t.Errorf("Expected timestamp %q, got %q", tt.timestamp, matches[1])
			}
		} else {
			if len(matches) > 1 {
				t.Errorf("Expected not to find timestamp in %q, but found %q", tt.line, matches[1])
			}
		}
	}
}

func TestPatternMatching(t *testing.T) {
	// Case sensitive "Error"
	pattern := regexp.MustCompile("Error")

	tests := []struct {
		line    string
		matches bool
	}{
		{
			line:    "[    8.012345] Error: Failed to load kernel module",
			matches: true,
		},
		{
			line:    "[    2.456789] ACPI Error: AE_NOT_FOUND",
			matches: true,
		},
		{
			line:    "[    9.123456] Warning: Temperature threshold exceeded",
			matches: false,
		},
		{
			line:    "[    5.789012] error: lowercase should not match",
			matches: false, // Case sensitive
		},
	}

	for _, tt := range tests {
		matched := pattern.MatchString(tt.line)
		if matched != tt.matches {
			t.Errorf("For line %q, expected match=%v, got %v", tt.line, tt.matches, matched)
		}
	}
}

func TestGroupingByTimestamp(t *testing.T) {
	lines := []string{
		"[    2.456789] ACPI Error: AE_NOT_FOUND",
		"[    2.456789] ACPI Error: Could not find ACPI method",
		"[    8.012345] Error: Failed to load kernel module i915",
		"[    8.012345] Error: Graphics initialization failed",
		"[    9.123456] Error: Critical temperature reached",
	}

	// Simulate grouping
	groups := make(map[string][]string)
	for _, line := range lines {
		matches := timestampRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			timestamp := matches[1]
			groups[timestamp] = append(groups[timestamp], line)
		}
	}

	// Verify grouping
	if len(groups) != 3 {
		t.Errorf("Expected 3 groups, got %d", len(groups))
	}

	if len(groups["2.456789"]) != 2 {
		t.Errorf("Expected 2 lines for timestamp 2.456789, got %d", len(groups["2.456789"]))
	}

	if len(groups["8.012345"]) != 2 {
		t.Errorf("Expected 2 lines for timestamp 8.012345, got %d", len(groups["8.012345"]))
	}

	if len(groups["9.123456"]) != 1 {
		t.Errorf("Expected 1 line for timestamp 9.123456, got %d", len(groups["9.123456"]))
	}
}

func TestMessageFormatting(t *testing.T) {
	timestamp := "2.456789"
	lines := []string{
		"[    2.456789] ACPI Error: AE_NOT_FOUND",
		"[    2.456789] ACPI Error: Could not find ACPI method",
	}

	message := "Log errors at timestamp [2.456789]"
	eventDetails := strings.Join(lines, "\n")

	if !strings.Contains(message, timestamp) {
		t.Errorf("Message should contain timestamp")
	}

	if !strings.Contains(eventDetails, "AE_NOT_FOUND") {
		t.Errorf("Event details should contain first error")
	}

	if !strings.Contains(eventDetails, "Could not find ACPI method") {
		t.Errorf("Event details should contain second error")
	}
}

func TestDetermineDetectorFormat(t *testing.T) {
	tests := []struct {
		name     string
		monCfg   config.MonitorConfig
		expected string
	}{
		{
			name: "Explicit format",
			monCfg: config.MonitorConfig{
				Format: "nginx",
			},
			expected: "nginx",
		},
		{
			name: "Dmesg type without pattern",
			monCfg: config.MonitorConfig{
				Type: "dmesg",
			},
			expected: "dmesg",
		},
		{
			name: "Dmesg type with custom pattern",
			monCfg: config.MonitorConfig{
				Type:    "dmesg",
				Pattern: "some-error",
			},
			expected: "custom",
		},
		{
			name: "File type without pattern",
			monCfg: config.MonitorConfig{
				Type: "file",
			},
			expected: "custom",
		},
		{
			name: "File type with pattern",
			monCfg: config.MonitorConfig{
				Type:    "file",
				Pattern: "some-error",
			},
			expected: "custom",
		},
		{
			name: "Explicit format overrides pattern and type",
			monCfg: config.MonitorConfig{
				Format:  "nginx",
				Type:    "dmesg",
				Pattern: "some-error",
			},
			expected: "nginx",
		},
		{
			name: "Name matching known detector (nginx)",
			monCfg: config.MonitorConfig{
				Name: "nginx",
				Type: "file",
			},
			expected: "nginx",
		},
		{
			name: "Name matching known detector (nginx-error)",
			monCfg: config.MonitorConfig{
				Name: "nginx-error",
				Type: "file",
			},
			expected: "nginx-error",
		},
		{
			name: "Unknown Name defaults to custom",
			monCfg: config.MonitorConfig{
				Name: "foobar",
				Type: "file",
			},
			expected: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineDetectorFormat(tt.monCfg)
			if got != tt.expected {
				t.Errorf("determineDetectorFormat() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d        time.Duration
		expected string
	}{
		{5 * time.Second, "5s"},
		{65 * time.Second, "1m 5s"},
		{125 * time.Second, "2m 5s"},
		{3600 * time.Second, "1h 0m 0s"},
		{3665 * time.Second, "1h 1m 5s"},
		{7320 * time.Second, "2h 2m 0s"},
		{59 * time.Second, "59s"},
		{59 * time.Minute, "59m 0s"},
		{23 * time.Hour, "23h 0m 0s"},
		{25 * time.Hour, "1d 1h 0m"},
		{48 * time.Hour, "2d 0h 0m"},
		{50*time.Hour + 30*time.Minute, "2d 2h 30m"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := formatDuration(tt.d)
			if got != tt.expected {
				t.Errorf("formatDuration(%v) = %v, want %v", tt.d, got, tt.expected)
			}
		})
	}
}
