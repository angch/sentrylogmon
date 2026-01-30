package sources

import (
	"bytes"
	"io"
	"testing"
)

// BenchmarkReadLoop compares the performance of allocating the buffer inside vs outside the read loop.
// It uses bytes.Reader to minimize I/O overhead and focus on memory allocation costs.
func BenchmarkReadLoop(b *testing.B) {
	// Create 10MB of dummy data
	data := make([]byte, 1024*1024*10)

	b.Run("AllocInside_4KB", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			r := bytes.NewReader(data)
			for {
				buf := make([]byte, 4096)
				n, err := r.Read(buf)
				if n <= 0 || err == io.EOF {
					break
				}
			}
		}
	})

	b.Run("AllocOutside_4KB", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			r := bytes.NewReader(data)
			buf := make([]byte, 4096)
			for {
				n, err := r.Read(buf)
				if n <= 0 || err == io.EOF {
					break
				}
			}
		}
	})

	b.Run("AllocOutside_32KB", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			r := bytes.NewReader(data)
			buf := make([]byte, 32768)
			for {
				n, err := r.Read(buf)
				if n <= 0 || err == io.EOF {
					break
				}
			}
		}
	})
}
