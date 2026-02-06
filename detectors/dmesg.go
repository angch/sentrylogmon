package detectors

import (
	"bytes"
	"regexp"
	"strings"
)

// DmesgDetector detects issues in kernel logs.
// Default pattern: (?i)(error|fail|panic|oops|exception)
type DmesgDetector struct {
	*GenericDetector
	// State for context tracking
	lastMatchTime   float64
	lastMatchHeader string
}

var (
	// Example: [787739.009553] ata1.00: exception Emask...
	dmesgLineRegex = regexp.MustCompile(`^\[\s*(\d+\.\d+)\]\s*([^:]+):`)
	// Example: [ 123.456] ...
	dmesgStartRegex = regexp.MustCompile(`^\[\s*\d+\.\d+\]`)
)

func NewDmesgDetector() *DmesgDetector {
	// Added "exception" to the pattern
	d, _ := NewGenericDetector("(?i)(error|fail|panic|oops|exception)")
	return &DmesgDetector{GenericDetector: d}
}

func (d *DmesgDetector) Detect(line []byte) bool {
	// 1. Check if it matches the error pattern first
	isError := d.GenericDetector.Detect(line)

	// 2. Check if it looks like a new dmesg line (starts with timestamp)
	isDmesgLine := dmesgStartRegex.Match(line)

	// 3. Parse the line for detailed info using FindSubmatchIndex to avoid allocations.
	// FindSubmatchIndex returns []int with indices instead of allocating [][]byte slices.
	// For each capture group, we get a pair of indices [start, end).
	// indices[0:2] = full match, indices[2:4] = first group (timestamp), indices[4:6] = second group (header)
	indices := dmesgLineRegex.FindSubmatchIndex(line)
	var timestamp float64
	var headerBytes []byte

	if len(indices) >= 6 {
		// Extract timestamp and header by slicing the original line bytes directly.
		// This avoids the allocation that FindSubmatch would create.
		timestampBytes := line[indices[2]:indices[3]]
		headerBytes = line[indices[4]:indices[5]]

		// Use parseFloatFromBytes to avoid string allocation.
		timestamp, _ = parseFloatFromBytes(timestampBytes)
	}

	if isError {
		// Update state
		if timestamp > 0 {
			d.lastMatchTime = timestamp
		}
		if len(headerBytes) > 0 {
			// String conversion here is necessary for storing the header for later comparison.
			d.lastMatchHeader = string(headerBytes)
		}
		return true
	}

	// 4. If not an explicit error, check if it's related context
	if d.lastMatchHeader != "" {
		if isDmesgLine {
			// It's a new log line. Check if it's related.
			if len(headerBytes) > 0 && timestamp > 0 {
				if (timestamp - d.lastMatchTime) <= 5.0 {
					// Pass headerBytes directly without string conversion
					if areHeadersRelated(d.lastMatchHeader, headerBytes) {
						return true
					}
				}
			}
			// If it's a dmesg line but not related (or couldn't parse header/timestamp),
			// we assume it's NOT context.
			return false
		} else {
			// It does not look like a dmesg line (no timestamp).
			// Assume it is a continuation line (stack trace, etc.)
			// We accept it as part of the context.
			return true
		}
	}

	return false
}

// TransformMessage strips the timestamp from the dmesg line.
func (d *DmesgDetector) TransformMessage(line []byte) []byte {
	// Check if it starts with timestamp
	if loc := dmesgStartRegex.FindIndex(line); loc != nil {
		// loc[1] is the index after the timestamp (including brackets)
		// Return the rest of the line, trimmed of whitespace
		return bytes.TrimSpace(line[loc[1]:])
	}
	return line
}

// ExtractTimestamp extracts timestamp from dmesg line.
func (d *DmesgDetector) ExtractTimestamp(line []byte) (float64, string, bool) {
	return ParseDmesgTimestamp(line)
}

func areHeadersRelated(h1 string, h2 []byte) bool {
	h1 = strings.TrimSpace(h1)
	h2 = bytes.TrimSpace(h2)
	if h1 == "" || len(h2) == 0 {
		return false
	}
	// Check for strict equality
	if equalsStringBytes(h1, h2) {
		return true
	}
	// Check for prefix match (e.g. ata1 vs ata1.00)
	if hasPrefixStringBytes(h1, h2) || hasPrefixBytesString(h2, h1) {
		return true
	}
	return false
}

func equalsStringBytes(s string, b []byte) bool {
	if len(s) != len(b) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] != b[i] {
			return false
		}
	}
	return true
}

func hasPrefixStringBytes(s string, prefix []byte) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if s[i] != prefix[i] {
			return false
		}
	}
	return true
}

func hasPrefixBytesString(b []byte, prefix string) bool {
	if len(b) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if b[i] != prefix[i] {
			return false
		}
	}
	return true
}
