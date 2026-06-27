package detectors

import (
	"testing"
)

var dummyResult float64
var dummyStr string
var dummyBool bool

func BenchmarkParseNginxError(b *testing.B) {
	line := []byte("2023/10/27 10:00:00 [error] 123#123: *456 foo")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dummyResult, dummyStr, dummyBool = ParseNginxError(line)
	}
}
