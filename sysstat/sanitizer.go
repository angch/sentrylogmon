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
}

var sensitiveSuffixes = []string{
	"password",
	"token",
	"secret",
	"_key", // Matches api_key, access_key, but not keyboard
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
			if sensitiveFlags[key] {
				sanitized = append(sanitized, key+"=[REDACTED]")
				continue
			}

			// If not sensitive, keep as is
			sanitized = append(sanitized, arg)
			continue
		}

		// Check for sensitive flags that take the next argument
		if val, ok := sensitiveFlags[arg]; ok && val {
			sanitized = append(sanitized, arg)
			if i+1 < len(args) {
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
			return true
		}
	}
	return false
}
