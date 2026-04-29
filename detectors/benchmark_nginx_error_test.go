package detectors

import (
	"testing"
)

func BenchmarkNginxErrorTimestamp_Manual(b *testing.B) {
	line := []byte("2023/10/27 10:00:00 [error] 123#123: *456 message")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, _, ok := ParseNginxError(line); !ok {
			b.Fatal("should match")
		}
	}
}
