package sysstat

import (
	"errors"
	"strings"
	"unicode"
)

// SplitCommand parses a command string into arguments, respecting quotes (single and double) and backslash escapes.
// This is used for correctly splitting command arguments that may contain spaces or sensitive information.
func SplitCommand(s string) ([]string, error) {
	var args []string
	var currentArg strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false
	hasToken := false

	for _, r := range s {
		if escaped {
			currentArg.WriteRune(r)
			escaped = false
			hasToken = true
			continue
		}

		if r == '\\' {
			if inSingleQuote {
				currentArg.WriteRune(r)
				hasToken = true
			} else {
				escaped = true
				hasToken = true
			}
			continue
		}

		if inSingleQuote {
			if r == '\'' {
				inSingleQuote = false
				hasToken = true
			} else {
				currentArg.WriteRune(r)
				hasToken = true
			}
			continue
		}

		if inDoubleQuote {
			if r == '"' {
				inDoubleQuote = false
				hasToken = true
			} else {
				currentArg.WriteRune(r)
				hasToken = true
			}
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

		if r == '\'' {
			inSingleQuote = true
			hasToken = true
			continue
		}

		if r == '"' {
			inDoubleQuote = true
			hasToken = true
			continue
		}

		currentArg.WriteRune(r)
		hasToken = true
	}

	if escaped || inSingleQuote || inDoubleQuote {
		return nil, errors.New("unclosed quote or escape sequence")
	}

	if hasToken {
		args = append(args, currentArg.String())
	}

	return args, nil
}
