package sysstat

import (
	"strings"
)

var sensitiveFlags = map[string]bool{
	"--password":      true,
	"-p":              false, // -p is ambiguous (e.g., port), so we don't redact it automatically.
	"--token":         true,
	"--api-key":       true,
	"--apikey":        true,
	"--secret":        true,
	"--client-secret": true,
	"--access-token":  true,
	"--auth-token":    true,
	"--session-id":    true,
}

var sensitiveSuffixes = []string{
	"password",
	"token",
	"secret",
	"_key", // Matches api_key, access_key, but not keyboard
	"-key", // Matches ssh-key, private-key
	".key", // Matches file.key
	"signature",
	"credential",
	"cookie",
	"session",
}

// SanitizeCommand joins command arguments into a string, redacting sensitive information.
func SanitizeCommand(args []string) string {
	if len(args) == 0 {
		return ""
	}

	var sanitized []string
	skipNext := false

	for i, arg := range args {
		if skipNext {
			sanitized = append(sanitized, "[REDACTED]")
			skipNext = false
			continue
		}

		// Check for --flag=value
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			key := parts[0]
			// val := parts[1] // unused

			// Normalize key for checking (remove leading dashes)
			cleanKey := strings.TrimLeft(key, "-")

			if isSensitiveKey(cleanKey) {
				sanitized = append(sanitized, key+"=[REDACTED]")
				continue
			}

			// Also check if the key matches a sensitive flag explicitly
			// We must check lowercased key to handle case variations
			if sensitiveFlags[strings.ToLower(key)] {
				sanitized = append(sanitized, key+"=[REDACTED]")
				continue
			}

			// If not sensitive, keep as is
			sanitized = append(sanitized, arg)
			continue
		}

		// Check for sensitive flags that take the next argument
		// 1. Check strict list (case-insensitive)
		lowerArg := strings.ToLower(arg)
		if val, ok := sensitiveFlags[lowerArg]; ok && val {
			sanitized = append(sanitized, arg)
			if i+1 < len(args) {
				skipNext = true
			}
			continue
		}

		// 2. Check heuristics (suffix matching)
		// Clean the arg (remove leading dashes)
		cleanArg := strings.TrimLeft(arg, "-")
		if isSensitiveKey(cleanArg) {
			sanitized = append(sanitized, arg)
			// Only redact next if it doesn't look like another flag
			// This prevents false positives for boolean flags (e.g., --enable-password-auth --verbose)
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				skipNext = true
			}
			continue
		}

		sanitized = append(sanitized, arg)
	}

	return strings.Join(sanitized, " ")
}

func isSensitiveKey(key string) bool {
	lowerKey := strings.ToLower(key)

	// Exact matches
	if lowerKey == "password" || lowerKey == "token" || lowerKey == "secret" || lowerKey == "key" || lowerKey == "auth" {
		return true
	}

	// Suffix matches
	for _, suffix := range sensitiveSuffixes {
		if strings.HasSuffix(lowerKey, suffix) {
			// If the match is the entire string, it's a match
			if len(lowerKey) == len(suffix) {
				return true
			}

			// If the suffix itself starts with a separator, it implies a boundary
			if suffix[0] == '-' || suffix[0] == '_' || suffix[0] == '.' {
				return true
			}

			// Otherwise, check if the suffix is preceded by a separator
			matchIndex := len(lowerKey) - len(suffix)
			if matchIndex > 0 {
				charBefore := lowerKey[matchIndex-1]
				if charBefore == '-' || charBefore == '_' || charBefore == '.' {
					return true
				}
			}
		}
	}
	return false
}
