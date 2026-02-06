package detectors

import (
	"bytes"
	"regexp"
	"testing"
)

func BenchmarkGenericDetector_Literal(b *testing.B) {
	pattern := "error"
	detector, _ := NewGenericDetector(pattern)
	line := []byte("This is a log line containing an error message.")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !detector.Detect(line) {
			b.Fatal("should have detected")
		}
	}
}

func BenchmarkGenericDetector_Regex(b *testing.B) {
	pattern := "err[or]+"
	detector, _ := NewGenericDetector(pattern)
	line := []byte("This is a log line containing an error message.")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !detector.Detect(line) {
			b.Fatal("should have detected")
		}
	}
}

func BenchmarkBytesContains(b *testing.B) {
	pattern := []byte("error")
	line := []byte("This is a log line containing an error message.")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !bytes.Contains(line, pattern) {
			b.Fatal("should have detected")
		}
	}
}

func BenchmarkRegexpMatch(b *testing.B) {
	pattern := "error"
	re, _ := regexp.Compile(pattern)
	line := []byte("This is a log line containing an error message.")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !re.Match(line) {
			b.Fatal("should have detected")
		}
	}
}

// BenchmarkDmesgDetector_Detect measures allocations in the DmesgDetector.Detect hot path.
// This benchmark exercises:
// - Regex parsing of dmesg lines (timestamp + header extraction)
// - Error pattern detection
// - Context header tracking
func BenchmarkDmesgDetector_Detect(b *testing.B) {
	detector := NewDmesgDetector()
	// Realistic dmesg lines with timestamps and headers
	lines := [][]byte{
		[]byte("[787739.009553] ata1.00: exception Emask 0x10 SAct 0x10000 SErr 0x40d0000"),
		[]byte("[787739.009558] ata1.00: irq_stat 0x00000040, connection status changed"),
		[]byte("[787739.009559] ata1: SError: { PHYRdyChg CommWake 10B8B DevExch }"),
		[]byte("[787739.009562] ata1.00: failed command: READ FPDMA QUEUED"),
		[]byte("[787739.898553] ata1: SATA link up 6.0 Gbps (SStatus 133 SControl 300)"),
		[]byte("[787739.929456] ata1.00: configured for UDMA/133"),
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, line := range lines {
			detector.Detect(line)
		}
	}
}

// BenchmarkDmesgDetector_FullLifecycle measures Detect + ExtractTimestamp for an error line.
func BenchmarkDmesgDetector_FullLifecycle(b *testing.B) {
	detector := NewDmesgDetector()
	line := []byte("[787739.009553] ata1.00: exception Emask 0x10 SAct 0x10000 SErr 0x40d0000")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if detector.Detect(line) {
			detector.ExtractTimestamp(line)
		}
	}
}

// BenchmarkDmesgDetector_ContextLine measures Detect + ExtractTimestamp for a context line.
func BenchmarkDmesgDetector_ContextLine(b *testing.B) {
	detector := NewDmesgDetector()
	// Set up state
	errorLine := []byte("[787739.009553] ata1.00: exception Emask 0x10 SAct 0x10000 SErr 0x40d0000")
	detector.Detect(errorLine) // Sets lastMatchHeader

	// Related line
	contextLine := []byte("[787739.009558] ata1.00: irq_stat 0x00000040, connection status changed")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Detect should return true (context)
		if detector.Detect(contextLine) {
			detector.ExtractTimestamp(contextLine)
		} else {
			b.Fatal("Should be detected as context")
		}
	}
}

func BenchmarkJsonDetector(b *testing.B) {
	d, err := NewJsonDetector("level:error")
	if err != nil {
		b.Fatalf("Failed to create detector: %v", err)
	}

	line := []byte(`{"level":"error","msg":"something went wrong","time":"2023-10-27T10:00:00Z"}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !d.Detect(line) {
			b.Fatal("Expected match")
		}
		// Also simulate getting context as Monitor does
		d.GetContext(line)
	}
}

func BenchmarkJsonDetector_NoMatch(b *testing.B) {
	d, err := NewJsonDetector("level:error")
	if err != nil {
		b.Fatalf("Failed to create detector: %v", err)
	}

	line := []byte(`{"level":"info","msg":"everything is fine","time":"2023-10-27T10:00:00Z"}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if d.Detect(line) {
			b.Fatal("Expected no match")
		}
	}
}
