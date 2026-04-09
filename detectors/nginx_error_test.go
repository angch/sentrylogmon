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
		wantUnix  float64
	}{
		{
			name:      "Valid nginx error log",
			line:      "2023/10/27 10:00:00 [error] 123#123: *456 connect() failed",
			wantOk:    true,
			wantTsStr: "2023/10/27 10:00:00",
			wantUnix:  float64(time.Date(2023, 10, 27, 10, 0, 0, 0, time.UTC).Unix()),
		},
		{
			name:      "Invalid length",
			line:      "2023/10",
			wantOk:    false,
		},
		{
			name:      "Invalid format",
			line:      "2023-10-27 10:00:00 [error] 123#123",
			wantOk:    false,
		},
		{
			name:      "Invalid month",
			line:      "2023/13/27 10:00:00 [error]",
			wantOk:    false,
		},
		{
			name:      "Invalid day",
			line:      "2023/10/32 10:00:00 [error]",
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, tsStr, ok := ParseNginxError([]byte(tt.line))
			if ok != tt.wantOk {
				t.Errorf("ParseNginxError() ok = %v, want %v", ok, tt.wantOk)
				return
			}
			if !ok {
				return
			}
			if tsStr != tt.wantTsStr {
				t.Errorf("ParseNginxError() tsStr = %v, want %v", tsStr, tt.wantTsStr)
			}
			if ts != tt.wantUnix {
				t.Errorf("ParseNginxError() ts = %v, want %v", ts, tt.wantUnix)
			}
		})
	}
}
