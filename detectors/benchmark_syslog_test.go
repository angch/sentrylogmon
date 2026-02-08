package detectors

import (
	"testing"
)

func BenchmarkParseSyslogTimestamp(b *testing.B) {
	line := []byte("Oct 27 10:00:00 myhost myprocess[123]: message")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseSyslogTimestamp(line)
	}
}
