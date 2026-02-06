package detectors

import (
	"testing"
	"time"
)

func TestParseNginxAccess(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantOk    bool
		wantTsStr string
		wantTime  time.Time
	}{
		{
			name:      "Valid",
			line:      `127.0.0.1 - - [27/Oct/2023:10:00:00 +0000] "GET / HTTP/1.1"`,
			wantOk:    true,
			wantTsStr: "27/Oct/2023:10:00:00 +0000",
			wantTime:  time.Date(2023, time.October, 27, 10, 0, 0, 0, time.UTC),
		},
		{
			name:      "Valid with different timezone",
			line:      `127.0.0.1 - - [27/Oct/2023:10:00:00 -0700] "GET / HTTP/1.1"`,
			wantOk:    true,
			wantTsStr: "27/Oct/2023:10:00:00 -0700",
			wantTime:  time.Date(2023, time.October, 27, 17, 0, 0, 0, time.UTC), // 10:00 -0700 is 17:00 UTC
		},
		{
			name:      "Valid at start",
			line:      `[27/Oct/2023:10:00:00 +0000] "GET / HTTP/1.1"`,
			wantOk:    true,
			wantTsStr: "27/Oct/2023:10:00:00 +0000",
			wantTime:  time.Date(2023, time.October, 27, 10, 0, 0, 0, time.UTC),
		},
		{
			name:   "Invalid month",
			line:   `127.0.0.1 - - [27/Foo/2023:10:00:00 +0000]`,
			wantOk: false,
		},
		{
			name:   "Invalid format",
			line:   `127.0.0.1 - - [27/Oct/2023 10:00:00 +0000]`,
			wantOk: false,
		},
		{
			name:   "No closing bracket",
			line:   `127.0.0.1 - - [27/Oct/2023:10:00:00 +0000`,
			wantOk: false,
		},
		{
			name:   "Short line",
			line:   `[27/Oct]`,
			wantOk: false,
		},
		{
			name:   "Empty",
			line:   ``,
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, tsStr, ok := ParseNginxAccess([]byte(tt.line))
			if ok != tt.wantOk {
				t.Errorf("ParseNginxAccess() ok = %v, want %v", ok, tt.wantOk)
			}
			if ok {
				if tsStr != tt.wantTsStr {
					t.Errorf("ParseNginxAccess() tsStr = %q, want %q", tsStr, tt.wantTsStr)
				}

				// Allow small float precision diff
				expectedTs := float64(tt.wantTime.Unix()) + float64(tt.wantTime.Nanosecond())/1e9
				if ts != expectedTs {
					t.Errorf("ParseNginxAccess() ts = %v, want %v", ts, expectedTs)
				}
			}
		})
	}
}
