package detectors

import (
	"testing"
	"time"
)

func TestParseSyslogTimestamp(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantOk     bool
		wantTsStr  string
		timeOffset time.Duration // Rough offset from now (0 means current year)
	}{
		{
			name:      "Valid padded day",
			line:      "<34>Oct  1 10:00:00 myhost",
			wantOk:    true,
			wantTsStr: "Oct  1 10:00:00",
		},
		{
			name:      "Valid 2-digit day",
			line:      "Oct 10 10:00:00 myhost",
			wantOk:    true,
			wantTsStr: "Oct 10 10:00:00",
		},
		{
			name:      "Invalid month",
			line:      "Foo 10 10:00:00",
			wantOk:    false,
		},
		{
			name:      "Invalid day",
			line:      "Oct 32 10:00:00",
			wantOk:    false,
		},
		{
			name:      "Invalid time",
			line:      "Oct 10 25:00:00",
			wantOk:    false,
		},
		{
			name:      "Short line",
			line:      "Oct 1",
			wantOk:    false,
		},
		{
			name:      "With Priority",
			line:      "<1>Oct 10 10:00:00",
			wantOk:    true,
			wantTsStr: "Oct 10 10:00:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, tsStr, ok := ParseSyslogTimestamp([]byte(tt.line))
			if ok != tt.wantOk {
				t.Errorf("ParseSyslogTimestamp() ok = %v, want %v", ok, tt.wantOk)
			}
			if ok {
				if tsStr != tt.wantTsStr {
					t.Errorf("ParseSyslogTimestamp() tsStr = %v, want %v", tsStr, tt.wantTsStr)
				}
				if ts == 0 {
					t.Errorf("ParseSyslogTimestamp() ts = 0")
				}
			}
		})
	}
}

func TestParseNginxError(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantOk    bool
		wantTsStr string
	}{
		{
			name:      "Valid",
			line:      "2023/10/27 10:00:00 [error] ...",
			wantOk:    true,
			wantTsStr: "2023/10/27 10:00:00",
		},
		{
			name:      "Invalid month",
			line:      "2023/13/27 10:00:00 [error] ...",
			wantOk:    false,
		},
		{
			name:      "Invalid day",
			line:      "2023/10/32 10:00:00 [error] ...",
			wantOk:    false,
		},
		{
			name:      "Invalid hour",
			line:      "2023/10/27 24:00:00 [error] ...",
			wantOk:    false,
		},
		{
			name:      "Invalid minute",
			line:      "2023/10/27 10:60:00 [error] ...",
			wantOk:    false,
		},
		{
			name:      "Invalid second",
			line:      "2023/10/27 10:00:61 [error] ...",
			wantOk:    false,
		},
		{
			name:      "Short line",
			line:      "2023/10/27",
			wantOk:    false,
		},
		{
			name:      "Invalid format",
			line:      "2023-10-27 10:00:00 [error] ...",
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
					t.Errorf("ParseNginxError() tsStr = %v, want %v", tsStr, tt.wantTsStr)
				}
				if ts == 0 {
					t.Errorf("ParseNginxError() ts = 0")
				}
			}
		})
	}
}
