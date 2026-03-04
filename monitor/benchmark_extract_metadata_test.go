package monitor

import (
	"testing"
	"github.com/angch/sentrylogmon/detectors"
)

func BenchmarkExtractMetadata(b *testing.B) {
	line := []byte(`{"level":"error","msg":"something went wrong","time":"2023-10-27T10:00:00Z"}`)
	tsStr := "2023-10-27T10:00:00Z"
	d, _ := detectors.NewJsonDetector("level:error")

	// Pre-detect to populate cache
	d.Detect(line)

	m := &Monitor{
		Detector: d,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.extractMetadata(line, tsStr)
	}
}

func BenchmarkExtractMetadata_NoContext(b *testing.B) {
	line := []byte("[100.0] some error")
	tsStr := "100.0"
	d := detectors.NewDmesgDetector()

	m := &Monitor{
		Detector: d,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.extractMetadata(line, tsStr)
	}
}
