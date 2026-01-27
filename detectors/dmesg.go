package detectors

import (
	"regexp"
	"strconv"
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
)

func NewDmesgDetector() *DmesgDetector {
	// Added "exception" to the pattern
	d, _ := NewGenericDetector("(?i)(error|fail|panic|oops|exception)")
	return &DmesgDetector{GenericDetector: d}
}

func (d *DmesgDetector) Detect(line string) bool {
	// 1. Check if it matches the error pattern first
	isError := d.GenericDetector.Detect(line)

	// 2. Parse the line
	matches := dmesgLineRegex.FindStringSubmatch(line)
	var timestamp float64
	var header string
	var err error

	if len(matches) >= 3 {
		timestamp, err = strconv.ParseFloat(matches[1], 64)
		if err != nil {
			// Should not happen given regex
			timestamp = 0
		}
		header = matches[2]
	}

	if isError {
		// Update state
		if len(matches) >= 3 {
			d.lastMatchTime = timestamp
			d.lastMatchHeader = header
		}
		return true
	}

	// 3. If not an explicit error, check if it's related context
	if d.lastMatchHeader != "" {
		// Check timestamp window (5 seconds)
		if timestamp > 0 && (timestamp-d.lastMatchTime) <= 5.0 {
			// Check if headers are related
			if areHeadersRelated(d.lastMatchHeader, header) {
				// Update time, so we extend the window?
				// User says "blocks of 5 seconds after the first message".
				// So maybe we shouldn't update lastMatchTime if we want a fixed window.
				// But we definitely return true.
				// However, if we don't update lastMatchTime, we might miss errors that happen 6 seconds later but are part of the same long sequence?
				// But "blocks of 5 seconds" suggests fixed.
				// Let's stick to the prompt: "delayed until the same subheader or related errors are output and gathered to be sent as one single sentry message"
				// And "we group errors by blocks of 5 seconds after the first message".
				// I will keep lastMatchTime as the *start* of the error sequence.
				// Wait, if I don't update lastMatchTime, then `Detect` relies on the FIRST error's time.
				// That seems correct for "blocks of 5 seconds after the first message".
				// But what if we have:
				// T=0 Error
				// T=1 Info (related)
				// T=4 Info (related)
				// T=6 Info (related) -> Should this be captured?
				// If we strictly follow "5 seconds after first message", then T=6 is dropped (or starts new group if it was an error, but it's just info).
				// If it's just info, and we drop it, we might lose context.
				// But strict 5s rule implies we stop collecting.
				// I'll assume we capture if <= 5s.

				// However, if we see another *Error* at T=4, does it restart the timer?
				// My code above says: `if isError { d.lastMatchTime = timestamp ... }`.
				// So yes, a new error restarts the timer and context.
				// This seems reasonable. If a new error happens, we are interested in its context.
				return true
			}
		}
	}

	return false
}

func areHeadersRelated(h1, h2 string) bool {
	h1 = strings.TrimSpace(h1)
	h2 = strings.TrimSpace(h2)
	if h1 == "" || h2 == "" {
		return false
	}
	// Check for strict equality
	if h1 == h2 {
		return true
	}
	// Check for prefix match (e.g. ata1 vs ata1.00)
	// We want to match if one contains the other as a prefix?
	// ata1 is prefix of ata1.00 -> True
	// ata1.00 is NOT prefix of ata1.
	// So checking both ways.
	if strings.HasPrefix(h1, h2) || strings.HasPrefix(h2, h1) {
		return true
	}
	return false
}
