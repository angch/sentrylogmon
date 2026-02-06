package monitor

import (
	"bufio"
	"context"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/angch/sentrylogmon/detectors"
	"github.com/angch/sentrylogmon/metrics"
	"github.com/angch/sentrylogmon/sources"
	"github.com/angch/sentrylogmon/sysstat"
	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus"
)

var commonTimeLayouts = []string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
}

var severityKeys = []string{"level", "severity", "log_level", "type"}

func extractSyslogPriority(line []byte) (int, int, int, bool) {
	// Fast path: must start with '<'
	if len(line) < 3 || line[0] != '<' {
		return 0, 0, 0, false
	}

	// Find closing '>'
	// PRI is 1-3 digits. So '>' can be at index 2, 3, or 4.
	// line[0] is '<'
	// line[1] is digit

	end := -1
	// Optimization: check up to index 4 (length 5: <123>)
	limit := 5
	if len(line) < limit {
		limit = len(line)
	}

	for i := 1; i < limit; i++ {
		if line[i] == '>' {
			end = i
			break
		}
	}

	if end == -1 || end == 1 { // Empty or not found
		return 0, 0, 0, false
	}

	// Parse number between 1 and end
	pri := 0
	for i := 1; i < end; i++ {
		b := line[i]
		if b < '0' || b > '9' {
			return 0, 0, 0, false
		}
		pri = pri*10 + int(b-'0')
	}

	facility := pri / 8
	severity := pri % 8
	return pri, facility, severity, true
}

func extractTimestamp(line []byte) (float64, string) {
	if len(line) == 0 {
		return 0, ""
	}

	// 1. Try dmesg format first (fastest/most common for this tool initially)
	// Check if it starts with '['
	if line[0] == '[' {
		if ts, tsStr, ok := detectors.ParseDmesgTimestamp(line); ok {
			return ts, tsStr
		}
	}

	// 2. Try ISO8601/RFC3339 or Nginx
	// Starts with digit
	if line[0] >= '0' && line[0] <= '9' {
		if ts, tsStr, ok := detectors.ParseISO8601(line); ok {
			return ts, tsStr
		}

		if ts, tsStr, ok := detectors.ParseNginxError(line); ok {
			return ts, tsStr
		}

		if indices := detectors.TimestampRegexISO.FindSubmatchIndex(line); len(indices) >= 4 {
			tsStr := string(line[indices[2]:indices[3]])
			// Try parsing with common layouts
			for _, layout := range commonTimeLayouts {
				if t, err := time.Parse(layout, tsStr); err == nil {
					return float64(t.Unix()) + float64(t.Nanosecond())/1e9, tsStr
				}
			}
		}

		// Try Nginx Error (2023/10/27 10:00:00)
		if indices := detectors.TimestampRegexNginxError.FindSubmatchIndex(line); len(indices) >= 4 {
			tsStr := string(line[indices[2]:indices[3]])
			if t, err := time.Parse("2006/01/02 15:04:05", tsStr); err == nil {
				return float64(t.Unix()) + float64(t.Nanosecond())/1e9, tsStr
			}
		}
	}

	// 3. Try Syslog (Oct 27 10:00:00)
	// Starts with '<' or uppercase letter
	if line[0] == '<' || (line[0] >= 'A' && line[0] <= 'Z') {
		if ts, tsStr, ok := detectors.ParseSyslogTimestamp(line); ok {
			return ts, tsStr
		}
	}

	// 4. Try Nginx Access ([27/Oct/2023:10:00:00 +0000])
	// This regex is unanchored, so it can find the timestamp anywhere in the line.
	// This handles IPv6 access logs starting with '[' or other custom formats.
	if indices := detectors.TimestampRegexNginxAccess.FindSubmatchIndex(line); len(indices) >= 4 {
		tsStr := string(line[indices[2]:indices[3]])
		if t, err := time.Parse("02/Jan/2006:15:04:05 -0700", tsStr); err == nil {
			return float64(t.Unix()) + float64(t.Nanosecond())/1e9, tsStr
		}
	}

	return 0, ""
}

const (
	// Max buffer size to prevent memory leaks (e.g. 1000 lines)
	MaxBufferSize = 1000
	// Max buffer bytes to prevent memory exhaustion (256KB)
	MaxBufferBytes = 256 * 1024
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

type SyslogPriority struct {
	Pri      int
	Facility int
	Severity int
}

type BatchMetadata struct {
	TimestampStr string
	SyslogPri    *SyslogPriority
	Context      map[string]interface{}
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
	Hub               *sentry.Hub

	// Cached metrics
	metricProcessedLines prometheus.Counter
	metricIssuesDetected prometheus.Counter
	metricSentrySent     prometheus.Counter
	metricSentryDropped  prometheus.Counter
	metricLastActivity   prometheus.Gauge

	// Buffering
	buffer           strings.Builder
	bufferCount      int
	bufferMutex      sync.Mutex
	bufferStartTime  float64
	currentBatchMeta BatchMetadata
	flushTimer       *time.Timer
	lastActivityTime time.Time

	// Inactivity detection
	maxInactivity     time.Duration
	lastReadTime      int64 // atomic unix nano
	inactivityAlerted int32 // atomic boolean
}

type Options struct {
	Verbose           bool
	ExcludePattern    string
	MaxInactivity     string
	RateLimitBurst    int
	RateLimitWindow   string
	SentryDSN         string
	SentryEnvironment string
	SentryRelease     string
}

func New(ctx context.Context, source sources.LogSource, detector detectors.Detector, collector *sysstat.Collector, opts Options) (*Monitor, error) {
	m := &Monitor{
		ctx:       ctx,
		Source:    source,
		Detector:  detector,
		Collector: collector,
		Verbose:   opts.Verbose,
	}

	// Initialize cached metrics
	m.metricProcessedLines = metrics.ProcessedLinesTotal.With(prometheus.Labels{"source": source.Name()})
	m.metricIssuesDetected = metrics.IssuesDetectedTotal.With(prometheus.Labels{"source": source.Name()})
	m.metricSentrySent = metrics.SentryEventsTotal.With(prometheus.Labels{"source": source.Name(), "status": "sent"})
	m.metricSentryDropped = metrics.SentryEventsTotal.With(prometheus.Labels{"source": source.Name(), "status": "dropped"})
	m.metricLastActivity = metrics.LastActivityTimestamp.With(prometheus.Labels{"source": source.Name()})

	// Initialize Sentry Hub
	if opts.SentryDSN != "" {
		client, err := sentry.NewClient(sentry.ClientOptions{
			Dsn:         opts.SentryDSN,
			Environment: opts.SentryEnvironment,
			Release:     opts.SentryRelease,
		})
		if err != nil {
			return nil, err
		}
		m.Hub = sentry.NewHub(client, sentry.NewScope())
	} else {
		m.Hub = sentry.CurrentHub()
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
		} else {
			// Default to 1s if unspecified
			window = 1 * time.Second
			if opts.Verbose {
				log.Printf("Rate limit window not specified, defaulting to 1s")
			}
		}
		m.RateLimiter = &RateLimiter{
			limit:       opts.RateLimitBurst,
			window:      window,
			windowStart: time.Now(),
		}
	}

	// Initialize MaxInactivity
	if opts.MaxInactivity != "" {
		d, err := time.ParseDuration(opts.MaxInactivity)
		if err == nil {
			m.maxInactivity = d
		} else {
			log.Printf("Invalid max inactivity duration '%s': %v", opts.MaxInactivity, err)
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

	atomic.StoreInt64(&m.lastReadTime, time.Now().UnixNano())

	if m.maxInactivity > 0 {
		go m.watchdog()
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

		var lastMetricUpdateTime time.Time
		for scanner.Scan() {
			m.metricProcessedLines.Inc()

			now := time.Now()
			// Update lastReadTime for inactivity detection
			atomic.StoreInt64(&m.lastReadTime, now.UnixNano())

			if now.Sub(lastMetricUpdateTime) > 1*time.Second {
				m.metricLastActivity.Set(float64(now.Unix()))
				lastMetricUpdateTime = now
			}

			lineBytes := scanner.Bytes()
			if m.Detector.Detect(lineBytes) {
				if m.ExclusionDetector != nil && m.ExclusionDetector.Detect(lineBytes) {
					if m.Verbose {
						log.Printf("[%s] Excluded: %s", m.Source.Name(), string(lineBytes))
					}
					continue
				}
				m.metricIssuesDetected.Inc()
				if m.Verbose {
					log.Printf("[%s] Matched: %s", m.Source.Name(), string(lineBytes))
				}
				m.processMatch(lineBytes)
			}
		}

		// Flush any remaining buffer
		m.forceFlush()
		m.metricLastActivity.Set(float64(time.Now().Unix()))

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

func (m *Monitor) watchdog() {
	// Check at half the inactivity duration or at least every 100ms
	interval := m.maxInactivity / 2
	if interval < 100*time.Millisecond {
		interval = 100 * time.Millisecond
	}
	if interval > 10*time.Second {
		interval = 10 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			lastRead := time.Unix(0, atomic.LoadInt64(&m.lastReadTime))
			silenceDuration := time.Since(lastRead)

			if silenceDuration > m.maxInactivity {
				if atomic.CompareAndSwapInt32(&m.inactivityAlerted, 0, 1) {
					if m.Verbose {
						log.Printf("[%s] Inactivity detected: %v > %v", m.Source.Name(), silenceDuration, m.maxInactivity)
					}
					m.Hub.WithScope(func(scope *sentry.Scope) {
						scope.SetTag("source", m.Source.Name())
						scope.SetTag("alert_type", "inactivity")
						scope.SetLevel(sentry.LevelWarning)
						m.Hub.CaptureMessage(m.Source.Name() + ": Monitor source inactivity detected (silence for " + silenceDuration.String() + ")")
					})
				}
			} else {
				if atomic.CompareAndSwapInt32(&m.inactivityAlerted, 1, 0) {
					if m.Verbose {
						log.Printf("[%s] Activity resumed.", m.Source.Name())
					}
					m.Hub.WithScope(func(scope *sentry.Scope) {
						scope.SetTag("source", m.Source.Name())
						scope.SetTag("alert_type", "inactivity")
						scope.SetLevel(sentry.LevelInfo)
						m.Hub.CaptureMessage(m.Source.Name() + ": Monitor source activity resumed")
					})
				}
			}
		}
	}
}

func (m *Monitor) extractMetadata(line []byte, tsStr string) BatchMetadata {
	meta := BatchMetadata{
		TimestampStr: tsStr,
	}

	if pri, facility, severity, ok := extractSyslogPriority(line); ok {
		meta.SyslogPri = &SyslogPriority{
			Pri:      pri,
			Facility: facility,
			Severity: severity,
		}
	}

	if extractor, ok := m.Detector.(detectors.ContextExtractor); ok {
		if ctx := extractor.GetContext(line); ctx != nil {
			meta.Context = ctx
		}
	}

	return meta
}

func (m *Monitor) processMatch(line []byte) {
	m.bufferMutex.Lock()
	m.lastActivityTime = time.Now()

	var timestamp float64
	var tsStr string
	var ok bool

	if extractor, isExtractor := m.Detector.(detectors.TimestampExtractor); isExtractor {
		timestamp, tsStr, ok = extractor.ExtractTimestamp(line)
	}

	if !ok {
		timestamp, tsStr = extractTimestamp(line)
	}

	if transformer, ok := m.Detector.(detectors.MessageTransformer); ok {
		line = transformer.TransformMessage(line)
	}

	var msgToSend string
	var metaToSend BatchMetadata

	if m.bufferCount == 0 {
		m.buffer.Write(line)
		m.bufferCount = 1
		m.bufferStartTime = timestamp
		m.currentBatchMeta = m.extractMetadata(line, tsStr)
		m.resetTimerLocked()
	} else {
		// Check max buffer size to prevent memory leaks
		if m.bufferCount >= MaxBufferSize || (m.buffer.Len()+len(line)) >= MaxBufferBytes {
			// Force flush current buffer and start new
			msgToSend = m.buffer.String()
			metaToSend = m.currentBatchMeta

			m.buffer.Reset()
			m.buffer.Write(line)
			m.bufferCount = 1
			m.bufferStartTime = timestamp
			m.currentBatchMeta = m.extractMetadata(line, tsStr)
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
				metaToSend = m.currentBatchMeta

				m.buffer.Reset()
				m.buffer.Write(line)
				m.bufferCount = 1
				m.bufferStartTime = timestamp
				m.currentBatchMeta = m.extractMetadata(line, tsStr)
				m.resetTimerLocked()
			}
		}
	}
	m.bufferMutex.Unlock()

	if msgToSend != "" {
		m.sendToSentry(msgToSend, metaToSend)
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
	meta := m.currentBatchMeta
	m.buffer.Reset()
	m.bufferCount = 0
	m.currentBatchMeta = BatchMetadata{}
	m.bufferMutex.Unlock()

	m.sendToSentry(msg, meta)
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
	meta := m.currentBatchMeta
	m.buffer.Reset()
	m.bufferCount = 0
	m.currentBatchMeta = BatchMetadata{}
	m.bufferMutex.Unlock()

	m.sendToSentry(msg, meta)
}

func (m *Monitor) sendToSentry(line string, meta BatchMetadata) {
	if m.RateLimiter != nil && !m.RateLimiter.Allow() {
		m.metricSentryDropped.Inc()
		if m.Verbose {
			log.Printf("[%s] Rate limited, dropping event.", m.Source.Name())
		}
		return
	}

	m.metricSentrySent.Inc()

	m.Hub.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("source", m.Source.Name())

		if meta.TimestampStr != "" {
			scope.SetTag("log_timestamp", meta.TimestampStr)
		}

		if meta.SyslogPri != nil {
			scope.SetTag("syslog_priority", strconv.Itoa(meta.SyslogPri.Pri))
			scope.SetTag("syslog_facility", strconv.Itoa(meta.SyslogPri.Facility))
			scope.SetTag("syslog_severity", strconv.Itoa(meta.SyslogPri.Severity))

			// Map severity to Sentry Level
			var level sentry.Level
			switch meta.SyslogPri.Severity {
			case 0, 1, 2: // Emergency, Alert, Critical
				level = sentry.LevelFatal
			case 3: // Error
				level = sentry.LevelError
			case 4: // Warning
				level = sentry.LevelWarning
			case 5, 6: // Notice, Informational
				level = sentry.LevelInfo
			case 7: // Debug
				level = sentry.LevelDebug
			default:
				level = sentry.LevelInfo
			}
			scope.SetLevel(level)
		}

		scope.SetExtra("raw_line", line)

		if m.Collector != nil {
			state := m.Collector.GetState()
			// Use ToMap() to directly convert struct to map, avoiding double JSON marshaling
			scope.SetContext("Server State", state.ToMap())
		}

		if meta.Context != nil {
			scope.SetContext("Log Data", meta.Context)

			// Try to extract level/severity from context
			var levelStr string

			for _, key := range severityKeys {
				if val, ok := meta.Context[key]; ok {
					if s, ok := val.(string); ok {
						levelStr = strings.ToLower(s)
						break
					}
				}
			}

			if levelStr != "" {
				var level sentry.Level
				switch levelStr {
				case "fatal", "critical", "alert", "emergency", "panic":
					level = sentry.LevelFatal
				case "error", "err":
					level = sentry.LevelError
				case "warning", "warn":
					level = sentry.LevelWarning
				case "info", "information":
					level = sentry.LevelInfo
				case "debug", "trace":
					level = sentry.LevelDebug
				}

				if level != "" {
					scope.SetLevel(level)
				}
			}
		}

		// We send the line as the message.
		// Sentry will group these based on the message content.
		m.Hub.CaptureMessage(line)
	})
}
