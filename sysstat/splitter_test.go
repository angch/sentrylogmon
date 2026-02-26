package sysstat

import (
	"reflect"
	"testing"
)

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:  "Simple",
			input: "ls -la",
			want:  []string{"ls", "-la"},
		},
		{
			name:  "Quotes",
			input: `echo "hello world"`,
			want:  []string{"echo", "hello world"},
		},
		{
			name:  "Mixed Quotes",
			input: `echo "hello 'world'"`,
			want:  []string{"echo", "hello 'world'"},
		},
		{
			name:  "Single Quotes",
			input: `echo 'hello world'`,
			want:  []string{"echo", "hello world"},
		},
		{
			name:  "Escaped Quotes",
			input: `echo "hello \"world\""`,
			want:  []string{"echo", `hello "world"`},
		},
		{
			name:  "Escaped Spaces",
			input: `ls hello\ world`,
			want:  []string{"ls", "hello world"},
		},
		{
			name:  "Empty Strings",
			input: `echo "" ''`,
			want:  []string{"echo", "", ""},
		},
		{
			name:  "Empty Command",
			input: "",
			want:  nil,
		},
		{
			name:  "Multiple Spaces",
			input: "ls    -la",
			want:  []string{"ls", "-la"},
		},
		{
			name:    "Unclosed Quote",
			input:   `echo "hello`,
			wantErr: true,
		},
		{
			name:    "Trailing Backslash",
			input:   `echo hello\`,
			wantErr: true,
		},
		{
			name:  "Complex",
			input: `curl --header "Authorization: Bearer 123" -v`,
			want:  []string{"curl", "--header", "Authorization: Bearer 123", "-v"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitCommand(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}
