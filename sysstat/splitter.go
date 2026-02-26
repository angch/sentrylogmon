package sysstat

import (
	"fmt"
	"strings"
)

// SplitCommand splits a command string into arguments, respecting shell quoting rules.
// It handles single quotes ('), double quotes ("), and backslash escapes (\).
func SplitCommand(command string) ([]string, error) {
	var args []string
	var current strings.Builder
	var inSingleQuote, inDoubleQuote bool
	var escaped bool
	var hasToken bool // To handle empty quoted strings like ""

	for _, r := range command {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' {
			if inSingleQuote {
				current.WriteRune(r) // Backslash is literal inside single quotes
			} else {
				escaped = true
				hasToken = true // Backslash implies a token is being built
			}
			continue
		}

		if inSingleQuote {
			if r == '\'' {
				inSingleQuote = false
			} else {
				current.WriteRune(r)
			}
			continue
		}

		if inDoubleQuote {
			if r == '"' {
				inDoubleQuote = false
			} else {
				current.WriteRune(r)
			}
			continue
		}

		switch r {
		case '\'':
			inSingleQuote = true
			hasToken = true
		case '"':
			inDoubleQuote = true
			hasToken = true
		case ' ', '\t':
			if hasToken {
				args = append(args, current.String())
				current.Reset()
				hasToken = false
			}
		default:
			current.WriteRune(r)
			hasToken = true
		}
	}

	if escaped {
		return nil, fmt.Errorf("unexpected end of command: trailing backslash")
	}
	if inSingleQuote || inDoubleQuote {
		return nil, fmt.Errorf("unexpected end of command: unclosed quote")
	}

	if hasToken {
		args = append(args, current.String())
	}

	return args, nil
}
