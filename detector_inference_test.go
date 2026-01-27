package main

import (
	"testing"

	"github.com/angch/sentrylogmon/config"
)

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
