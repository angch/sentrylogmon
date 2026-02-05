package detectors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

type JsonDetector struct {
	Field    string
	Pattern  *regexp.Regexp

	mu       sync.Mutex
	lastData map[string]interface{}
	lastLine []byte
}

func NewJsonDetector(pattern string) (*JsonDetector, error) {
	parts := strings.SplitN(pattern, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid json pattern format: expected 'key:regex', got '%s'", pattern)
	}
	field := strings.TrimSpace(parts[0])
	regexStr := strings.TrimSpace(parts[1])

	re, err := regexp.Compile(regexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid regex for json detector: %v", err)
	}

	return &JsonDetector{
		Field:   field,
		Pattern: re,
	}, nil
}

func (d *JsonDetector) Detect(line []byte) bool {
	// We do not lock initially because Unmarshal is heavy and we don't want to block readers if possible.
	// However, usually Detect is called before readers.

	var data map[string]interface{}
	if err := json.Unmarshal(line, &data); err != nil {
		d.mu.Lock()
		d.lastData = nil
		d.lastLine = nil
		d.mu.Unlock()
		return false
	}

	val, ok := data[d.Field]
	if !ok {
		d.mu.Lock()
		d.lastData = nil
		d.lastLine = nil
		d.mu.Unlock()
		return false
	}

	// Convert value to string for regex matching
	valStr := fmt.Sprintf("%v", val)
	if d.Pattern.MatchString(valStr) {
		d.mu.Lock()
		d.lastData = data
		// Clone line
		d.lastLine = make([]byte, len(line))
		copy(d.lastLine, line)
		d.mu.Unlock()
		return true
	}

	d.mu.Lock()
	d.lastData = nil
	d.lastLine = nil
	d.mu.Unlock()
	return false
}

func (d *JsonDetector) GetContext(line []byte) map[string]interface{} {
	d.mu.Lock()
	// Verify cache validity by checking content equality
	if d.lastData != nil && bytes.Equal(d.lastLine, line) {
		data := d.lastData
		d.mu.Unlock()
		return data
	}
	d.mu.Unlock()

	var data map[string]interface{}
	if err := json.Unmarshal(line, &data); err != nil {
		return nil
	}
	return data
}

func (d *JsonDetector) ExtractTimestamp(line []byte) (float64, string, bool) {
	var data map[string]interface{}

	d.mu.Lock()
	if d.lastData != nil && bytes.Equal(d.lastLine, line) {
		data = d.lastData
	}
	d.mu.Unlock()

	if data == nil {
		if err := json.Unmarshal(line, &data); err != nil {
			return 0, "", false
		}
	}

	// Helper to check fields
	checkField := func(key string) (float64, string, bool) {
		val, ok := data[key]
		if !ok {
			return 0, "", false
		}

		switch v := val.(type) {
		case string:
			// Try parsing as ISO8601/RFC3339
			// Add more layouts if needed
			layouts := []string{
				time.RFC3339,
				time.RFC3339Nano,
				"2006-01-02 15:04:05",
				"2006-01-02T15:04:05",
				"2006-01-02T15:04:05Z07:00",
			}
			for _, layout := range layouts {
				if t, err := time.Parse(layout, v); err == nil {
					return float64(t.Unix()) + float64(t.Nanosecond())/1e9, v, true
				}
			}
		case float64:
			// Assume unix timestamp (seconds or milliseconds)
			// Heuristic: if > 1e11 (year ~5138), maybe milliseconds?
			if v > 1e11 {
				return v / 1000.0, fmt.Sprintf("%.3f", v/1000.0), true
			}
			return v, fmt.Sprintf("%.3f", v), true
		}
		return 0, "", false
	}

	fields := []string{"time", "timestamp", "ts", "date", "@timestamp"}
	for _, f := range fields {
		if ts, tsStr, ok := checkField(f); ok {
			return ts, tsStr, ok
		}
	}

	return 0, "", false
}
