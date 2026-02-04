package ipc

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/angch/sentrylogmon/config"
)

func TestStatusRedaction(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "sentrylogmon.sock")

	secretDSN := "https://secret_key@sentry.io/123"
	cfg := &config.Config{
		Sentry: config.SentryConfig{
			DSN: secretDSN,
		},
		Monitors: []config.MonitorConfig{
			{
				Name: "test-monitor",
				Sentry: config.SentryConfig{
					DSN: secretDSN,
				},
			},
		},
	}

	// Start Server
	// We need to run this in a goroutine as it blocks
	go func() {
		// StartServer blocks until error or close
		_ = StartServer(socketPath, cfg, nil)
	}()

	// Wait for socket to appear
	deadline := time.Now().Add(2 * time.Second)
	socketReady := false
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			socketReady = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !socketReady {
		t.Fatal("Timeout waiting for socket creation")
	}

	// Connect using unix socket
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	resp, err := client.Get("http://unix/status")
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check Global DSN
	if status.Config.Sentry.DSN == secretDSN {
		t.Error("Global DSN was exposed (not redacted)")
	} else if status.Config.Sentry.DSN != "***" {
		t.Errorf("Global DSN was %q, expected '***'", status.Config.Sentry.DSN)
	}

	// Check Monitor DSN
	if len(status.Config.Monitors) > 0 {
		if status.Config.Monitors[0].Sentry.DSN == secretDSN {
			t.Error("Monitor DSN was exposed (not redacted)")
		} else if status.Config.Monitors[0].Sentry.DSN != "***" {
			t.Errorf("Monitor DSN was %q, expected '***'", status.Config.Monitors[0].Sentry.DSN)
		}
	} else {
		t.Error("Monitors list is empty, expected 1 monitor")
	}
}
