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
			input:    "",
			expected: nil,
			wantErr:  false,
		},
		{
			input:    "simple command",
			expected: []string{"simple", "command"},
			wantErr:  false,
		},
		{
			input:    "  multiple   spaces  ",
			expected: []string{"multiple", "spaces"},
			wantErr:  false,
		},
		{
			input:    `command "with quotes"`,
			expected: []string{"command", "with quotes"},
			wantErr:  false,
		},
		{
			input:    `command 'single quotes'`,
			expected: []string{"command", "single quotes"},
			wantErr:  false,
		},
		{
			input:    `mixed "quotes" 'types'`,
			expected: []string{"mixed", "quotes", "types"},
			wantErr:  false,
		},
		{
			input:    `escaped \"quote\"`,
			expected: []string{"escaped", `"quote"`},
			wantErr:  false,
		},
		{
			input:    `escaped\ space`,
			expected: []string{"escaped space"},
			wantErr:  false,
		},
		{
			input:    `nested "quote 'inside' double"`,
			expected: []string{"nested", "quote 'inside' double"},
			wantErr:  false,
		},
		{
			input:    `nested 'quote "inside" single'`,
			expected: []string{"nested", `quote "inside" single`},
			wantErr:  false,
		},
		{
			input:    `key="value with spaces"`,
			expected: []string{`key=value with spaces`},
			wantErr:  false,
		},
		{
			input:    `--token="secret value"`,
			expected: []string{`--token=secret value`},
			wantErr:  false,
		},
		{
			input:    `--token "secret value"`,
			expected: []string{`--token`, `secret value`},
			wantErr:  false,
		},
		{
			input:    `unterminated "quote`,
			expected: nil,
			wantErr:  true,
		},
		{
			input:    `unterminated 'quote`,
			expected: nil,
			wantErr:  true,
		},
		{
			input:    `trailing escape \`,
			expected: nil,
			wantErr:  true,
		},
		{
			input:    `empty "" string`,
			expected: []string{"empty", "", "string"},
			wantErr:  false,
		},
		{
			input:    `concatenated"quotes"`,
			expected: []string{"concatenatedquotes"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		got, err := SplitCommand(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("SplitCommand(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("SplitCommand(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
