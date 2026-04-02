package detectors

import (
	"testing"
)

func BenchmarkJsonDetector_Detect_Match(b *testing.B) {
	d, _ := NewJsonDetector("level:error")
	line := []byte(`{"level":"error", "msg":"something went wrong", "id": 12345, "data": {"a": 1, "b": 2}}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Detect(line)
	}
}

func BenchmarkJsonDetector_Detect_NoMatch(b *testing.B) {
	d, _ := NewJsonDetector("level:error")
	line := []byte(`{"level":"info", "msg":"all good", "id": 12345, "data": {"a": 1, "b": 2}}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Detect(line)
	}
}

func BenchmarkJsonDetector_Detect_MissingField(b *testing.B) {
	d, _ := NewJsonDetector("level:error")
	line := []byte(`{"msg":"all good", "id": 12345, "data": {"a": 1, "b": 2}}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Detect(line)
	}
}
