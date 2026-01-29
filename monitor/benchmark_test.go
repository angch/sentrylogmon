package monitor

import (
	"bufio"
	"strings"
	"testing"

	"github.com/angch/sentrylogmon/detectors"
	"github.com/getsentry/sentry-go"
)

func BenchmarkMonitorLoop(b *testing.B) {
	// Generate 10000 lines, 10% match
	var lines []string
	for i := 0; i < 10000; i++ {
		if i%10 == 0 {
			lines = append(lines, "2023-01-01 12:00:00 ERROR something bad happened")
		} else {
			lines = append(lines, "2023-01-01 12:00:00 INFO all is good")
		}
	}
	content := strings.Join(lines, "\n")

	// Create detector
	det, err := detectors.NewGenericDetector("ERROR")
	if err != nil {
		b.Fatalf("Failed to create detector: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		r := strings.NewReader(content)
		scanner := bufio.NewScanner(r)
		// Use the same buffer size logic as in monitor.go
		buf := make([]byte, 0, MaxScanTokenSize)
		scanner.Buffer(buf, MaxScanTokenSize)

		for scanner.Scan() {
			lineBytes := scanner.Bytes()
			if det.Detect(lineBytes) {
				// mimicking simple work
				line := string(lineBytes)
				_ = line
			}
		}
	}
}

func BenchmarkProcessMatch(b *testing.B) {
	// Setup Sentry Mock
	transport := &MockTransport{}
	_ = sentry.Init(sentry.ClientOptions{
		Transport: transport,
	})

	mon := &Monitor{
		Source: &MockSource{content: ""},
	}
	// Pre-allocate buffer to avoid nil check issues if any (though slice defaults to nil is fine)

	// Use a long line
	line := "[100.0] " + strings.Repeat("a", 100)
	lineBytes := []byte(line)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mon.processMatch(lineBytes)
	}
}
