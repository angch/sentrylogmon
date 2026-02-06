package detectors

import (
	"testing"
)

func BenchmarkParseISO8601(b *testing.B) {
	line := []byte("2023-10-27T10:00:00.123456Z message")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, ok := ParseISO8601(line)
		if !ok {
			b.Fatal("should match")
		}
	}
}

func BenchmarkParseISO8601_Simple(b *testing.B) {
	line := []byte("2023-10-27 10:00:00 message")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, ok := ParseISO8601(line)
		if !ok {
			b.Fatal("should match")
		}
	}
}
