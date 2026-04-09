//go:build windows

package ipc

import (
	"net"
	"os"
	"path/filepath"
)

// ListenUnix is a fallback for Windows that does not use umask.
func ListenUnix(socketPath string) (net.Listener, error) {
	return net.Listen("unix", socketPath)
}

// EnsureSecureDirectory ensures that the directory at path exists.
// Security checks are simplified for Windows.
func EnsureSecureDirectory(path string) error {
	return os.MkdirAll(path, 0700)
}

// GetSocketDir returns the secure socket directory.
func GetSocketDir() string {
	return filepath.Join(os.TempDir(), "sentrylogmon")
}
