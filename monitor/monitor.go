package monitor

import (
	"bufio"
	"encoding/json"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/angch/sentrylogmon/detectors"
	"github.com/angch/sentrylogmon/sources"
	"github.com/angch/sentrylogmon/sysstat"
	"github.com/getsentry/sentry-go"
)

var timestampRegex = regexp.MustCompile(`^\[\s*([0-9.]+)\]`)

type Monitor struct {
	Source    sources.LogSource
	Detector  detectors.Detector
	Collector *sysstat.Collector
	Verbose   bool

	// Buffering
	buffer           []string
	bufferMutex      sync.Mutex
	bufferStartTime  float64
	flushTimer       *time.Timer
	lastActivityTime time.Time
}

func New(source sources.LogSource, detector detectors.Detector, collector *sysstat.Collector, verbose bool) (*Monitor, error) {
	return &Monitor{
		Source:    source,
		Detector:  detector,
		Collector: collector,
		Verbose:   verbose,
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
			m.processMatch(line)
		}
	}

	// Flush any remaining buffer
	m.forceFlush()

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from source %s: %v", m.Source.Name(), err)
	}

	if m.Verbose {
		log.Printf("Monitor for %s stopped", m.Source.Name())
	}
}

func (m *Monitor) processMatch(line string) {
	m.bufferMutex.Lock()
	m.lastActivityTime = time.Now()

	matches := timestampRegex.FindStringSubmatch(line)
	var timestamp float64
	if len(matches) > 1 {
		timestamp, _ = strconv.ParseFloat(matches[1], 64)
	}

	var msgToSend string

	if len(m.buffer) == 0 {
		m.buffer = []string{line}
		m.bufferStartTime = timestamp
		m.resetTimerLocked()
	} else {
		// Group by 5 seconds window
		if timestamp == 0 || (timestamp-m.bufferStartTime) <= 5.0 {
			m.buffer = append(m.buffer, line)
			m.resetTimerLocked()
		} else {
			// Flush current
			msgToSend = strings.Join(m.buffer, "\n")
			// Start new
			m.buffer = []string{line}
			m.bufferStartTime = timestamp
			m.resetTimerLocked()
		}
	}
	m.bufferMutex.Unlock()

	if msgToSend != "" {
		m.sendToSentry(msgToSend)
	}
}

func (m *Monitor) resetTimerLocked() {
	if m.flushTimer != nil {
		m.flushTimer.Stop()
	}
	m.flushTimer = time.AfterFunc(5*time.Second, func() {
		m.flushBuffer()
	})
}

func (m *Monitor) flushBuffer() {
	m.bufferMutex.Lock()
	// Check for staleness to handle race conditions
	// We use a slightly smaller duration than 5s to allow for scheduling jitter,
	// but generally if it's very recent, it means it was just updated.
	if time.Since(m.lastActivityTime) < 4500*time.Millisecond {
		m.bufferMutex.Unlock()
		return
	}

	if len(m.buffer) == 0 {
		m.bufferMutex.Unlock()
		return
	}

	if m.flushTimer != nil {
		m.flushTimer.Stop()
	}

	msg := strings.Join(m.buffer, "\n")
	m.buffer = nil
	m.bufferMutex.Unlock()

	m.sendToSentry(msg)
}

func (m *Monitor) forceFlush() {
	m.bufferMutex.Lock()
	if m.flushTimer != nil {
		m.flushTimer.Stop()
	}

	if len(m.buffer) == 0 {
		m.bufferMutex.Unlock()
		return
	}

	msg := strings.Join(m.buffer, "\n")
	m.buffer = nil
	m.bufferMutex.Unlock()

	m.sendToSentry(msg)
}

func (m *Monitor) sendToSentry(line string) {
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("source", m.Source.Name())

		// Try to extract timestamp for metadata from the first line
		matches := timestampRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			scope.SetTag("log_timestamp", matches[1])
		}

		scope.SetExtra("raw_line", line)

		if m.Collector != nil {
			state := m.Collector.GetState()
			// Convert state to map[string]interface{} for SetContext
			var stateMap map[string]interface{}
			data, _ := json.Marshal(state)
			json.Unmarshal(data, &stateMap)
			scope.SetContext("Server State", stateMap)
		}

		// We send the line as the message.
		// Sentry will group these based on the message content.
		sentry.CaptureMessage(line)
	})
}
