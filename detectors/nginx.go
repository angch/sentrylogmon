package detectors

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

	if ts, tsStr, ok := ParseNginxAccess(line); ok {
		return ts, tsStr, true
	}

	return 0, "", false
}
