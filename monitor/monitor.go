package monitor

import (
	"bufio"
	"context"
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

const (
	// Max buffer size to prevent memory leaks (e.g. 1000 lines)
	MaxBufferSize = 1000
	// Scanner buffer size (1MB) to handle long log lines
	MaxScanTokenSize = 1024 * 1024
	// Flush interval
	FlushInterval = 5 * time.Second
)

type Monitor struct {
	ctx               context.Context
	Source            sources.LogSource
	Detector          detectors.Detector
	ExclusionDetector detectors.Detector
	Collector         *sysstat.Collector
	Verbose           bool
	StopOnEOF         bool

	// Buffering
	buffer           strings.Builder
	bufferCount      int
	bufferMutex      sync.Mutex
	bufferStartTime  float64
	flushTimer       *time.Timer
	lastActivityTime time.Time
}

func New(ctx context.Context, source sources.LogSource, detector detectors.Detector, collector *sysstat.Collector, verbose bool, excludePattern string) (*Monitor, error) {
	m := &Monitor{
		ctx:       ctx,
		Source:    source,
		Detector:  detector,
		Collector: collector,
		Verbose:   verbose,
	}
	if excludePattern != "" {
		ed, err := detectors.NewGenericDetector(excludePattern)
		if err != nil {
			return nil, err
		}
		m.ExclusionDetector = ed
	}
	// Initialize timer as stopped
	m.flushTimer = time.AfterFunc(FlushInterval, func() {
		m.flushBuffer()
	})
	m.flushTimer.Stop()
	return m, nil
}

func (m *Monitor) Start() {
	if m.Verbose {
		log.Printf("Starting monitor for %s", m.Source.Name())
	}

	for {
		reader, err := m.Source.Stream()
		if err != nil {
			log.Printf("Error starting source %s: %v", m.Source.Name(), err)
			time.Sleep(1 * time.Second)
			continue
		}

		scanner := bufio.NewScanner(reader)
		// Increase buffer size to handle long lines
		buf := make([]byte, 0, MaxScanTokenSize)
		scanner.Buffer(buf, MaxScanTokenSize)

		for scanner.Scan() {
			lineBytes := scanner.Bytes()
			if m.Detector.Detect(lineBytes) {
				if m.ExclusionDetector != nil && m.ExclusionDetector.Detect(lineBytes) {
					if m.Verbose {
						log.Printf("[%s] Excluded: %s", m.Source.Name(), string(lineBytes))
					}
					continue
				}
				if m.Verbose {
					log.Printf("[%s] Matched: %s", m.Source.Name(), string(lineBytes))
				}
				m.processMatch(lineBytes)
			}
		}

		// Flush any remaining buffer
		m.forceFlush()

		if err := scanner.Err(); err != nil {
			// Suppress specific errors when stopping on EOF is enabled
			if !m.StopOnEOF || !strings.Contains(err.Error(), "file already closed") {
				log.Printf("Error reading from source %s: %v", m.Source.Name(), err)
			}
		}

		if m.StopOnEOF {
			if m.Verbose {
				log.Printf("Monitor for %s stopped (StopOnEOF set).", m.Source.Name())
			}
			break
		}

		if m.Verbose {
			log.Printf("Monitor for %s stopped, restarting in 1s...", m.Source.Name())
		}
		select {
		case <-m.ctx.Done():
			return
		case <-time.After(1 * time.Second):
		}
	}
}

func (m *Monitor) processMatch(line []byte) {
	m.bufferMutex.Lock()
	m.lastActivityTime = time.Now()

	matches := timestampRegex.FindSubmatch(line)
	var timestamp float64
	if len(matches) > 1 {
		// ParseFloat requires string, but the timestamp part is short
		timestamp, _ = strconv.ParseFloat(string(matches[1]), 64)
	}

	var msgToSend string

	if m.bufferCount == 0 {
		m.buffer.Write(line)
		m.bufferCount = 1
		m.bufferStartTime = timestamp
		m.resetTimerLocked()
	} else {
		// Check max buffer size to prevent memory leaks
		if m.bufferCount >= MaxBufferSize {
			// Force flush current buffer and start new
			msgToSend = m.buffer.String()
			m.buffer.Reset()
			m.buffer.Write(line)
			m.bufferCount = 1
			m.bufferStartTime = timestamp
			m.resetTimerLocked()
		} else {
			// Group by 5 seconds window
			if timestamp == 0 || (timestamp-m.bufferStartTime) <= 5.0 {
				m.buffer.WriteByte('\n')
				m.buffer.Write(line)
				m.bufferCount++
				m.resetTimerLocked()
			} else {
				// Flush current
				msgToSend = m.buffer.String()
				m.buffer.Reset()
				m.buffer.Write(line)
				m.bufferCount = 1
				m.bufferStartTime = timestamp
				m.resetTimerLocked()
			}
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
		m.flushTimer.Reset(FlushInterval)
	}
}

func (m *Monitor) flushBuffer() {
	m.bufferMutex.Lock()
	// Check for staleness to handle race conditions
	// If activity happened recently (less than FlushInterval), it means the timer was reset
	// but this execution is from a previous firing that wasn't stopped in time (or just concurrent scheduling).
	// We use a slightly smaller duration to allow for jitter.
	if time.Since(m.lastActivityTime) < (FlushInterval - 100*time.Millisecond) {
		m.bufferMutex.Unlock()
		return
	}

	if m.bufferCount == 0 {
		m.bufferMutex.Unlock()
		return
	}

	msg := m.buffer.String()
	m.buffer.Reset()
	m.bufferCount = 0
	m.bufferMutex.Unlock()

	m.sendToSentry(msg)
}

func (m *Monitor) forceFlush() {
	m.bufferMutex.Lock()
	if m.flushTimer != nil {
		m.flushTimer.Stop()
	}

	if m.bufferCount == 0 {
		m.bufferMutex.Unlock()
		return
	}

	msg := m.buffer.String()
	m.buffer.Reset()
	m.bufferCount = 0
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
			scope.SetContext("Server State", state.ToMap())
		}

		// We send the line as the message.
		// Sentry will group these based on the message content.
		sentry.CaptureMessage(line)
	})
}
