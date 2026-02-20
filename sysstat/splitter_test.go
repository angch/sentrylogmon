package sysstat

import (
	"reflect"
	"testing"
)

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
		hasError bool
	}{
		// Simple cases
		{"", nil, false},
		{"   ", nil, false},
		{"ls", []string{"ls"}, false},
		{"ls -l", []string{"ls", "-l"}, false},
		{"ls -l /tmp", []string{"ls", "-l", "/tmp"}, false},
		{"  ls   -l   ", []string{"ls", "-l"}, false},

		// Quoted strings
		{"echo \"hello world\"", []string{"echo", "hello world"}, false},
		{"echo 'hello world'", []string{"echo", "hello world"}, false},
		{"echo \"hello 'world'\"", []string{"echo", "hello 'world'"}, false},
		{"echo 'hello \"world\"'", []string{"echo", "hello \"world\""}, false},

		// Empty strings in quotes
		{"echo \"\"", []string{"echo", ""}, false},
		{"echo ''", []string{"echo", ""}, false},
		{"echo \"\" foo", []string{"echo", "", "foo"}, false},
		{"echo '' foo", []string{"echo", "", "foo"}, false},

		// Concatenation
		{"echo foo\"bar\"", []string{"echo", "foobar"}, false},
		{"echo \"foo\"bar", []string{"echo", "foobar"}, false},
		{"echo 'foo'bar", []string{"echo", "foobar"}, false},

		// Escapes outside quotes
		{"echo foo\\ bar", []string{"echo", "foo bar"}, false},
		{"echo \\\"foo\\\"", []string{"echo", "\"foo\""}, false},
		{"echo \\\\", []string{"echo", "\\"}, false},

		// Escapes inside double quotes
		{"echo \"foo\\\"bar\"", []string{"echo", "foo\"bar"}, false},
		{"echo \"foo\\\\bar\"", []string{"echo", "foo\\bar"}, false},
		{"echo \"foo\\bar\"", []string{"echo", "foo\\bar"}, false}, // Backslash not escaping special char is preserved

		// Escapes inside single quotes (literal)
		{"echo 'foo\\bar'", []string{"echo", "foo\\bar"}, false},
		{"echo 'foo\\'bar'", nil, true}, // Unbalanced quote (backslash is literal, so quote closes string, 'bar' is extra, wait no)
		// 'foo\'bar' -> 'foo\' -> string ends. bar' -> bar starts. -> foo\bar. Correct.
		// Wait, inside single quotes backslash is LITERAL.
		// So 'foo\'bar' means:
		// ' starts quote.
		// foo\ are chars.
		// ' ends quote.
		// bar' are chars.
		// Result: foo\bar'.
		// But in standard shell: 'foo\'bar' -> > (multiline) because ' matches '... wait.
		// 'foo\' -> string "foo\".
		// bar' -> string "bar'".
		// Result: "foo\bar'".
		// My parser:
		// ' -> inSingleQuote=true.
		// foo\ -> writes foo\.
		// ' -> inSingleQuote=false.
		// bar' -> writes bar. ' -> inSingleQuote=true.
		// End of string. inSingleQuote=true. Error: Unbalanced.
		{"echo 'foo\\'bar'", nil, true},

		// Unbalanced quotes
		{"echo \"foo", nil, true},
		{"echo 'foo", nil, true},
		{"echo foo\\", nil, true}, // Trailing backslash
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := SplitCommand(tt.input)
			if (err != nil) != tt.hasError {
				t.Errorf("SplitCommand(%q) error = %v, hasError %v", tt.input, err, tt.hasError)
				return
			}
			if !tt.hasError && !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("SplitCommand(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
