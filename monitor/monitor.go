package monitor

import (
	"bufio"
	"log"
	"regexp"

	"github.com/angch/sentrylogmon/detectors"
	"github.com/angch/sentrylogmon/sources"
	"github.com/getsentry/sentry-go"
)

var timestampRegex = regexp.MustCompile(`^\[\s*([0-9.]+)\]`)

type Monitor struct {
	Source   sources.LogSource
	Detector detectors.Detector
	Verbose  bool
}

func New(source sources.LogSource, detector detectors.Detector, verbose bool) (*Monitor, error) {
	return &Monitor{
		Source:   source,
		Detector: detector,
		Verbose:  verbose,
	}, nil
}

func (m *Monitor) Start() {
	if m.Verbose {
		log.Printf("Starting monitor for %s", m.Source.Name())
	}

	reader, err := m.Source.Stream()
	if err != nil {
		log.Printf("Error starting source %s: %v", m.Source.Name(), err)
		return
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if m.Detector.Detect(line) {
			if m.Verbose {
				log.Printf("[%s] Matched: %s", m.Source.Name(), line)
			}
			m.sendToSentry(line)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from source %s: %v", m.Source.Name(), err)
	}

	if m.Verbose {
		log.Printf("Monitor for %s stopped", m.Source.Name())
	}
}

func (m *Monitor) sendToSentry(line string) {
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("source", m.Source.Name())

		// Try to extract timestamp for metadata
		matches := timestampRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			scope.SetTag("log_timestamp", matches[1])
		}

		scope.SetExtra("raw_line", line)

		// We send the line as the message.
		// Sentry will group these based on the message content.
		sentry.CaptureMessage(line)
	})
}
