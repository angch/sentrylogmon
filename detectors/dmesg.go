package detectors

// DmesgDetector detects issues in kernel logs.
// Default pattern: (?i)(error|fail|panic|oops)
type DmesgDetector struct {
	*GenericDetector
}

func NewDmesgDetector() *DmesgDetector {
	// We ignore error here because we know the pattern is valid.
	// In a real app we might want to handle it, but this is a preset.
	d, _ := NewGenericDetector("(?i)(error|fail|panic|oops)")
	return &DmesgDetector{GenericDetector: d}
}
