package detectors

import (
	"testing"
	"time"
)

func BenchmarkNginxErrorTimestamp_Baseline(b *testing.B) {
	line := []byte("2023/10/27 10:00:00 [error] 1234#5678: *9012 connect() failed")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Mimic the old implementation
		if len(line) < 19 {
			continue
		}
		if line[4] != '/' || line[7] != '/' || line[10] != ' ' || line[13] != ':' || line[16] != ':' {
			continue
		}
		tsStr := string(line[:19])
		if _, err := time.Parse("2006/01/02 15:04:05", tsStr); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNginxErrorTimestamp_Manual(b *testing.B) {
	line := []byte("2023/10/27 10:00:00 [error] 1234#5678: *9012 connect() failed")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, _, ok := ParseNginxError(line); !ok {
			b.Fatal("should match")
		}
	}
}
