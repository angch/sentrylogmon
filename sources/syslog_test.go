package sources

import (
	"bufio"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestSyslogSource_UDP(t *testing.T) {
	// Use port 0 to let OS pick one
	source := NewSyslogSource("test_udp", "udp:127.0.0.1:0")
	reader, err := source.Stream()
	if err != nil {
		t.Fatalf("Failed to stream: %v", err)
	}
	defer source.Close()

	addr := source.Addr()
	if addr == nil {
		t.Fatal("Source address is nil")
	}

	conn, err := net.Dial("udp", addr.String())
	if err != nil {
		t.Fatalf("Failed to dial UDP: %v", err)
	}
	defer conn.Close()

	msg := "test udp message"
	// UDP packet without newline
	_, err = fmt.Fprintf(conn, "%s", msg)
	if err != nil {
		t.Fatalf("Failed to write to UDP: %v", err)
	}

	scanner := bufio.NewScanner(reader)
	// Give it some time
	done := make(chan bool)
	go func() {
		if scanner.Scan() {
			txt := scanner.Text()
			if txt == msg {
				done <- true
			} else {
				t.Errorf("Expected '%s', got '%s'", msg, txt)
				done <- false
			}
		} else {
			if err := scanner.Err(); err != nil {
				t.Errorf("Scanner error: %v", err)
			} else {
				t.Error("Scanner closed unexpectedly")
			}
			done <- false
		}
	}()

	select {
	case result := <-done:
		if !result {
			t.Fail()
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestSyslogSource_TCP(t *testing.T) {
	source := NewSyslogSource("test_tcp", "tcp:127.0.0.1:0")
	reader, err := source.Stream()
	if err != nil {
		t.Fatalf("Failed to stream: %v", err)
	}
	defer source.Close()

	addr := source.Addr()
	if addr == nil {
		t.Fatal("Source address is nil")
	}

	conn, err := net.Dial("tcp", addr.String())
	if err != nil {
		t.Fatalf("Failed to dial TCP: %v", err)
	}
	defer conn.Close()

	msg := "test tcp message"
	// TCP stream with newline
	_, err = fmt.Fprintf(conn, "%s\n", msg)
	if err != nil {
		t.Fatalf("Failed to write to TCP: %v", err)
	}

	scanner := bufio.NewScanner(reader)
	done := make(chan bool)
	go func() {
		if scanner.Scan() {
			txt := scanner.Text()
			if txt == msg {
				done <- true
			} else {
				t.Errorf("Expected '%s', got '%s'", msg, txt)
				done <- false
			}
		} else {
			if err := scanner.Err(); err != nil {
				t.Errorf("Scanner error: %v", err)
			} else {
				t.Error("Scanner closed unexpectedly")
			}
			done <- false
		}
	}()

	select {
	case result := <-done:
		if !result {
			t.Fail()
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestSyslogSource_Close(t *testing.T) {
	source := NewSyslogSource("test_close", "udp:127.0.0.1:0")
	reader, err := source.Stream()
	if err != nil {
		t.Fatalf("Failed to stream: %v", err)
	}

	// Start reading in bg to drain pipe
	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {}
	}()

	time.Sleep(100 * time.Millisecond)
	if err := source.Close(); err != nil {
		t.Fatalf("Failed to close: %v", err)
	}

	// Should be able to open again?
	// The implementation allows re-Stream?
	// Currently NewSyslogSource sets channel. Close closes it.
	// If we call Stream again, it uses same channel.
	// If channel is closed, the goroutines will exit immediately.
	// So SyslogSource is not reusable after Close.
}
