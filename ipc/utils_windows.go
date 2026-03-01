//go:build windows

package ipc

import (
	"net"
	"os"
	"path/filepath"
)

// EnsureSecureDirectory ensures that the directory at path exists.
// Security checks are simplified for Windows.
func EnsureSecureDirectory(path string) error {
	return os.MkdirAll(path, 0700)
}

// GetSocketDir returns the secure socket directory.
func GetSocketDir() string {
	return filepath.Join(os.TempDir(), "sentrylogmon")
}

// ListenUnix creates a Unix domain socket listener. On Windows, we just
// rely on standard net.Listen as Unix socket permissions via umask aren't directly applicable.
func ListenUnix(socketPath string) (net.Listener, error) {
	return net.Listen("unix", socketPath)
}
