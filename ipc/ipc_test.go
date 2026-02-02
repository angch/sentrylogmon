package ipc

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetSocketDir(t *testing.T) {
	dir := GetSocketDir()
	if dir == "" {
		t.Error("GetSocketDir() returned empty string")
	}

	isWindows := runtime.GOOS == "windows"
	if !isWindows {
		expectedSuffix := fmt.Sprintf("sentrylogmon-%d", os.Getuid())
		if filepath.Base(dir) != expectedSuffix {
			t.Errorf("GetSocketDir() = %s, expected base %s", dir, expectedSuffix)
		}
	} else {
		if filepath.Base(dir) != "sentrylogmon" {
			t.Errorf("GetSocketDir() = %s, expected base sentrylogmon", dir)
		}
	}
}

func TestEnsureSecureDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	isWindows := runtime.GOOS == "windows"

	// Case 1: Directory does not exist (should create)
	dir1 := filepath.Join(tmpDir, "dir1")
	if err := EnsureSecureDirectory(dir1); err != nil {
		t.Fatalf("Case 1 failed: %v", err)
	}
	info, err := os.Stat(dir1)
	if err != nil {
		t.Fatalf("Case 1 stat failed: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("Case 1: expected directory")
	}
	if !isWindows {
		if info.Mode().Perm() != 0700 {
			t.Errorf("Case 1: expected 0700, got %o", info.Mode().Perm())
		}
	}

	// Case 2: Directory exists with correct permissions (should pass)
	if err := EnsureSecureDirectory(dir1); err != nil {
		t.Fatalf("Case 2 failed: %v", err)
	}

	// Case 3: Directory exists with wrong permissions (should fix on Unix)
	if !isWindows {
		dir3 := filepath.Join(tmpDir, "dir3")
		if err := os.Mkdir(dir3, 0777); err != nil {
			t.Fatalf("Failed to create dir3: %v", err)
		}
		// Verify it's wrong first
		info, _ = os.Stat(dir3)
		t.Logf("Case 3 initial permissions: %o", info.Mode().Perm())

		if err := EnsureSecureDirectory(dir3); err != nil {
			t.Fatalf("Case 3 failed: %v", err)
		}
		info, _ = os.Stat(dir3)
		if info.Mode().Perm() != 0700 {
			t.Errorf("Case 3: expected 0700 after fix, got %o", info.Mode().Perm())
		}
	}

	// Case 4: Path exists but is a file (should error)
	file4 := filepath.Join(tmpDir, "file4")
	if err := os.WriteFile(file4, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create file4: %v", err)
	}
	if err := EnsureSecureDirectory(file4); err == nil {
		t.Errorf("Case 4: expected error for file path, got nil")
	}

	// Case 5: Path is a symlink (should error on Unix)
	if !isWindows {
		link5 := filepath.Join(tmpDir, "link5")
		// Point to valid dir
		if err := os.Symlink(dir1, link5); err != nil {
			t.Fatalf("Failed to create symlink: %v", err)
		}
		if err := EnsureSecureDirectory(link5); err == nil {
			t.Errorf("Case 5: expected error for symlink, got nil")
		} else {
			t.Logf("Case 5: successfully rejected symlink: %v", err)
		}
	}
}
