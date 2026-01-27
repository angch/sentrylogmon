package sources

import (
	"bufio"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileSourceRotation(t *testing.T) {
	// Create a temp dir
	tmpDir, err := os.MkdirTemp("", "sentrylogmon_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "test.log")

	// Create initial file
	f, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("initial content\n")
	f.Sync()
	f.Close()

	// Start source
	src := NewFileSource("test", logPath)
	stream, err := src.Stream()
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	// Give watcher time to start
	time.Sleep(200 * time.Millisecond)

	scanner := bufio.NewScanner(stream)

	// Helper to read a line with timeout
	readLine := func() string {
		done := make(chan string)
		go func() {
			if scanner.Scan() {
				done <- scanner.Text()
			} else {
				close(done)
			}
		}()

		select {
		case line := <-done:
			return line
		case <-time.After(2 * time.Second):
			return "TIMEOUT"
		}
	}

	// Should start at end, so "initial content" is skipped?
	// The implementation seeks to end on initial open.
	// So we verify we DON'T get "initial content".

	// Write "line 1"
	f, err = os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("line 1\n")
	f.Sync()
	f.Close()

	if line := readLine(); line != "line 1" {
		t.Errorf("Expected 'line 1', got '%s'", line)
	}

	// Rotate
	rotatedPath := logPath + ".1"
	if err := os.Rename(logPath, rotatedPath); err != nil {
		t.Fatal(err)
	}

	// Create new file
	f, err = os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("line 2\n")
	f.Sync()
	f.Close()

	// Should read "line 2"
	// We allow some time for events to propagate
	if line := readLine(); line != "line 2" {
		t.Errorf("Expected 'line 2', got '%s'", line)
	}
}
