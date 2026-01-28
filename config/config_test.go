package config

import (
	"os"
	"testing"
)

func TestLoadConfigFromFile(t *testing.T) {
	// Set the flag value directly since we are in the same package
	configPath := "../testdata/config_test.yaml"
	*configFile = configPath
	defer func() { *configFile = "" }()

	// Run Load
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Sentry.DSN != "https://test@sentry.io/123" {
		t.Errorf("Expected DSN 'https://test@sentry.io/123', got '%s'", cfg.Sentry.DSN)
	}

	if cfg.Sentry.Environment != "staging" {
		t.Errorf("Expected Environment 'staging', got '%s'", cfg.Sentry.Environment)
	}

	if len(cfg.Monitors) != 1 {
		t.Errorf("Expected 1 monitor, got %d", len(cfg.Monitors))
	}

	if cfg.Monitors[0].Name != "test-monitor" {
		t.Errorf("Expected monitor name 'test-monitor', got '%s'", cfg.Monitors[0].Name)
	}

	if cfg.Monitors[0].Format != "custom" {
		t.Errorf("Expected format 'custom', got '%s'", cfg.Monitors[0].Format)
	}
}

func TestLoadConfigFallback(t *testing.T) {
	// Create a minimal config file without Sentry info
	minimalConfig := `
monitors:
  - name: test
    type: file
`
	tmpfile, err := os.CreateTemp("", "config_fallback_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(minimalConfig)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	*configFile = tmpfile.Name()
	defer func() { *configFile = "" }()

	// Set fallback flags
	expectedDSN := "https://fallback@sentry.io/0"
	*dsn = expectedDSN
	defer func() { *dsn = "" }()

	expectedEnv := "fallback-env"
	*environment = expectedEnv
	defer func() { *environment = "production" }() // Restore default

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Sentry.DSN != expectedDSN {
		t.Errorf("Expected fallback DSN '%s', got '%s'", expectedDSN, cfg.Sentry.DSN)
	}

	if cfg.Sentry.Environment != expectedEnv {
		t.Errorf("Expected fallback Environment '%s', got '%s'", expectedEnv, cfg.Sentry.Environment)
	}
}

func TestMonitorConfigExcludeParsing(t *testing.T) {
	// Test parsing of exclude_pattern
	yamlConfig := `
monitors:
  - name: exclude-test
    type: file
    path: /tmp/test.log
    exclude_pattern: "ignore me"
`
	tmpfile, err := os.CreateTemp("", "config_exclude_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(yamlConfig)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	*configFile = tmpfile.Name()
	defer func() { *configFile = "" }()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.Monitors) != 1 {
		t.Fatalf("Expected 1 monitor, got %d", len(cfg.Monitors))
	}

	if cfg.Monitors[0].ExcludePattern != "ignore me" {
		t.Errorf("Expected ExcludePattern 'ignore me', got '%s'", cfg.Monitors[0].ExcludePattern)
	}
}

func TestLoadConfigFromFlags(t *testing.T) {
	// Reset config file
	*configFile = ""

	// Set flags
	*inputFile = "/tmp/test.log"
	defer func() { *inputFile = "" }()

	*pattern = "Error"
	defer func() { *pattern = "Error" }()

	*excludePattern = "Info"
	defer func() { *excludePattern = "" }()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.Monitors) != 1 {
		t.Fatalf("Expected 1 monitor, got %d", len(cfg.Monitors))
	}

	if cfg.Monitors[0].ExcludePattern != "Info" {
		t.Errorf("Expected ExcludePattern 'Info', got '%s'", cfg.Monitors[0].ExcludePattern)
	}
}
