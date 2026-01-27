package detectors

import (
	"regexp"
	"strings"
)

// GenericDetector uses a regex pattern to detect issues.
type GenericDetector struct {
	pattern   *regexp.Regexp
	literal   string
	isLiteral bool
}

func NewGenericDetector(pattern string) (*GenericDetector, error) {
	if pattern == regexp.QuoteMeta(pattern) {
		return &GenericDetector{
			literal:   pattern,
			isLiteral: true,
		}, nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &GenericDetector{pattern: re}, nil
}

func (d *GenericDetector) Detect(line string) bool {
	if d.isLiteral {
		return strings.Contains(line, d.literal)
	}
	return d.pattern.MatchString(line)
}
