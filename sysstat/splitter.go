package sysstat

import (
	"fmt"
	"strings"
	"unicode"
)

// SplitCommand parses a command string into arguments, handling quotes and escapes.
// It supports:
// - Single quotes: 'foo bar' -> "foo bar" (literal, no escapes inside)
// - Double quotes: "foo bar" -> "foo bar" (supports \" escaping)
// - Backslash escapes: \  -> " " (escaped space), \\ -> "\"
func SplitCommand(input string) ([]string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return []string{}, nil
	}

	var args []string
	var currentArg strings.Builder

	inSingleQuote := false
	inDoubleQuote := false
	escaped := false
	// Track if we have started processing a token.
	// This distinguishes between "no token yet" and "empty token" (e.g. from "")
	hasToken := false

	for _, r := range input {
		if escaped {
			currentArg.WriteRune(r)
			escaped = false
			hasToken = true
			continue
		}

		if r == '\\' {
			if inSingleQuote {
				currentArg.WriteRune(r) // Backslash is literal in single quotes
				hasToken = true
			} else {
				escaped = true
				hasToken = true // Backslash starts a token even if next char is pending
			}
			continue
		}

		if inSingleQuote {
			if r == '\'' {
				inSingleQuote = false
			} else {
				currentArg.WriteRune(r)
			}
			hasToken = true // Anything inside quotes contributes to token
			continue
		}

		if inDoubleQuote {
			if r == '"' {
				inDoubleQuote = false
			} else {
				currentArg.WriteRune(r)
			}
			hasToken = true // Anything inside quotes contributes to token
			continue
		}

		// Not in quote or escaped
		if r == '\'' {
			inSingleQuote = true
			hasToken = true // Opening a quote starts a token
			continue
		}

		if r == '"' {
			inDoubleQuote = true
			hasToken = true // Opening a quote starts a token
			continue
		}

		if unicode.IsSpace(r) {
			if hasToken {
				args = append(args, currentArg.String())
				currentArg.Reset()
				hasToken = false
			}
			continue
		}

		currentArg.WriteRune(r)
		hasToken = true
	}

	if escaped {
		return nil, fmt.Errorf("command ends with trailing backslash")
	}
	if inSingleQuote {
		return nil, fmt.Errorf("unterminated single quote")
	}
	if inDoubleQuote {
		return nil, fmt.Errorf("unterminated double quote")
	}

	// If we ended with a token in progress (even empty one like ""), add it
	if hasToken {
		args = append(args, currentArg.String())
	}

	return args, nil
}
