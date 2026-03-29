package monitor

import (
	"testing"
)

func BenchmarkExtractSyslogPriority(b *testing.B) {
	lines := [][]byte{
		[]byte("<34>Oct 11 22:14:15 mymachine su: 'su root' failed"),
		[]byte("<0>Emergency"),
		[]byte("<191>Debug"),
		[]byte("No PRI here"),
		[]byte("<abc>Invalid"),
		[]byte("<1234>Too long"),
		[]byte("<1>Short"),
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, line := range lines {
			extractSyslogPriority(line)
		}
	}
}
