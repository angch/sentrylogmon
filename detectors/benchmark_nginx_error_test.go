package detectors

import (
	"testing"
)

func BenchmarkParseNginxError(b *testing.B) {
	line := []byte("2023/10/27 10:00:00 [error] 1234#0: *1 open()")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = ParseNginxError(line)
	}
}
