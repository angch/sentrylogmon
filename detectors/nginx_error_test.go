package detectors

import (
	"testing"
)

func TestParseNginxError(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expectedOK bool
	}{
		{
			name:       "Valid",
			input:      "2023/10/27 10:00:00 [error] 1234#0: *1 open() failed",
			expectedOK: true,
		},
		{
			name:       "Invalid Format",
			input:      "2023-10-27 10:00:00 [error]",
			expectedOK: false,
		},
		{
			name:       "Invalid Month",
			input:      "2023/13/27 10:00:00 [error]",
			expectedOK: false,
		},
		{
			name:       "Invalid Day",
			input:      "2023/10/32 10:00:00 [error]",
			expectedOK: false,
		},
		{
			name:       "Invalid Hour",
			input:      "2023/10/27 25:00:00 [error]",
			expectedOK: false,
		},
		{
			name:       "Invalid Minute",
			input:      "2023/10/27 10:60:00 [error]",
			expectedOK: false,
		},
		{
			name:       "Invalid Second",
			input:      "2023/10/27 10:00:61 [error]",
			expectedOK: false,
		},
		{
			name:       "Negative Year",
			input:      "-001/10/27 10:00:00 [error]",
			expectedOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, ok := ParseNginxError([]byte(tt.input))
			if ok != tt.expectedOK {
				t.Errorf("ParseNginxError() ok = %v, want %v", ok, tt.expectedOK)
			}
		})
	}
}
