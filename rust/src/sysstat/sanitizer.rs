use once_cell::sync::Lazy;
use std::collections::HashMap;

static SENSITIVE_FLAGS: Lazy<HashMap<&'static str, bool>> = Lazy::new(|| {
    let mut m = HashMap::new();
    m.insert("--password", true);
    m.insert("-p", false); // Ambiguous
    m.insert("--token", true);
    m.insert("--api-key", true);
    m.insert("--apikey", true);
    m.insert("--secret", true);
    m.insert("--client-secret", true);
    m.insert("--access-token", true);
    m.insert("--auth-token", true);
    m
});

static SENSITIVE_SUFFIXES: Lazy<Vec<&'static str>> =
    Lazy::new(|| vec!["password", "token", "secret", "_key", "-key", ".key", "signature", "credential", "cookie", "session"]);

// sanitize_command reconstructs the command line string from arguments while redacting sensitive information.
// It aims for parity with the Go implementation, handling both `--flag=value` and `--flag value` patterns.
// Note: Space-separated flags rely on a hardcoded allowlist (SENSITIVE_FLAGS), while `=` separated flags
// use heuristic matching for keys (e.g. ending in "password" or "token").
pub fn sanitize_command(args: &[String]) -> String {
    if args.is_empty() {
        return String::new();
    }

    let mut sanitized = Vec::new();
    let mut skip_next = false;

    for (i, arg) in args.iter().enumerate() {
        if skip_next {
            sanitized.push("[REDACTED]".to_string());
            skip_next = false;
            continue;
        }

        // Check for --flag=value
        if let Some((key, _)) = arg.split_once('=') {
            // Normalize key (remove leading dashes)
            let clean_key = key.trim_start_matches('-');

            if is_sensitive_key(clean_key) {
                sanitized.push(format!("{}=[REDACTED]", key));
                continue;
            }

            // Check if key matches a sensitive flag explicitly
            // We must check lowercased key to handle case variations
            if SENSITIVE_FLAGS.contains_key(key.to_lowercase().as_str()) {
                sanitized.push(format!("{}=[REDACTED]", key));
                continue;
            }

            sanitized.push(arg.clone());
            continue;
        }

        // Check for sensitive flags that take the next argument
        // 1. Check strict list (case-insensitive)
        let lower_arg = arg.to_lowercase();
        if let Some(&should_skip) = SENSITIVE_FLAGS.get(lower_arg.as_str()) {
            sanitized.push(arg.clone());
            if should_skip && i + 1 < args.len() {
                skip_next = true;
            }
            continue;
        }

        // 2. Check heuristics (suffix matching)
        // Clean the arg (remove leading dashes)
        let clean_arg = arg.trim_start_matches('-');
        if is_sensitive_key(clean_arg) {
            sanitized.push(arg.clone());
            // Only redact next if it doesn't look like another flag
            // This prevents false positives for boolean flags
            if i + 1 < args.len() && !args[i + 1].starts_with('-') {
                skip_next = true;
            }
            continue;
        }

        sanitized.push(arg.clone());
    }

    sanitized.join(" ")
}

fn is_sensitive_key(key: &str) -> bool {
    let lower_key = key.to_lowercase();

    // Exact matches
    if matches!(
        lower_key.as_str(),
        "password" | "token" | "secret" | "key" | "auth"
    ) {
        return true;
    }

    // Suffix matches
    for suffix in SENSITIVE_SUFFIXES.iter() {
        if lower_key.ends_with(suffix) {
            // If the match is the entire string, it's a match
            if lower_key.len() == suffix.len() {
                return true;
            }

            // If the suffix itself starts with a separator, it implies a boundary
            if suffix.starts_with('-') || suffix.starts_with('_') || suffix.starts_with('.') {
                return true;
            }

            // Otherwise, check if the suffix is preceded by a separator
            let match_index = lower_key.len() - suffix.len();
            if match_index > 0 {
                let char_before = lower_key.as_bytes()[match_index - 1];
                if char_before == b'-' || char_before == b'_' || char_before == b'.' {
                    return true;
                }

                // Check for camelCase boundary in the original key string
                // The character at `match_index` in the original `key` should be uppercase
                // (e.g. `P` in `dbPassword`)
                // Ensure we don't index out of bounds if `to_lowercase` expanded unicode bytes
                if match_index < key.len() {
                    let orig_char = key.as_bytes()[match_index];
                    if orig_char.is_ascii_uppercase() {
                        // prevent `-pSecret` from matching by enforcing `match_index > 1` when it's just a camelCase letter boundary
                        if match_index > 1 {
                            return true;
                        }
                    }
                }

                // If there's no boundary but the key contains no separators at all
                // (e.g. `adminpassword`), we assume it's a concatenated word flag.
                // However, we want to prevent matching `pSecret` (where `match_index` is 1).
                // So we only accept it if `match_index` > 1 (e.g. it's a real flag like `dbpassword`)
                // or if it doesn't look like a short flag. Also exclude values that contain `:` (like `user:password`).
                if match_index > 1 && !key.contains('-') && !key.contains('_') && !key.contains('.') && !key.contains(':') {
                    return true;
                }
            }
        }
    }

    false
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_sanitize_command() {
        let cases = vec![
            (
                vec!["curl", "-u", "user:password", "http://example.com"],
                "curl -u user:password http://example.com", // -u is not in sensitive list
            ),
            (
                vec!["myapp", "--password", "secret123"],
                "myapp --password [REDACTED]",
            ),
            (
                vec!["myapp", "--PASSWORD", "secret123"],
                "myapp --PASSWORD [REDACTED]",
            ),
            (
                vec!["myapp", "--token=secret123"],
                "myapp --token=[REDACTED]",
            ),
            (
                vec!["myapp", "--api-key", "abcdef"],
                "myapp --api-key [REDACTED]",
            ),
            (
                vec!["db", "--db_password=secure"],
                "db --db_password=[REDACTED]",
            ),
            (
                vec!["db", "--db-password", "secure"],
                "db --db-password [REDACTED]",
            ),
            (
                vec!["service", "--aws_secret_access_key=XYZ"],
                "service --aws_secret_access_key=[REDACTED]", // matches _key suffix
            ),
            (
                vec!["ssh", "-p", "2222"],
                "ssh -p 2222", // -p is ambiguous, false in map
            ),
            (
                vec!["myapp", "-pSecret", "dbname"],
                "myapp -pSecret dbname", // -pSecret should not redact 'dbname'
            ),
            (
                vec!["myapp", "--ssh-key", "key.pem"],
                "myapp --ssh-key [REDACTED]",
            ),
            (
                vec!["myapp", "--file.key", "key.pem"],
                "myapp --file.key [REDACTED]",
            ),
            (
                vec!["myapp", "--dbPassword", "supersecret"],
                "myapp --dbPassword [REDACTED]",
            ),
            (
                vec!["myapp", "--adminpassword", "supersecret"],
                "myapp --adminpassword [REDACTED]",
            ),
            (
                vec!["myapp", "--apiToken=123"],
                "myapp --apiToken=[REDACTED]",
            ),
        ];

        for (input, expected) in cases {
            let input_vec: Vec<String> = input.iter().map(|s| s.to_string()).collect();
            assert_eq!(
                sanitize_command(&input_vec),
                expected,
                "Failed on input: {:?}",
                input
            );
        }
    }
}
