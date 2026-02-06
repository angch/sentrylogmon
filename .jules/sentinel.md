# Sentinel's Journal

## 2025-02-20 - Insecure IPC Directory Creation
**Vulnerability:** The application created its IPC socket directory in `/tmp` using `os.MkdirAll` without verifying ownership or checking for symbolic links. This allowed a local attacker to pre-create the directory (or a symlink) to hijack the socket or perform a DoS/Symlink attack (changing permissions of arbitrary files).
**Learning:** `os.MkdirAll`, `os.Stat`, and `os.Chmod` follow symlinks by default. When securing a directory in a shared location like `/tmp`, you MUST use `os.Lstat` to detect symlinks and reject them. Additionally, checking file ownership (`syscall.Stat_t.Uid`) is platform-specific and requires build tags to maintain cross-platform compatibility.
**Prevention:**
1. Always use `os.Lstat` to check for symlinks before trusting a directory in `/tmp`.
2. Explicitly verify directory ownership matches the current process user.
3. Use build tags (`//go:build unix`) for OS-specific security checks.

## 2026-01-27 - CLI Argument Redaction Gaps
**Vulnerability:** The command-line sanitizer used an incomplete list of suffixes to identify sensitive flags. It missed flags ending in `-key` (e.g., `--ssh-key`, `--private-key`), allowing sensitive keys to be logged in plain text to Sentry.
**Learning:** Security allowlists/blocklists for partial matching (suffixes/prefixes) must account for common naming conventions like kebab-case (`-key`) and dot-notation (`.key`). Relying on a single separator (like `_key`) is insufficient.
**Prevention:**
1. Include multiple common separators (`-`, `.`, `_`) in suffix matching lists.
2. Add explicit test cases for common sensitive flag variations (`--ssh-key`, `--private-key`).

## 2026-02-14 - Case-Sensitive CLI Redaction Failure
**Vulnerability:** The CLI sanitizer relied on a case-sensitive map lookup for space-separated flags (e.g., `--password`), causing uppercase variants (e.g., `--PASSWORD`) to leak secrets. Additionally, heuristic suffix matching was only applied to `key=value` arguments, missing space-separated flags like `--db-password`.
**Learning:** Security controls based on string matching must always be case-insensitive unless there is a specific reason not to be. Furthermore, heuristic logic (like suffix matching) should be applied consistently across different input formats (space-separated vs. equals-separated) to avoid coverage gaps.
**Prevention:**
1. Normalize inputs (lowercase) before checking against allow/blocklists.
2. Unify validation logic for different input formats to ensure consistent security coverage.
3. When using heuristics on space-separated flags, verify the next argument is not a flag (starts with `-`) to reduce false positives.

## 2026-02-02 - Local DoS via Hardcoded IPC Path
**Vulnerability:** The application used a hardcoded path (`/tmp/sentrylogmon`) for its IPC socket directory. While the directory was secured (0700 permissions), this allowed a local user to pre-create the directory and block other users from starting their own instances (Local Denial of Service) because the application would fail to secure/own the directory.
**Learning:** Hardcoded paths in shared temporary directories (`/tmp`) create resource collision vulnerabilities in multi-user environments. Even if file permissions are secure, the *existence* of the directory owned by another user causes a conflict.
**Prevention:**
1. Avoid hardcoded paths in shared directories.
2. Namespace temporary directories using the user's UID (e.g., `/tmp/app-<uid>`) or use `os.MkdirTemp`.
3. On Windows, rely on `os.TempDir()` which is typically per-user.

## 2026-02-20 - Heuristic Redaction False Positives
**Vulnerability:** The command line sanitizer used suffix matching without checking for word boundaries, causing arguments like `-pSecret` (where `pSecret` ends in `Secret`) to be incorrectly identified as a sensitive flag name. This resulted in the redaction of the *next* argument (often legitimate data like database names) while leaving the sensitive value itself exposed.
**Learning:** Heuristic suffix matching for security redaction must enforce boundaries. Simply ending with "secret" or "password" is insufficient for short flags or attached values. Requiring a separator (e.g., `-`, `_`, `.`) or an exact match ensures that only likely flag names are targeted.
**Prevention:**
1. When using suffix matching for flag detection, enforce that the suffix is preceded by a separator or constitutes the entire string.
2. Avoid applying generic flag heuristics to arguments that don't look like standard flags (e.g. short flags with attached values).

## 2026-02-06 - Unbounded Memory Consumption via Log Buffering
**Vulnerability:** The application buffered log lines based solely on line count (1000 lines) before flushing to Sentry. An attacker or a verbose system could generate large log lines (up to 1MB each), causing the buffer to grow to ~1GB, leading to potential OOM crashes or Sentry payload rejection.
**Learning:** Limiting buffers by "item count" is insufficient when the size of items varies significantly. Always enforce a hard byte limit (e.g., 256KB) in addition to count limits to ensure predictable memory usage and stay within external service payload limits.
**Prevention:**
1. Implement dual thresholds (count AND size) for all buffering logic.
2. Flush the buffer immediately when either threshold is exceeded.
