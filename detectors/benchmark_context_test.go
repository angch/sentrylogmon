package detectors

import (
	"testing"
)

func BenchmarkGetContext(b *testing.B) {
	line := []byte(`{"level":"error","msg":"something went wrong","time":"2023-10-27T10:00:00Z"}`)
	d, _ := NewJsonDetector("level:error")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d.Detect(line)
		d.GetContext(line)
	}
}
