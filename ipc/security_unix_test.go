//go:build unix || linux || darwin

package ipc

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestListenSecurePermissions(t *testing.T) {
	// 1. Create a temporary directory
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	// 2. Set a lax umask (0000) to see if listenSecure protects us
	// Save the old mask to restore it later
	oldMask := syscall.Umask(0000)
	defer syscall.Umask(oldMask)

	// 3. Call listenSecure
	listener, err := listenSecure("unix", socketPath)
	if err != nil {
		t.Fatalf("listenSecure failed: %v", err)
	}
	defer listener.Close()

	// 4. Verify file permissions
	// With umask 0000, net.Listen would normally create 0777 or 0666.
	// listenSecure uses umask 0077 internally, so result should be:
	// 0666 & ~0077 = 0600 (or 0777 & ~0077 = 0700)
	info, err := os.Stat(socketPath)
	if err != nil {
		t.Fatalf("Failed to stat socket: %v", err)
	}

	mode := info.Mode().Perm()
	// Expected permission is usually 0700 or 0600 depending on Go version/OS details for sockets.
	// But definitively NO group or other permissions.
	// 0077 mask corresponds to group (rwx) and other (rwx) bits.
	if mode&0077 != 0 {
		t.Errorf("Socket has insecure permissions: %o (expected no group/other access)", mode)
	}

	t.Logf("Socket permissions: %o", mode)
}
