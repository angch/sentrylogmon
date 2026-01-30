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
	"github.com/angch/sentrylogmon/metrics"
	"github.com/angch/sentrylogmon/sources"
	"github.com/angch/sentrylogmon/sysstat"
	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// [1234.5678]
	timestampRegexDmesg = regexp.MustCompile(`^\[\s*([0-9.]+)\]`)
	// 2006-01-02T15:04:05Z07:00 or 2006-01-02 15:04:05
	timestampRegexISO = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)`)
	// Oct 27 10:00:00 or <34>Oct 27 10:00:00
	timestampRegexSyslog = regexp.MustCompile(`^(?:<\d{1,3}>)?([A-Z][a-z]{2}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})`)

	syslogPriRegex = regexp.MustCompile(`^<(\d{1,3})>`)
)

func extractSyslogPriority(line []byte) (int, int, int, bool) {
	if matches := syslogPriRegex.FindSubmatch(line); len(matches) == 2 {
		priStr := string(matches[1])
		if pri, err := strconv.Atoi(priStr); err == nil {
			facility := pri / 8
			severity := pri % 8
			return pri, facility, severity, true
		}
	}
	return 0, 0, 0, false
}

func extractTimestamp(line []byte) (float64, string) {
	// 1. Try dmesg format first (fastest/most common for this tool initially)
	if indices := timestampRegexDmesg.FindSubmatchIndex(line); len(indices) >= 4 {
		tsStr := string(line[indices[2]:indices[3]])
		// ParseFloat requires string, but the timestamp part is short
		if ts, err := strconv.ParseFloat(tsStr, 64); err == nil {
			return ts, tsStr
		}
	}

	// 2. Try ISO8601/RFC3339
	if indices := timestampRegexISO.FindSubmatchIndex(line); len(indices) >= 4 {
		tsStr := string(line[indices[2]:indices[3]])
		// Try parsing with common layouts
		layouts := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
		}
		for _, layout := range layouts {
			if t, err := time.Parse(layout, tsStr); err == nil {
				return float64(t.Unix()) + float64(t.Nanosecond())/1e9, tsStr
			}
		}
	}

	// 3. Try Syslog (Oct 27 10:00:00)
	if indices := timestampRegexSyslog.FindSubmatchIndex(line); len(indices) >= 4 {
		tsStr := string(line[indices[2]:indices[3]])
		// Syslog usually doesn't have year. We assume current year.
		if t, err := time.Parse(time.Stamp, tsStr); err == nil {
			// time.Parse(time.Stamp) returns year 0. Add current year.
			now := time.Now()
			t = t.AddDate(now.Year(), 0, 0)
			// Simple heuristic for year boundary: if result is more than 30 days in future, assume previous year
			if t.Sub(now) > 30*24*time.Hour {
				t = t.AddDate(-1, 0, 0)
			}
			return float64(t.Unix()) + float64(t.Nanosecond())/1e9, tsStr
		}
	}

	return 0, ""
}

const (
	// Max buffer size to prevent memory leaks (e.g. 1000 lines)
	MaxBufferSize = 1000
	// Scanner buffer size (1MB) to handle long log lines
	MaxScanTokenSize = 1024 * 1024
	// Flush interval
	FlushInterval = 5 * time.Second
)

type RateLimiter struct {
	limit       int
	window      time.Duration
	count       int
	windowStart time.Time
	mu          sync.Mutex
}

func (r *RateLimiter) Allow() bool {
	if r.limit <= 0 {
		return true
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	if now.Sub(r.windowStart) > r.window {
		r.windowStart = now
		r.count = 0
	}
	if r.count < r.limit {
		r.count++
		return true
	}
	return false
}

type Monitor struct {
	ctx               context.Context
	Source            sources.LogSource
	Detector          detectors.Detector
	ExclusionDetector detectors.Detector
	Collector         *sysstat.Collector
	Verbose           bool
	StopOnEOF         bool
	RateLimiter       *RateLimiter

	// Buffering
	buffer           strings.Builder
	bufferCount      int
	bufferMutex      sync.Mutex
	bufferStartTime  float64
	flushTimer       *time.Timer
	lastActivityTime time.Time
}

type Options struct {
	Verbose         bool
	ExcludePattern  string
	RateLimitBurst  int
	RateLimitWindow string
}

func New(ctx context.Context, source sources.LogSource, detector detectors.Detector, collector *sysstat.Collector, opts Options) (*Monitor, error) {
	m := &Monitor{
		ctx:       ctx,
		Source:    source,
		Detector:  detector,
		Collector: collector,
		Verbose:   opts.Verbose,
	}
	if opts.ExcludePattern != "" {
		ed, err := detectors.NewGenericDetector(opts.ExcludePattern)
		if err != nil {
			return nil, err
		}
		m.ExclusionDetector = ed
	}

	// Initialize RateLimiter
	if opts.RateLimitBurst > 0 {
		window := 0 * time.Second
		if opts.RateLimitWindow != "" {
			d, err := time.ParseDuration(opts.RateLimitWindow)
			if err == nil {
				window = d
			} else {
				log.Printf("Invalid rate limit window '%s', defaulting to 0: %v", opts.RateLimitWindow, err)
			}
		}
		m.RateLimiter = &RateLimiter{
			limit:       opts.RateLimitBurst,
			window:      window,
			windowStart: time.Now(),
		}
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
			metrics.ProcessedLinesTotal.With(prometheus.Labels{"source": m.Source.Name()}).Inc()
			lineBytes := scanner.Bytes()
			if m.Detector.Detect(lineBytes) {
				if m.ExclusionDetector != nil && m.ExclusionDetector.Detect(lineBytes) {
					if m.Verbose {
						log.Printf("[%s] Excluded: %s", m.Source.Name(), string(lineBytes))
					}
					continue
				}
				metrics.IssuesDetectedTotal.With(prometheus.Labels{"source": m.Source.Name()}).Inc()
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

	timestamp, _ := extractTimestamp(line)

	if transformer, ok := m.Detector.(detectors.MessageTransformer); ok {
		line = transformer.TransformMessage(line)
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
	if m.RateLimiter != nil && !m.RateLimiter.Allow() {
		metrics.SentryEventsTotal.With(prometheus.Labels{"source": m.Source.Name(), "status": "dropped"}).Inc()
		if m.Verbose {
			log.Printf("[%s] Rate limited, dropping event.", m.Source.Name())
		}
		return
	}

	metrics.SentryEventsTotal.With(prometheus.Labels{"source": m.Source.Name(), "status": "sent"}).Inc()
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("source", m.Source.Name())

		// Try to extract timestamp for metadata from the first line
		_, tsStr := extractTimestamp([]byte(line))
		if tsStr != "" {
			scope.SetTag("log_timestamp", tsStr)
		}

		// Try to extract syslog priority
		if pri, facility, severity, ok := extractSyslogPriority([]byte(line)); ok {
			scope.SetTag("syslog_priority", strconv.Itoa(pri))
			scope.SetTag("syslog_facility", strconv.Itoa(facility))
			scope.SetTag("syslog_severity", strconv.Itoa(severity))
		}

		scope.SetExtra("raw_line", line)

		if m.Collector != nil {
			state := m.Collector.GetState()
			// Use ToMap() to directly convert struct to map, avoiding double JSON marshaling
			scope.SetContext("Server State", state.ToMap())
		}

		if extractor, ok := m.Detector.(detectors.ContextExtractor); ok {
			// Extract context from the first line
			firstLine := line
			if idx := strings.IndexByte(line, '\n'); idx != -1 {
				firstLine = line[:idx]
			}
			if ctx := extractor.GetContext([]byte(firstLine)); ctx != nil {
				scope.SetContext("Log Data", ctx)
			}
		}

		// We send the line as the message.
		// Sentry will group these based on the message content.
		sentry.CaptureMessage(line)
	})
}
