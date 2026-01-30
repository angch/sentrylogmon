package main

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestWatchConfig(t *testing.T) {
	// Create a temporary config file
	tmpfile, err := os.CreateTemp("", "config_test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	// Write initial valid config
	initialConfig := `
sentry:
  dsn: "https://example@sentry.io/123"
  environment: "test"
monitors:
  - name: "test"
    type: "file"
    path: "/tmp/test.log"
`
	if _, err := tmpfile.Write([]byte(initialConfig)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reloadCh := make(chan struct{})
	onReload := func() {
		close(reloadCh)
	}

	// Start watcher
	go watchConfig(ctx, tmpfile.Name(), onReload)

	// Wait for watcher to start (naive sleep, but fsnotify startup is fast)
	time.Sleep(100 * time.Millisecond)

	// Test Case 1: Valid Change
	newConfig := `
sentry:
  dsn: "https://example@sentry.io/456"
monitors: []
`
	if err := os.WriteFile(tmpfile.Name(), []byte(newConfig), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case <-reloadCh:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for reload callback on valid config change")
	}
}

func TestWatchConfig_Invalid(t *testing.T) {
	// Create a temporary config file
	tmpfile, err := os.CreateTemp("", "config_test_invalid_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	initialConfig := `
sentry:
  dsn: "https://example@sentry.io/123"
`
	if _, err := tmpfile.Write([]byte(initialConfig)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reloadCh := make(chan struct{}, 1)
	onReload := func() {
		reloadCh <- struct{}{}
	}

	go watchConfig(ctx, tmpfile.Name(), onReload)
	time.Sleep(100 * time.Millisecond)

	// Test Case 2: Invalid Change (Bad YAML)
	invalidConfig := `
sentry:
  dsn: "https://example@sentry.io/123"
  invalid_yaml_indentation
`
	if err := os.WriteFile(tmpfile.Name(), []byte(invalidConfig), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case <-reloadCh:
		t.Fatal("Reload callback called for invalid config")
	case <-time.After(1 * time.Second):
		// Success: should NOT be called
	}
}
