package detectors

import (
	"bytes"
	"regexp"
)

// GenericDetector uses a regex pattern to detect issues.
type GenericDetector struct {
	pattern   *regexp.Regexp
	literal   []byte
	isLiteral bool
}

func NewGenericDetector(pattern string) (*GenericDetector, error) {
	if pattern == regexp.QuoteMeta(pattern) {
		return &GenericDetector{
			literal:   []byte(pattern),
			isLiteral: true,
		}, nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &GenericDetector{pattern: re}, nil
}

func (d *GenericDetector) Detect(line []byte) bool {
	if d.isLiteral {
		return bytes.Contains(line, d.literal)
	}
	return d.pattern.Match(line)
}
