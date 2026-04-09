//go:build unix || linux || darwin

package ipc

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

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

	// 3. Check ownership (do this before trying to fix permissions)
	stat, ok := info.Sys().(*syscall.Stat_t)
	if ok {
		uid := uint32(os.Getuid())
		if stat.Uid != uid {
			return fmt.Errorf("insecure ownership on %s: owned by uid %d, expected %d", path, stat.Uid, uid)
		}
	}

	// 4. Check permissions
	mode := info.Mode().Perm()
	if mode != 0700 {
		// Attempt to fix permissions securely using OpenFile with O_NOFOLLOW
		f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
		if err != nil {
			return fmt.Errorf("failed to open %s to fix permissions: %v", path, err)
		}
		defer f.Close()

		if err := f.Chmod(0700); err != nil {
			return fmt.Errorf("insecure permissions on %s (%o) and failed to fix: %v", path, mode, err)
		}
	}

	return nil
}

// GetSocketDir returns the secure socket directory for the current user.
func GetSocketDir() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("sentrylogmon-%d", os.Getuid()))
}

var umaskMu sync.Mutex

// ListenUnix creates a Unix domain socket listener with secure 0600 permissions
// atomically by temporarily changing the umask.
func ListenUnix(socketPath string) (net.Listener, error) {
	umaskMu.Lock()
	defer umaskMu.Unlock()

	// Set umask to 0177 to ensure the created file has 0600 permissions (0777 & ~0177 = 0600).
	// syscall.Umask returns the previous umask.
	oldUmask := syscall.Umask(0177)
	defer syscall.Umask(oldUmask)

	return net.Listen("unix", socketPath)
}
