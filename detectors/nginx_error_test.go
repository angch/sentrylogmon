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
	}{
		{
			name:      "valid",
			line:      "2023/10/27 10:00:00 [error] 123#123: *456 message",
			wantOk:    true,
			wantTsStr: "2023/10/27 10:00:00",
		},
		{
			name:      "invalid length",
			line:      "2023/10/27",
			wantOk:    false,
		},
		{
			name:      "invalid format",
			line:      "2023-10-27 10:00:00 [error]",
			wantOk:    false,
		},
		{
			name:      "invalid components",
			line:      "2023/13/27 25:00:00 [error]",
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

			// Compare with time.Parse
			expectedTime, _ := time.Parse("2006/01/02 15:04:05", tt.wantTsStr)
			expectedTs := float64(expectedTime.Unix()) + float64(expectedTime.Nanosecond())/1e9
			if ts != expectedTs {
				t.Errorf("ParseNginxError() ts = %v, want %v", ts, expectedTs)
			}
		})
	}
}
