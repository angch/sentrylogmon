# Sentinel's Journal

## 2025-02-20 - Insecure IPC Directory Creation
**Vulnerability:** The application created its IPC socket directory in `/tmp` using `os.MkdirAll` without verifying ownership or checking for symbolic links. This allowed a local attacker to pre-create the directory (or a symlink) to hijack the socket or perform a DoS/Symlink attack (changing permissions of arbitrary files).
**Learning:** `os.MkdirAll`, `os.Stat`, and `os.Chmod` follow symlinks by default. When securing a directory in a shared location like `/tmp`, you MUST use `os.Lstat` to detect symlinks and reject them. Additionally, checking file ownership (`syscall.Stat_t.Uid`) is platform-specific and requires build tags to maintain cross-platform compatibility.
**Prevention:**
1. Always use `os.Lstat` to check for symlinks before trusting a directory in `/tmp`.
2. Explicitly verify directory ownership matches the current process user.
3. Use build tags (`//go:build unix`) for OS-specific security checks.
