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
	case "nginx-error":
		return NewNginxErrorDetector(), nil
	case "json":
		if pattern == "" {
			return nil, fmt.Errorf("pattern is required for json detector (format: key:regex)")
		}
		return NewJsonDetector(pattern)
	case "custom", "":
		if pattern == "" {
			return nil, fmt.Errorf("pattern is required for custom detector")
		}
		return NewGenericDetector(pattern)
	default:
		return nil, fmt.Errorf("unknown detector format: %s", format)
	}
}

// IsKnownDetector checks if the given name matches a known detector type.
func IsKnownDetector(name string) bool {
	switch name {
	case "dmesg", "nginx", "nginx-error", "json":
		return true
	default:
		return false
	}
}
