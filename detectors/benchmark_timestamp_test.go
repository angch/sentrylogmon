package detectors

import (
	"testing"
	"time"
)

func BenchmarkSyslogTimestamp_Regex(b *testing.B) {
	line := []byte("<34>Oct 27 10:00:00 myhost myprogram[123]: message")
	// Expected match: "Oct 27 10:00:00"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if indices := TimestampRegexSyslog.FindSubmatchIndex(line); len(indices) >= 4 {
			tsStr := string(line[indices[2]:indices[3]])
			if _, err := time.Parse(time.Stamp, tsStr); err != nil {
				b.Fatal(err)
			}
		} else {
			b.Fatal("should match")
		}
	}
}

func BenchmarkSyslogTimestamp_Manual(b *testing.B) {
	line := []byte("<34>Oct 27 10:00:00 myhost myprogram[123]: message")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, _, ok := ParseSyslogTimestamp(line); !ok {
			b.Fatal("should match")
		}
	}
}

func BenchmarkNginxAccessTimestamp_Regex(b *testing.B) {
	line := []byte(`127.0.0.1 - - [27/Oct/2023:10:00:00 +0000] "GET / HTTP/1.1" 200 1234`)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if indices := TimestampRegexNginxAccess.FindSubmatchIndex(line); len(indices) >= 4 {
			tsStr := string(line[indices[2]:indices[3]])
			if _, err := time.Parse("02/Jan/2006:15:04:05 -0700", tsStr); err != nil {
				b.Fatal(err)
			}
		} else {
			b.Fatal("should match")
		}
	}
}

func BenchmarkNginxAccessTimestamp_Manual(b *testing.B) {
	line := []byte(`127.0.0.1 - - [27/Oct/2023:10:00:00 +0000] "GET / HTTP/1.1" 200 1234`)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, _, ok := ParseNginxAccess(line); !ok {
			b.Fatal("should match")
		}
	}
}
