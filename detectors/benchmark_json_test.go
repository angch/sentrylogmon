package detectors

import (
	"testing"
)

func BenchmarkJsonDetector_NoMatch_MissingField(b *testing.B) {
	d, err := NewJsonDetector("level:error")
	if err != nil {
		b.Fatalf("Failed to create detector: %v", err)
	}

	line := []byte(`{"msg":"everything is fine","time":"2023-10-27T10:00:00Z"}`)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if d.Detect(line) {
			b.Fatal("Expected no match")
		}
	}
}
