package detectors

import (
	"testing"
	"time"
)

func TestParseISO8601(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantOk    bool
		wantTsStr string
		wantTs    float64 // If non-zero, check approximately
	}{
		{
			name:      "Standard RFC3339",
			line:      "2023-10-27T10:00:00Z message",
			wantOk:    true,
			wantTsStr: "2023-10-27T10:00:00Z",
		},
		{
			name:      "RFC3339 with fractional seconds",
			line:      "2023-10-27T10:00:00.123Z message",
			wantOk:    true,
			wantTsStr: "2023-10-27T10:00:00.123Z",
		},
		{
			name:      "Space separator, no timezone",
			line:      "2023-10-27 10:00:00 message",
			wantOk:    true,
			wantTsStr: "2023-10-27 10:00:00",
		},
		{
			name:      "Space separator, with timezone",
			line:      "2023-10-27 10:00:00+00:00 message",
			wantOk:    true,
			wantTsStr: "2023-10-27 10:00:00+00:00",
		},
		{
			name:      "Space separator, with fractional and timezone",
			line:      "2023-10-27 10:00:00.999-05:00 message",
			wantOk:    true,
			wantTsStr: "2023-10-27 10:00:00.999-05:00",
		},
		{
			name:      "Invalid separator",
			line:      "2023/10/27 10:00:00 message",
			wantOk:    false,
		},
		{
			name:      "Invalid time separator",
			line:      "2023-10-27 10-00-00 message",
			wantOk:    false,
		},
		{
			name:      "Short string",
			line:      "2023-10-27",
			wantOk:    false,
		},
		{
			name:      "Bad month",
			line:      "2023-13-27T10:00:00Z",
			wantOk:    false, // time.Parse would catch this
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, tsStr, ok := ParseISO8601([]byte(tt.line))
			if ok != tt.wantOk {
				t.Errorf("ParseISO8601() ok = %v, want %v", ok, tt.wantOk)
			}
			if ok {
				if tsStr != tt.wantTsStr {
					t.Errorf("ParseISO8601() tsStr = %q, want %q", tsStr, tt.wantTsStr)
				}
				if ts == 0 {
					t.Errorf("ParseISO8601() ts = 0")
				}

				// Optional: Verify strict timestamp match
				// We can parse expected string with time.Parse to compare float values
				var expectedTime time.Time
				var err error
				if tt.line[10] == 'T' {
					expectedTime, err = time.Parse(time.RFC3339Nano, tt.wantTsStr)
				} else {
					if len(tt.wantTsStr) > 19 {
						// Heuristic for space separated with TZ
						expectedTime, err = time.Parse("2006-01-02 15:04:05.999999999Z07:00", tt.wantTsStr)
					} else {
						expectedTime, err = time.Parse("2006-01-02 15:04:05", tt.wantTsStr)
					}
				}

				if err == nil {
					expectedTs := float64(expectedTime.Unix()) + float64(expectedTime.Nanosecond())/1e9
					// Allow small floating point difference
					if diff := ts - expectedTs; diff < -0.000001 || diff > 0.000001 {
						t.Errorf("ParseISO8601() ts = %v, want %v", ts, expectedTs)
					}
				}
			}
		})
	}
}
