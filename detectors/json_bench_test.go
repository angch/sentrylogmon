package detectors

import (
	"testing"
)

func BenchmarkJsonDetector_Detect_Match(b *testing.B) {
	d, _ := NewJsonDetector("level:error")
	line := []byte(`{"level":"error", "id":1}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Detect(line)
	}
}

func BenchmarkJsonDetector_Detect_NoMatch(b *testing.B) {
	d, _ := NewJsonDetector("level:error")
	line := []byte(`{"level":"info", "id":1}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Detect(line)
	}
}

func BenchmarkJsonDetector_Detect_MissingField(b *testing.B) {
	d, _ := NewJsonDetector("level:error")
	line := []byte(`{"id":1, "msg":"hello world"}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Detect(line)
	}
}
