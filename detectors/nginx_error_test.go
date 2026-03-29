package detectors

import (
	"testing"
	"time"
)

func TestParseNginxError(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantOk    bool
		wantTsStr string
		wantTime  time.Time
	}{
		{
			name:      "Valid",
			line:      `2023/10/27 10:00:00 [error] 12345#0: *123 open() "/usr/share/nginx/html/missing" failed`,
			wantOk:    true,
			wantTsStr: "2023/10/27 10:00:00",
			wantTime:  time.Date(2023, time.October, 27, 10, 0, 0, 0, time.UTC),
		},
		{
			name:   "Invalid format",
			line:   `2023-10-27 10:00:00 [error]`,
			wantOk: false,
		},
		{
			name:   "Invalid month",
			line:   `2023/13/27 10:00:00 [error]`,
			wantOk: false,
		},
		{
			name:   "Invalid time",
			line:   `2023/10/27 25:00:00 [error]`,
			wantOk: false,
		},
		{
			name:   "Short line",
			line:   `2023/10/27 10:`,
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, tsStr, ok := ParseNginxError([]byte(tt.line))
			if ok != tt.wantOk {
				t.Errorf("ParseNginxError() ok = %v, want %v", ok, tt.wantOk)
			}
			if ok {
				if tsStr != tt.wantTsStr {
					t.Errorf("ParseNginxError() tsStr = %q, want %q", tsStr, tt.wantTsStr)
				}
				expectedTs := float64(tt.wantTime.Unix()) + float64(tt.wantTime.Nanosecond())/1e9
				if ts != expectedTs {
					t.Errorf("ParseNginxError() ts = %v, want %v", ts, expectedTs)
				}
			}
		})
	}
}
