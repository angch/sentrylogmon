//go:build windows

package ipc

import (
	"net"
	"os"
	"path/filepath"
)

// listenSecure creates a listener. Windows permissions are less granular for sockets,
// so this just wraps net.Listen.
func listenSecure(network, address string) (net.Listener, error) {
	return net.Listen(network, address)
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
