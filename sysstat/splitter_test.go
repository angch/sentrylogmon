package sysstat

import (
	"reflect"
	"testing"
)

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
		wantErr  bool
	}{
		{
			name:     "Empty",
			input:    "",
			expected: []string{},
			wantErr:  false,
		},
		{
			name:     "Simple",
			input:    "ls -la /tmp",
			expected: []string{"ls", "-la", "/tmp"},
			wantErr:  false,
		},
		{
			name:     "Multiple spaces",
			input:    "ls   -la    /tmp",
			expected: []string{"ls", "-la", "/tmp"},
			wantErr:  false,
		},
		{
			name:     "Double quotes",
			input:    "echo \"hello world\"",
			expected: []string{"echo", "hello world"},
			wantErr:  false,
		},
		{
			name:     "Single quotes",
			input:    "echo 'hello world'",
			expected: []string{"echo", "hello world"},
			wantErr:  false,
		},
		{
			name:     "Mixed quotes",
			input:    "echo \"hello 'world'\"",
			expected: []string{"echo", "hello 'world'"},
			wantErr:  false,
		},
		{
			name:     "Escaped quotes in double quotes",
			input:    "echo \"hello \\\"world\\\"\"",
			expected: []string{"echo", "hello \"world\""},
			wantErr:  false,
		},
		{
			name:     "Escaped backslash",
			input:    "echo \"foo\\\\bar\"",
			expected: []string{"echo", "foo\\bar"},
			wantErr:  false,
		},
		{
			name:     "Unterminated double quote",
			input:    "echo \"hello world",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Unterminated single quote",
			input:    "echo 'hello world",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Trailing backslash",
			input:    "echo hello\\",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Complex command",
			input:    "curl -H \"Authorization: Bearer token\" 'https://example.com'",
			expected: []string{"curl", "-H", "Authorization: Bearer token", "https://example.com"},
			wantErr:  false,
		},
		{
			name:     "Escaped space outside quotes",
			input:    "ls My\\ Documents",
			expected: []string{"ls", "My Documents"},
			wantErr:  false,
		},
		{
			name:     "Empty quoted string (single)",
			input:    "sh -c ''",
			expected: []string{"sh", "-c", ""},
			wantErr:  false,
		},
		{
			name:     "Empty quoted string (double)",
			input:    "sh -c \"\"",
			expected: []string{"sh", "-c", ""},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitCommand(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("SplitCommand() = %v, want %v", got, tt.expected)
			}
		})
	}
}
