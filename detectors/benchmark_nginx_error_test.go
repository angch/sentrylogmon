package detectors

import (
	"testing"
)

func BenchmarkNginxErrorTimestamp(b *testing.B) {
	line := []byte("2023/10/27 10:00:00 [error] 1234#0: *1 open() \"/usr/share/nginx/html/favicon.ico\" failed (2: No such file or directory)")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, _, ok := ParseNginxError(line); !ok {
			b.Fatal("should match")
		}
	}
}
