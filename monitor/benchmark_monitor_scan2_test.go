package monitor

import (
	"bufio"
	"strings"
	"testing"
)

func BenchmarkScannerReuse(b *testing.B) {
	line := strings.Repeat("a", 100) + "\n"
	content := strings.Repeat(line, 1000)
    buf := make([]byte, 0, MaxScanTokenSize)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := strings.NewReader(content)
		scanner := bufio.NewScanner(r)
		scanner.Buffer(buf, MaxScanTokenSize)

		for scanner.Scan() {
			_ = scanner.Bytes()
		}
	}
}
