package config

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMonitorConfigFormatParsing(t *testing.T) {
	yamlData := `
monitors:
  - name: explicit-format
    type: file
    path: /var/log/test.log
    format: nginx
    pattern: ignored_when_format_is_known

  - name: implicit-format
    type: file
    path: /var/log/syslog
    pattern: error

  - name: custom-format
    type: file
    path: /var/log/app.log
    format: custom
    pattern: panic
`

	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "config_format_test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(yamlData)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// We can't easily use Load() because it relies on global flags,
	// but we can test the struct unmarshalling directly which is what matters for the field presence.

	var cfg Config
	err = yaml.Unmarshal([]byte(yamlData), &cfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if len(cfg.Monitors) != 3 {
		t.Fatalf("Expected 3 monitors, got %d", len(cfg.Monitors))
	}

	// 1. Explicit format
	m1 := cfg.Monitors[0]
	if m1.Name != "explicit-format" {
		t.Errorf("Expected first monitor to be 'explicit-format', got '%s'", m1.Name)
	}
	if m1.Format != "nginx" {
		t.Errorf("Expected m1.Format to be 'nginx', got '%s'", m1.Format)
	}

	// 2. Implicit format (should be empty string, logic in main.go handles fallback)
	m2 := cfg.Monitors[1]
	if m2.Name != "implicit-format" {
		t.Errorf("Expected second monitor to be 'implicit-format', got '%s'", m2.Name)
	}
	if m2.Format != "" {
		t.Errorf("Expected m2.Format to be empty (default), got '%s'", m2.Format)
	}

	// 3. Custom format
	m3 := cfg.Monitors[2]
	if m3.Name != "custom-format" {
		t.Errorf("Expected third monitor to be 'custom-format', got '%s'", m3.Name)
	}
	if m3.Format != "custom" {
		t.Errorf("Expected m3.Format to be 'custom', got '%s'", m3.Format)
	}
}
