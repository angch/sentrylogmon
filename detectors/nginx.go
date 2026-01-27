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
