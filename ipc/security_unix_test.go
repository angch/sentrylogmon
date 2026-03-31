//go:build unix || linux || darwin

package ipc

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestSocketPermissions(t *testing.T) {
	// Set a permissive umask for the process (temporarily)
	// 0000 means rw-rw-rw- allowed
	oldMask := syscall.Umask(0000)
	defer syscall.Umask(oldMask)

	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "perm_test.sock")

	// Create listener directly to verify listenSecure behavior
	l, err := listenSecure("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to listenSecure: %v", err)
	}
	defer l.Close()

	info, err := os.Stat(socketPath)
	if err != nil {
		t.Fatalf("Failed to stat socket: %v", err)
	}

	mode := info.Mode().Perm()
	// Check that group and other permissions are zero
	// We expect 0600 or 0700 depending on platform implementation, but strictly no group/other access
	if mode&0077 != 0 {
		t.Errorf("Expected secure permissions (group/other=0), got %04o", mode)
	}
}
