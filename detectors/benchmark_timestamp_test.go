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

func BenchmarkNginxErrorTimestamp_Regex(b *testing.B) {
	line := []byte(`2023/10/27 10:00:00 [error] 123#123: *123 open() "/path" failed`)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if len(line) >= 19 {
			tsStr := string(line[:19])
			if _, err := time.Parse("2006/01/02 15:04:05", tsStr); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkNginxErrorTimestamp_Manual(b *testing.B) {
	line := []byte(`2023/10/27 10:00:00 [error] 123#123: *123 open() "/path" failed`)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, _, ok := ParseNginxError(line); !ok {
			b.Fatal("should match")
		}
	}
}

func BenchmarkNginxErrorTimestamp_ManualFast(b *testing.B) {
	line := []byte(`2023/10/27 10:00:00 [error] 123#123: *123 open() "/path" failed`)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if len(line) < 19 {
			b.Fatal("should match")
		}
		y := atoi4(line[0:4])
		m := atoi2(line[5:7])
		d := atoi2(line[8:10])
		h := atoi2(line[11:13])
		min := atoi2(line[14:16])
		s := atoi2(line[17:19])

		if y < 0 || m < 1 || m > 12 || d < 1 || d > 31 || h > 23 || min > 59 || s > 60 {
			b.Fatal("should match")
		}

		t := time.Date(y, time.Month(m), d, h, min, s, 0, time.UTC)
		_ = t
		_ = string(line[:19])
	}
}
