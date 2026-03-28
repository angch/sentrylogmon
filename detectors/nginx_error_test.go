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
			line:      "2023/10/27 10:00:00 [error] 1234#0: *1 open()",
			wantOk:    true,
			wantTsStr: "2023/10/27 10:00:00",
			wantTime:  time.Date(2023, time.October, 27, 10, 0, 0, 0, time.UTC),
		},
		{
			name:      "Valid with different time",
			line:      "2024/01/01 00:00:01 [crit] ...",
			wantOk:    true,
			wantTsStr: "2024/01/01 00:00:01",
			wantTime:  time.Date(2024, time.January, 1, 0, 0, 1, 0, time.UTC),
		},
		{
			name:   "Invalid format - dashes",
			line:   "2023-10-27 10:00:00 [error]",
			wantOk: false,
		},
		{
			name:   "Invalid format - space instead of slash",
			line:   "2023 10 27 10:00:00 [error]",
			wantOk: false,
		},
		{
			name:   "Short line",
			line:   "2023/10/27",
			wantOk: false,
		},
		{
			name:   "Empty",
			line:   "",
			wantOk: false,
		},
		{
			name:   "Invalid month",
			line:   "2023/13/27 10:00:00 [error]",
			wantOk: false,
		},
		{
			name:   "Invalid day",
			line:   "2023/10/32 10:00:00 [error]",
			wantOk: false,
		},
		{
			name:   "Invalid hour",
			line:   "2023/10/27 25:00:00 [error]",
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

				// Nginx error log timestamps don't include timezone, time.Parse defaults to UTC if no location is parsed?
				// Actually time.Parse uses UTC if no timezone information is present.

				expectedTs := float64(tt.wantTime.Unix()) + float64(tt.wantTime.Nanosecond())/1e9
				if ts != expectedTs {
					t.Errorf("ParseNginxError() ts = %v, want %v", ts, expectedTs)
				}
			}
		})
	}
}
