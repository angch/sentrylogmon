package detectors

// Detector is the interface for detecting issues in log lines.
type Detector interface {
	// Detect returns true if the line contains an issue.
	Detect(line string) bool
}
