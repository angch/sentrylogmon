package sysstat

import (
	"fmt"
	"strings"
)

// SplitCommand parses a command string into arguments, respecting shell-style quoting.
// It supports:
// - Single quotes: 'arg with spaces' (literal content)
// - Double quotes: "arg with spaces" (supports \" and \\ escapes)
// - Backslash escapes outside quotes: \ (space), \" (quote), \\ (backslash)
// It returns an error if quotes are unbalanced or trailing backslash.
func SplitCommand(s string) ([]string, error) {
	var args []string
	var current strings.Builder
	var escaped bool
	var inSingleQuote bool
	var inDoubleQuote bool
	var inArg bool

	for _, r := range s {
		if inSingleQuote {
			if r == '\'' {
				inSingleQuote = false
			} else {
				current.WriteRune(r)
			}
			inArg = true
			continue
		}

		if inDoubleQuote {
			if escaped {
				// Inside double quote, we only escape " and \
				if r == '"' || r == '\\' {
					current.WriteRune(r)
				} else {
					current.WriteRune('\\')
					current.WriteRune(r)
				}
				escaped = false
			} else if r == '\\' {
				escaped = true
			} else if r == '"' {
				inDoubleQuote = false
			} else {
				current.WriteRune(r)
			}
			inArg = true
			continue
		}

		// Outside quotes
		if escaped {
			current.WriteRune(r)
			escaped = false
			inArg = true
			continue
		}

		if r == '\\' {
			escaped = true
			inArg = true
			continue
		}

		if r == '\'' {
			inSingleQuote = true
			inArg = true
			continue
		}

		if r == '"' {
			inDoubleQuote = true
			inArg = true
			continue
		}

		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if inArg {
				args = append(args, current.String())
				current.Reset()
				inArg = false
			}
			continue
		}

		current.WriteRune(r)
		inArg = true
	}

	if escaped {
		return nil, fmt.Errorf("unexpected end of command (trailing backslash)")
	}
	if inSingleQuote {
		return nil, fmt.Errorf("unbalanced single quote")
	}
	if inDoubleQuote {
		return nil, fmt.Errorf("unbalanced double quote")
	}

	if inArg {
		args = append(args, current.String())
	}

	return args, nil
}
