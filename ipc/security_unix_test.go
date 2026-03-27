//go:build unix || linux || darwin

package ipc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecureSocketPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "secure.sock")

	// Create listener using listenSecure
	listener, err := listenSecure("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create secure listener: %v", err)
	}
	defer listener.Close()

	// Check file permissions
	info, err := os.Stat(socketPath)
	if err != nil {
		t.Fatalf("Failed to stat socket file: %v", err)
	}

	mode := info.Mode().Perm()
	// Expected permission is 0600 (rw-------) or 0700 (rwx------)
	// Key requirement is that group and other have no permissions (0077 mask).
	if mode&0077 != 0 {
		t.Errorf("Socket permissions are insecure: %o, expected group/other to have no permissions", mode)
	}
	// Verify owner has at least read/write
	if mode&0600 != 0600 {
		t.Errorf("Socket permissions are invalid: %o, expected owner read/write", mode)
	}
}
