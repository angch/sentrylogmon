package sysstat

import (
	"testing"
)

func TestSanitizeCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "Empty",
			input:    []string{},
			expected: "",
		},
		{
			name:     "No sensitive info",
			input:    []string{"ls", "-la", "/tmp"},
			expected: "ls -la /tmp",
		},
		{
			name:     "Flag with space",
			input:    []string{"mysql", "--password", "supersecret"},
			expected: "mysql --password [REDACTED]",
		},
		{
			name:     "Flag with equals",
			input:    []string{"curl", "--url", "http://example.com", "--header", "Authorization=Bearer 123"},
			expected: "curl --url http://example.com --header Authorization=Bearer 123", // Assuming Authorization isn't a known flag key, but maybe we should catch it?
		},
		{
			name:     "Token flag",
			input:    []string{"app", "--token", "abcdef"},
			expected: "app --token [REDACTED]",
		},
		{
			name:     "API Key equals",
			input:    []string{"app", "--api-key=12345"},
			expected: "app --api-key=[REDACTED]",
		},
		{
			name:     "Mixed flags",
			input:    []string{"run", "--verbose", "--password", "secret", "--count=5"},
			expected: "run --verbose --password [REDACTED] --count=5",
		},
		{
			name:     "Env var style",
			input:    []string{"env", "DB_PASSWORD=secret", "command"},
			expected: "env DB_PASSWORD=[REDACTED] command",
		},
		{
			name:     "Ambiguous -p",
			input:    []string{"ssh", "-p", "22", "user@host"},
			expected: "ssh -p 22 user@host",
		},
		{
			name:     "Sensitive suffix",
			input:    []string{"./prog", "--client_secret=xyz"},
			expected: "./prog --client_secret=[REDACTED]",
		},
		{
			name:     "Non-sensitive suffix",
			input:    []string{"./prog", "--use-keyboard=us"},
			expected: "./prog --use-keyboard=us",
		},
		{
			name:     "SSH Key flag",
			input:    []string{"ssh-agent", "--ssh-key=PRIVATE_KEY"},
			expected: "ssh-agent --ssh-key=[REDACTED]",
		},
		{
			name:     "Private Key flag",
			input:    []string{"app", "--private-key=SECRET"},
			expected: "app --private-key=[REDACTED]",
		},
		{
			name:     "Dot Key flag",
			input:    []string{"app", "--app.key=SECRET"},
			expected: "app --app.key=[REDACTED]",
		},
		{
			name:     "Signature flag",
			input:    []string{"verify", "--signature=SIG123"},
			expected: "verify --signature=[REDACTED]",
		},
		{
			name:     "Credential flag",
			input:    []string{"login", "--aws-credential=XYZ"},
			expected: "login --aws-credential=[REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeCommand(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeCommand() = %v, want %v", got, tt.expected)
			}
		})
	}
}
