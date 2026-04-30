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
    Lazy::new(|| vec![
        "password",
        "token",
        "secret",
        "_key",
        "-key",
        ".key",
        "signature",
        "credential",
        "cookie",
        "session",
    ]);

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
            if SENSITIVE_FLAGS.contains_key(key.to_lowercase().as_str()) {
                sanitized.push(format!("{}=[REDACTED]", key));
                continue;
            }

            sanitized.push(arg.clone());
            continue;
        }

        let lower_arg = arg.to_lowercase();
        // Check for sensitive flags that take the next argument
        if let Some(&should_skip) = SENSITIVE_FLAGS.get(lower_arg.as_str()) {
            sanitized.push(arg.clone());
            if should_skip && i + 1 < args.len() {
                skip_next = true;
            }
            continue;
        }

        let clean_arg = lower_arg.trim_start_matches('-');
        if is_sensitive_key(clean_arg) {
            sanitized.push(arg.clone());
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
                let bytes = lower_key.as_bytes();
                let char_before = bytes[match_index - 1] as char;
                if char_before == '-' || char_before == '_' || char_before == '.' {
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
                vec!["service", "--aws_secret_access_key=XYZ"],
                "service --aws_secret_access_key=[REDACTED]", // matches _key suffix
            ),
            (
                vec!["ssh", "-p", "2222"],
                "ssh -p 2222", // -p is ambiguous, false in map
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
