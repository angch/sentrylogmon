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
	line := []byte("2023/10/27 10:00:00 [error] 123#123: *456 connect() failed")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if indices := TimestampRegexNginxError.FindSubmatchIndex(line); len(indices) >= 4 {
			// Simulating extraction and parse
			tsStr := string(line[indices[2]:indices[3]])
			_, _ = time.Parse("2006/01/02 15:04:05", tsStr)
		} else {
			b.Fatal("not matched")
		}
	}
}

func BenchmarkNginxErrorTimestamp_Manual(b *testing.B) {
	line := []byte("2023/10/27 10:00:00 [error] 123#123: *456 connect() failed")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if _, _, ok := ParseNginxError(line); !ok {
			b.Fatal("not parsed")
		}
	}
}

func BenchmarkNginxErrorTimestamp_ManualOpt(b *testing.B) {
	line := []byte("2023/10/27 10:00:00 [error] 123#123: *456 connect() failed")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Mock optimized version
		if len(line) < 19 {
			b.Fatal("not matched")
		}
		if line[4] != '/' || line[7] != '/' || line[10] != ' ' || line[13] != ':' || line[16] != ':' {
			b.Fatal("not matched")
		}

		y := atoi4(line[0:4])
		m := atoi2(line[5:7])
		d := atoi2(line[8:10])
		h := atoi2(line[11:13])
		min := atoi2(line[14:16])
		s := atoi2(line[17:19])

		// Basic validation
		if y < 1970 || y > 3000 || m < 1 || m > 12 || d < 1 || d > 31 ||
			h < 0 || h > 23 || min < 0 || min > 59 || s < 0 || s > 60 {
			b.Fatal("not matched")
		}

		t := time.Date(y, time.Month(m), d, h, min, s, 0, time.UTC)
		_ = float64(t.Unix()) + float64(t.Nanosecond())/1e9
		_ = string(line[:19])
	}
}

var globalTs float64
var globalTsStr string
var globalOk bool

func BenchmarkNginxErrorTimestamp_ManualOpt_Real(b *testing.B) {
	line := []byte("2023/10/27 10:00:00 [error] 123#123: *456 connect() failed")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if len(line) < 19 {
			b.Fatal("not matched")
		}
		if line[4] != '/' || line[7] != '/' || line[10] != ' ' || line[13] != ':' || line[16] != ':' {
			b.Fatal("not matched")
		}

		y := atoi4(line[0:4])
		m := atoi2(line[5:7])
		d := atoi2(line[8:10])
		h := atoi2(line[11:13])
		min := atoi2(line[14:16])
		s := atoi2(line[17:19])

		// Basic validation
		if y < 1970 || y > 3000 || m < 1 || m > 12 || d < 1 || d > 31 ||
			h < 0 || h > 23 || min < 0 || min > 59 || s < 0 || s > 60 {
			b.Fatal("not matched")
		}

		t := time.Date(y, time.Month(m), d, h, min, s, 0, time.UTC)
		globalTs = float64(t.Unix())
		globalTsStr = string(line[:19])
		globalOk = true
	}
}

func BenchmarkNginxErrorTimestamp_Manual_Real(b *testing.B) {
	line := []byte("2023/10/27 10:00:00 [error] 123#123: *456 connect() failed")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		globalTs, globalTsStr, globalOk = ParseNginxError(line)
		if !globalOk {
			b.Fatal("not parsed")
		}
	}
}
