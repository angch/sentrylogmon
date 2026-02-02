package monitor

import (
	"testing"
)

func TestExtractTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantTS   bool // check if timestamp > 0
		wantText string
	}{
		{
			name:     "Dmesg",
			line:     "[1234.5678] some log message",
			wantTS:   true,
			wantText: "1234.5678",
		},
		{
			name:     "ISO8601",
			line:     "2023-10-27T10:00:00Z some log message",
			wantTS:   true,
			wantText: "2023-10-27T10:00:00Z",
		},
		{
			name:     "Syslog",
			line:     "Oct 27 10:00:00 host process: message",
			wantTS:   true,
			wantText: "Oct 27 10:00:00",
		},
		{
			name:     "No Timestamp",
			line:     "Just a random log line",
			wantTS:   false,
			wantText: "",
		},
		{
			name:     "Nginx Error",
			line:     "2023/10/27 10:00:00 [error] 123#123: *1 open()",
			wantTS:   true,
			wantText: "2023/10/27 10:00:00",
		},
		{
			name:     "Nginx Access",
			line:     "127.0.0.1 - - [27/Oct/2023:10:00:00 +0000] \"GET / HTTP/1.1\"",
			wantTS:   true,
			wantText: "27/Oct/2023:10:00:00 +0000",
		},
		{
			name:     "Nginx Access IPv6",
			line:     "::1 - - [27/Oct/2023:10:00:00 +0000] \"GET / HTTP/1.1\"",
			wantTS:   true,
			wantText: "27/Oct/2023:10:00:00 +0000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, text := extractTimestamp([]byte(tt.line))
			if tt.wantTS && ts <= 0 {
				t.Errorf("extractTimestamp() timestamp = %v, want > 0", ts)
			}
			if !tt.wantTS && ts != 0 {
				t.Errorf("extractTimestamp() timestamp = %v, want 0", ts)
			}
			if text != tt.wantText {
				t.Errorf("extractTimestamp() text = %v, want %v", text, tt.wantText)
			}
		})
	}
}

func BenchmarkExtractTimestamp(b *testing.B) {
	lines := [][]byte{
		[]byte("[1234.5678] some log message"),
		[]byte("2023-10-27T10:00:00Z some log message"),
		[]byte("Oct 27 10:00:00 host process: message"),
		[]byte("Just a random log line"),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, line := range lines {
			_, _ = extractTimestamp(line)
		}
	}
}
