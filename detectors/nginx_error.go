package detectors

// NginxErrorDetector detects issues in Nginx error logs.
// It uses the same pattern as NginxDetector but is registered as a separate format "nginx-error".
// Pattern: (?i)(error|critical|crit|alert|emerg)
type NginxErrorDetector struct {
	*GenericDetector
}

func NewNginxErrorDetector() *NginxErrorDetector {
	d, _ := NewGenericDetector("(?i)(error|critical|crit|alert|emerg)")
	return &NginxErrorDetector{GenericDetector: d}
}
