package main

import (
	"regexp"
	"strings"
	"testing"
)

func TestTimestampExtraction(t *testing.T) {
	timestampRegex := regexp.MustCompile(`^\[\s*([0-9.]+)\]`)
	
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
	timestampRegex := regexp.MustCompile(`^\[\s*([0-9.]+)\]`)
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
