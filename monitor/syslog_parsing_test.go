package monitor

import (
	"testing"
)

func TestExtractTimestamp_SyslogWithPRI(t *testing.T) {
	// RFC 3164 format with PRI
	// <34>Oct 11 22:14:15 mymachine su: 'su root' failed
	// PRI 34 = (auth(4) * 8) + crit(2) = 32 + 2 = 34

	line := []byte("<34>Oct 11 22:14:15 mymachine su: 'su root' failed")

	ts, tsStr := extractTimestamp(line)

	if ts == 0 {
		t.Errorf("Failed to extract timestamp from syslog message with PRI: %s", string(line))
	} else {
		t.Logf("Successfully extracted timestamp: %f (%s)", ts, tsStr)
	}

	// Compare with without PRI
	lineNoPri := []byte("Oct 11 22:14:15 mymachine su: 'su root' failed")
	ts2, tsStr2 := extractTimestamp(lineNoPri)
	if ts2 == 0 {
		t.Errorf("Failed to extract timestamp from syslog message without PRI: %s", string(lineNoPri))
	} else {
		t.Logf("Successfully extracted timestamp (no PRI): %f (%s)", ts2, tsStr2)
	}
}

func TestExtractSyslogPriority(t *testing.T) {
	tests := []struct {
		line    string
		wantPri int
		wantFac int
		wantSev int
		wantOk  bool
	}{
		{"<34>Oct 11 22:14:15 mymachine su: 'su root' failed", 34, 4, 2, true},
		{"<0>Emergency", 0, 0, 0, true},
		{"<191>Debug", 191, 23, 7, true}, // Max usually 191 (23*8 + 7)
		{"No PRI here", 0, 0, 0, false},
		{"<abc>Invalid", 0, 0, 0, false},
		{"<1234>Too long", 0, 0, 0, false}, // Regex expects 1-3 digits then >
		{"<1>Short", 1, 0, 1, true},
	}

	for _, tt := range tests {
		pri, fac, sev, ok := extractSyslogPriority([]byte(tt.line))
		if ok != tt.wantOk {
			t.Errorf("extractSyslogPriority(%q) ok = %v, want %v", tt.line, ok, tt.wantOk)
			continue
		}
		if ok {
			if pri != tt.wantPri {
				t.Errorf("extractSyslogPriority(%q) pri = %v, want %v", tt.line, pri, tt.wantPri)
			}
			if fac != tt.wantFac {
				t.Errorf("extractSyslogPriority(%q) fac = %v, want %v", tt.line, fac, tt.wantFac)
			}
			if sev != tt.wantSev {
				t.Errorf("extractSyslogPriority(%q) sev = %v, want %v", tt.line, sev, tt.wantSev)
			}
		}
	}
}
