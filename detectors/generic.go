package detectors

import "regexp"

// GenericDetector uses a regex pattern to detect issues.
type GenericDetector struct {
	pattern *regexp.Regexp
}

func NewGenericDetector(pattern string) (*GenericDetector, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &GenericDetector{pattern: re}, nil
}

func (d *GenericDetector) Detect(line []byte) bool {
	return d.pattern.Match(line)
}
