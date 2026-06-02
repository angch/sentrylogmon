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
		year      int
		month     time.Month
		day       int
		hour      int
		min       int
		sec       int
	}{
		{
			name:      "Valid nginx error timestamp",
			line:      "2023/10/27 10:00:00 [error] 123#123: *1 open() failed",
			wantOk:    true,
			wantTsStr: "2023/10/27 10:00:00",
			year:      2023, month: time.October, day: 27, hour: 10, min: 0, sec: 0,
		},
		{
			name:   "Too short",
			line:   "2023/10/27",
			wantOk: false,
		},
		{
			name:   "Invalid format (missing slashes)",
			line:   "2023-10-27 10:00:00 [error]",
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, tsStr, ok := ParseNginxError([]byte(tt.line))
			if ok != tt.wantOk {
				t.Errorf("ParseNginxError() ok = %v, want %v", ok, tt.wantOk)
				return
			}
			if ok {
				if tsStr != tt.wantTsStr {
					t.Errorf("ParseNginxError() tsStr = %q, want %q", tsStr, tt.wantTsStr)
				}
				expectedTs := float64(time.Date(tt.year, tt.month, tt.day, tt.hour, tt.min, tt.sec, 0, time.UTC).Unix())
				if ts != expectedTs {
					t.Errorf("ParseNginxError() ts = %v, want %v", ts, expectedTs)
				}
			}
		})
	}
}
