package config

import (
	"strings"
	"testing"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		expectErr bool
		errContains string
	}{
		{
			name: "Valid Config",
			config: Config{
				Sentry: SentryConfig{
					DSN: "https://example.com",
				},
				Monitors: []MonitorConfig{
					{
						Name: "test-monitor",
						Type: "file",
						Path: "/var/log/test.log",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "Missing Sentry DSN",
			config: Config{
				Sentry: SentryConfig{
					DSN: "",
				},
				Monitors: []MonitorConfig{
					{
						Name: "test-monitor",
						Type: "file",
						Path: "/var/log/test.log",
					},
				},
			},
			expectErr: true,
			errContains: "Sentry DSN is required",
		},
		{
			name: "No Monitors",
			config: Config{
				Sentry: SentryConfig{
					DSN: "https://example.com",
				},
				Monitors: []MonitorConfig{},
			},
			expectErr: true,
			errContains: "no monitors configured",
		},
		{
			name: "Monitor Missing Name",
			config: Config{
				Sentry: SentryConfig{
					DSN: "https://example.com",
				},
				Monitors: []MonitorConfig{
					{
						Name: "",
						Type: "file",
						Path: "/var/log/test.log",
					},
				},
			},
			expectErr: true,
			errContains: "monitor name is required",
		},
		{
			name: "Monitor Invalid Type",
			config: Config{
				Sentry: SentryConfig{
					DSN: "https://example.com",
				},
				Monitors: []MonitorConfig{
					{
						Name: "test",
						Type: "invalid-type",
					},
				},
			},
			expectErr: true,
			errContains: "unknown monitor type",
		},
		{
			name: "File Monitor Missing Path",
			config: Config{
				Sentry: SentryConfig{
					DSN: "https://example.com",
				},
				Monitors: []MonitorConfig{
					{
						Name: "test",
						Type: "file",
						Path: "",
					},
				},
			},
			expectErr: true,
			errContains: "path is required",
		},
		{
			name: "Command Monitor Missing Args",
			config: Config{
				Sentry: SentryConfig{
					DSN: "https://example.com",
				},
				Monitors: []MonitorConfig{
					{
						Name: "test",
						Type: "command",
						Args: "",
					},
				},
			},
			expectErr: true,
			errContains: "command args are required",
		},
		{
			name: "Invalid Pattern Regex",
			config: Config{
				Sentry: SentryConfig{
					DSN: "https://example.com",
				},
				Monitors: []MonitorConfig{
					{
						Name:    "test",
						Type:    "dmesg",
						Pattern: "(",
					},
				},
			},
			expectErr: true,
			errContains: "invalid pattern regex",
		},
		{
			name: "Invalid Exclude Pattern Regex",
			config: Config{
				Sentry: SentryConfig{
					DSN: "https://example.com",
				},
				Monitors: []MonitorConfig{
					{
						Name:           "test",
						Type:           "dmesg",
						ExcludePattern: "[",
					},
				},
			},
			expectErr: true,
			errContains: "invalid exclude_pattern regex",
		},
		{
			name: "Invalid MaxInactivity Duration",
			config: Config{
				Sentry: SentryConfig{
					DSN: "https://example.com",
				},
				Monitors: []MonitorConfig{
					{
						Name:          "test",
						Type:          "dmesg",
						MaxInactivity: "invalid",
					},
				},
			},
			expectErr: true,
			errContains: "invalid max_inactivity",
		},
		{
			name: "Invalid RateLimitWindow Duration",
			config: Config{
				Sentry: SentryConfig{
					DSN: "https://example.com",
				},
				Monitors: []MonitorConfig{
					{
						Name:            "test",
						Type:            "dmesg",
						RateLimitWindow: "2days", // time.ParseDuration doesn't support days without definition, usually only h/m/s/ms/etc.
					},
				},
			},
			// Wait, "2days" is indeed invalid for ParseDuration (supports h, m, s, ms, us, ns).
			// If it was valid, I should use something like "invalid".
			// "2days" is definitely invalid.
			expectErr: true,
			errContains: "invalid rate_limit_window",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr {
				if err == nil {
					t.Errorf("Validate() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Validate() error = %v, want substring %s", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}
