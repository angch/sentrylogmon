package detectors

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type JsonDetector struct {
	Field   string
	Pattern *regexp.Regexp
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
	var data map[string]interface{}
	if err := json.Unmarshal(line, &data); err != nil {
		return false
	}

	val, ok := data[d.Field]
	if !ok {
		return false
	}

	// Convert value to string for regex matching
	valStr := fmt.Sprintf("%v", val)
	return d.Pattern.MatchString(valStr)
}

func (d *JsonDetector) GetContext(line []byte) map[string]interface{} {
	var data map[string]interface{}
	if err := json.Unmarshal(line, &data); err != nil {
		return nil
	}
	return data
}
