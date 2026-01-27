package detectors

import (
	"regexp"
	"strings"
	"testing"
)

func BenchmarkGenericDetector_Literal(b *testing.B) {
	pattern := "error"
	detector, _ := NewGenericDetector(pattern)
	line := "This is a log line containing an error message."

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
	line := "This is a log line containing an error message."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !detector.Detect(line) {
			b.Fatal("should have detected")
		}
	}
}

func BenchmarkStringsContains(b *testing.B) {
	pattern := "error"
	line := "This is a log line containing an error message."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !strings.Contains(line, pattern) {
			b.Fatal("should have detected")
		}
	}
}

func BenchmarkRegexpMatchString(b *testing.B) {
	pattern := "error"
	re, _ := regexp.Compile(pattern)
	line := "This is a log line containing an error message."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !re.MatchString(line) {
			b.Fatal("should have detected")
		}
	}
}
