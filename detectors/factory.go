package detectors

import "fmt"

// GetDetector returns a detector based on the format name.
// If format is "custom" or empty, it requires a pattern and returns a GenericDetector.
func GetDetector(format string, pattern string) (Detector, error) {
	switch format {
	case "dmesg":
		return NewDmesgDetector(), nil
	case "nginx":
		return NewNginxDetector(), nil
	case "custom", "":
		if pattern == "" {
			return nil, fmt.Errorf("pattern is required for custom detector")
		}
		return NewGenericDetector(pattern)
	default:
		return nil, fmt.Errorf("unknown detector format: %s", format)
	}
}
