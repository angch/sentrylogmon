package detectors

import (
	"testing"
)

func TestParseNginxError(t *testing.T) {
	tests := []struct {
		name      string
		line      []byte
		wantTs    float64
		wantTsStr string
		wantOk    bool
	}{
		{
			name:      "Valid",
			line:      []byte("2023/10/27 10:00:00 [error] 1234#5678: *9012 connect() failed"),
			wantTs:    1698400800,
			wantTsStr: "2023/10/27 10:00:00",
			wantOk:    true,
		},
		{
			name:   "Short line",
			line:   []byte("2023/10/27"),
			wantOk: false,
		},
		{
			name:   "Invalid format",
			line:   []byte("2023-10-27 10:00:00 [error]"),
			wantOk: false,
		},
		{
			name:   "Invalid month",
			line:   []byte("2023/13/27 10:00:00 [error]"),
			wantOk: false,
		},
		{
			name:   "Invalid day",
			line:   []byte("2023/10/32 10:00:00 [error]"),
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, tsStr, ok := ParseNginxError(tt.line)
			if ok != tt.wantOk {
				t.Errorf("ParseNginxError() ok = %v, want %v", ok, tt.wantOk)
				return
			}
			if !ok {
				return
			}
			if ts != tt.wantTs {
				t.Errorf("ParseNginxError() ts = %v, want %v", ts, tt.wantTs)
			}
			if tsStr != tt.wantTsStr {
				t.Errorf("ParseNginxError() tsStr = %v, want %v", tsStr, tt.wantTsStr)
			}
		})
	}
}
