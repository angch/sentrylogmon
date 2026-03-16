package detectors

import (
	"testing"
	"time"
)

func BenchmarkNginxErrorTimestamp_Regex(b *testing.B) {
	line := []byte(`2023/10/27 10:00:00 [error] 1234#0: *1 open() "/usr/share/nginx/html/favicon.ico" failed`)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if indices := TimestampRegexNginxError.FindSubmatchIndex(line); len(indices) >= 4 {
			tsStr := string(line[indices[2]:indices[3]])
			if _, err := time.Parse("2006/01/02 15:04:05", tsStr); err != nil {
				b.Fatal(err)
			}
		} else {
			b.Fatal("should match")
		}
	}
}

func BenchmarkNginxErrorTimestamp_Manual(b *testing.B) {
	line := []byte(`2023/10/27 10:00:00 [error] 1234#0: *1 open() "/usr/share/nginx/html/favicon.ico" failed`)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, _, ok := ParseNginxError(line); !ok {
			b.Fatal("should match")
		}
	}
}
