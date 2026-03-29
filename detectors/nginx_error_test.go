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
			name:      "Valid timestamp",
			line:      "2023/10/27 10:00:00 [error] ...",
			wantOk:    true,
			wantTsStr: "2023/10/27 10:00:00",
			wantTime:  time.Date(2023, 10, 27, 10, 0, 0, 0, time.UTC),
		},
		{
			name:      "Valid timestamp with different time",
			line:      "2024/01/01 00:00:01 [error]",
			wantOk:    true,
			wantTsStr: "2024/01/01 00:00:01",
			wantTime:  time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC),
		},
		{
			name:   "Invalid format - dashes",
			line:   "2023-10-27 10:00:00",
			wantOk: false,
		},
		{
			name:   "Invalid format - no space",
			line:   "2023/10/27T10:00:00",
			wantOk: false,
		},
		{
			name:   "Short line",
			line:   "2023/10/27",
			wantOk: false,
		},
		{
			name:   "Garbage",
			line:   "Not a timestamp",
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
					t.Errorf("ParseNginxError() tsStr = %v, want %v", tsStr, tt.wantTsStr)
				}
				// Allow small float error or convert back to int64 for comparison
				gotTime := time.Unix(int64(ts), int64((ts-float64(int64(ts)))*1e9))
				// Using UTC because ParseNginxError (via time.Parse) assumes UTC if no timezone is present?
				// Wait, time.Parse("2006/01/02 15:04:05") returns UTC.

				if !gotTime.Equal(tt.wantTime) {
					t.Errorf("ParseNginxError() time = %v, want %v", gotTime, tt.wantTime)
				}
			}
		})
	}
}
