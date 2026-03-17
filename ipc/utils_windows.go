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

// ListenUnix creates a Unix domain socket listener.
// Note: permissions are less strict on Windows
func ListenUnix(socketPath string) (net.Listener, error) {
	return net.Listen("unix", socketPath)
}
