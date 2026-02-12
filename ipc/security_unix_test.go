//go:build unix || linux || darwin

package ipc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSocketPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test_secure.sock")

	listener, err := listenSecure("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	info, err := os.Stat(socketPath)
	if err != nil {
		t.Fatalf("Failed to stat socket: %v", err)
	}

	mode := info.Mode().Perm()
	// Check if group or other have any permissions
	if mode&0077 != 0 {
		t.Errorf("Socket has insecure permissions: %o (expected no group/other access)", mode)
	}

	// Verify owner has read/write
	if mode&0600 != 0600 {
		t.Errorf("Socket missing owner read/write permissions: %o", mode)
	}
}
