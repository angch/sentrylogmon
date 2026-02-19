package sysstat

import (
	"fmt"
	"strings"
)

// SplitCommand splits a command string into arguments, respecting quotes and escapes.
// It supports single quotes, double quotes, and backslash escapes.
func SplitCommand(command string) ([]string, error) {
	var args []string
	var current strings.Builder
	var inSingle, inDouble bool
	var escaped bool
	var inArg bool

	for _, r := range command {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' {
			if inSingle {
				current.WriteRune(r)
			} else {
				escaped = true
				inArg = true
			}
			continue
		}

		if inSingle {
			if r == '\'' {
				inSingle = false
			} else {
				current.WriteRune(r)
			}
			continue
		}

		if inDouble {
			if r == '"' {
				inDouble = false
			} else {
				current.WriteRune(r)
			}
			continue
		}

		switch r {
		case ' ', '\t', '\n':
			if inArg {
				args = append(args, current.String())
				current.Reset()
				inArg = false
			}
		case '\'':
			inSingle = true
			inArg = true
		case '"':
			inDouble = true
			inArg = true
		default:
			current.WriteRune(r)
			inArg = true
		}
	}

	if escaped || inSingle || inDouble {
		return nil, fmt.Errorf("unterminated quote or escape")
	}

	if inArg {
		args = append(args, current.String())
	}

	return args, nil
}
