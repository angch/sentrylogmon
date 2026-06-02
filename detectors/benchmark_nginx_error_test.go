package detectors

import (
	"testing"
)

func BenchmarkNginxErrorTimestamp_Regex(b *testing.B) {
	line := []byte("2023/10/27 10:00:00 [error] 123#123: *1 open() failed")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if indices := TimestampRegexNginxError.FindSubmatchIndex(line); len(indices) >= 4 {
			// do something
		} else {
			b.Fatal("should match")
		}
	}
}

func BenchmarkNginxErrorTimestamp_Manual(b *testing.B) {
	line := []byte("2023/10/27 10:00:00 [error] 123#123: *1 open() failed")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, _, ok := ParseNginxError(line); !ok {
			b.Fatal("should match")
		}
	}
}
