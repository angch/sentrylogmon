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
