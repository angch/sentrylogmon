package detectors

import (
	"testing"
)

func BenchmarkNginxErrorTimestamp_Regex(b *testing.B) {
	line := []byte("2023/10/27 10:00:00 [error] 12345#0: *67890 open() \"/var/www/html/missing.html\" failed (2: No such file or directory)")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := TimestampRegexNginxError.FindSubmatch(line)
		if len(matches) > 1 {
			_ = string(matches[1])
		}
	}
}

func BenchmarkNginxErrorTimestamp_Manual(b *testing.B) {
	line := []byte("2023/10/27 10:00:00 [error] 12345#0: *67890 open() \"/var/www/html/missing.html\" failed (2: No such file or directory)")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseNginxError(line)
	}
}
