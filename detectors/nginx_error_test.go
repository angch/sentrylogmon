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
			name:      "Valid edge of month",
			line:      "2023/02/28 23:59:59 [error]",
			wantOk:    true,
			wantTsStr: "2023/02/28 23:59:59",
			wantTime:  time.Date(2023, time.February, 28, 23, 59, 59, 0, time.UTC),
		},
		{
			name:      "Invalid separator",
			line:      "2023-10-27 10:00:00",
			wantOk:    false,
		},
		{
			name:      "Invalid year",
			line:      "xxxx/10/27 10:00:00",
			wantOk:    false,
		},
		{
			name:      "Invalid month",
			line:      "2023/13/27 10:00:00",
			wantOk:    false, // time.Parse handles this
		},
		{
			name:      "Invalid day",
			line:      "2023/10/32 10:00:00",
			wantOk:    false, // time.Parse handles this
		},
		{
			name:      "Invalid time",
			line:      "2023/10/27 25:00:00",
			wantOk:    false, // time.Parse handles this
		},
		{
			name:      "Short line",
			line:      "2023/10/27",
			wantOk:    false,
		},
		{
			name:      "Empty",
			line:      "",
			wantOk:    false,
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
