package monitor

import (
	"bufio"
	"strings"
	"testing"
)

func BenchmarkScanner(b *testing.B) {
	line := strings.Repeat("a", 100) + "\n"
	content := strings.Repeat(line, 1000)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := strings.NewReader(content)
		scanner := bufio.NewScanner(r)
		buf := make([]byte, 0, MaxScanTokenSize)
		scanner.Buffer(buf, MaxScanTokenSize)

		for scanner.Scan() {
			_ = scanner.Bytes()
		}
	}
}
