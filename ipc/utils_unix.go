//go:build unix || linux || darwin

package ipc

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
)

// listenSecure creates a listener with secure permissions (0600) by setting the umask.
func listenSecure(network, address string) (net.Listener, error) {
	oldMask := syscall.Umask(0077)
	defer syscall.Umask(oldMask)

	return net.Listen(network, address)
}

// EnsureSecureDirectory ensures that the directory at path exists,
// is a directory, has 0700 permissions, and is owned by the current user.
// It also checks that the path is not a symlink.
func EnsureSecureDirectory(path string) error {
	// 1. Check if path exists and is a symlink using Lstat
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		// Directory doesn't exist, create it with 0700
		if err := os.MkdirAll(path, 0700); err != nil {
			return err
		}
		// Verify creation
		info, err = os.Lstat(path)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// 2. Reject Symlinks
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink", path)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}

	// 3. Check permissions
	mode := info.Mode().Perm()
	if mode != 0700 {
		// Attempt to fix permissions
		if err := os.Chmod(path, 0700); err != nil {
			return fmt.Errorf("insecure permissions on %s (%o) and failed to fix: %v", path, mode, err)
		}
	}

	// 4. Check ownership
	stat, ok := info.Sys().(*syscall.Stat_t)
	if ok {
		uid := uint32(os.Getuid())
		if stat.Uid != uid {
			return fmt.Errorf("insecure ownership on %s: owned by uid %d, expected %d", path, stat.Uid, uid)
		}
	}

	return nil
}

// GetSocketDir returns the secure socket directory for the current user.
func GetSocketDir() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("sentrylogmon-%d", os.Getuid()))
}
