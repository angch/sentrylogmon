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
			line:      `2023/10/27 10:00:00 [error] 1234#0: *5678 message`,
			wantOk:    true,
			wantTsStr: "2023/10/27 10:00:00",
			wantTime:  time.Date(2023, time.October, 27, 10, 0, 0, 0, time.UTC),
		},
		{
			name:   "Invalid length",
			line:   `2023/10/27`,
			wantOk: false,
		},
		{
			name:   "Invalid format separator",
			line:   `2023-10-27 10:00:00`,
			wantOk: false,
		},
		{
			name:   "Invalid month",
			line:   `2023/13/27 10:00:00`,
			wantOk: false,
		},
		{
			name:   "Invalid day",
			line:   `2023/10/32 10:00:00`,
			wantOk: false,
		},
		{
			name:   "Invalid hour",
			line:   `2023/10/27 24:00:00`,
			wantOk: false,
		},
		{
			name:   "Invalid minute",
			line:   `2023/10/27 10:60:00`,
			wantOk: false,
		},
		{
			name:   "Invalid char in year",
			line:   `20A3/10/27 10:00:00`,
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
