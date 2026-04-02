package sysstat

import (
	"reflect"
	"testing"
)

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
		wantErr  bool
	}{
		{
			input:    "command arg1 arg2",
			expected: []string{"command", "arg1", "arg2"},
			wantErr:  false,
		},
		{
			input:    "  command   arg1  ",
			expected: []string{"command", "arg1"},
			wantErr:  false,
		},
		{
			input:    "command \"arg with spaces\"",
			expected: []string{"command", "arg with spaces"},
			wantErr:  false,
		},
		{
			input:    "command 'arg with spaces'",
			expected: []string{"command", "arg with spaces"},
			wantErr:  false,
		},
		{
			input:    "command \"arg's quote\"",
			expected: []string{"command", "arg's quote"},
			wantErr:  false,
		},
		{
			input:    "command 'arg\"s quote'",
			expected: []string{"command", "arg\"s quote"},
			wantErr:  false,
		},
		{
			input:    "command arg\\ with\\ spaces",
			expected: []string{"command", "arg with spaces"},
			wantErr:  false,
		},
		{
			input:    "command \"escaped \\\"quote\\\"\"",
			expected: []string{"command", "escaped \"quote\""},
			wantErr:  false,
		},
		{
			input:    "command 'escaped backslash \\'", // Backslash is literal in single quotes
			expected: []string{"command", "escaped backslash \\"},
			wantErr:  false,
		},
		{
			input:    "command \"escaped backslash \\\\\"",
			expected: []string{"command", "escaped backslash \\"},
			wantErr:  false,
		},
		{
			input:    "command mixed\"quotes\"'and'spaces",
			expected: []string{"command", "mixedquotesandspaces"},
			wantErr:  false,
		},
		{
			input:    "command \"\"",
			expected: []string{"command", ""},
			wantErr:  false,
		},
		{
			input:    "",
			expected: nil,
			wantErr:  false,
		},
		{
			input:    "   ",
			expected: nil,
			wantErr:  false,
		},
		{
			input:    "command \"unclosed quote",
			expected: nil,
			wantErr:  true,
		},
		{
			input:    "command 'unclosed quote",
			expected: nil,
			wantErr:  true,
		},
		{
			input:    "command escaped\\",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := SplitCommand(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("SplitCommand() = %v, want %v", got, tt.expected)
			}
		})
	}
}
