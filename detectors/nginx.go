package detectors

import "time"

// NginxDetector detects issues in Nginx error logs.
// Default pattern: (?i)(error|critical|alert|emerg)
// Note: "warn" is often just noise, but can be added if needed.
type NginxDetector struct {
	*GenericDetector
}

func NewNginxDetector() *NginxDetector {
	d, _ := NewGenericDetector("(?i)(error|critical|crit|alert|emerg)")
	return &NginxDetector{GenericDetector: d}
}

func (d *NginxDetector) ExtractTimestamp(line []byte) (float64, string, bool) {
	if ts, tsStr, ok := ParseNginxError(line); ok {
		return ts, tsStr, true
	}

	if indices := TimestampRegexNginxAccess.FindSubmatchIndex(line); len(indices) >= 4 {
		tsStr := string(line[indices[2]:indices[3]])
		if t, err := time.Parse("02/Jan/2006:15:04:05 -0700", tsStr); err == nil {
			return float64(t.Unix()) + float64(t.Nanosecond())/1e9, tsStr, true
		}
	}

	return 0, "", false
}
