package detectors

import (
	"regexp"
	"strconv"
	"time"
)

var (
	// 2006-01-02T15:04:05Z07:00 or 2006-01-02 15:04:05
	TimestampRegexISO = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)`)
	// Oct 27 10:00:00 or <34>Oct 27 10:00:00
	TimestampRegexSyslog = regexp.MustCompile(`^(?:<\d{1,3}>)?([A-Z][a-z]{2}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})`)

	// 2023/10/27 10:00:00
	TimestampRegexNginxError = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})`)
	// [27/Oct/2023:10:00:00 +0000]
	TimestampRegexNginxAccess = regexp.MustCompile(`\[(\d{2}/[A-Z][a-z]{2}/\d{4}:\d{2}:\d{2}:\d{2}\s+[+-]\d{4})\]`)
)

func ParseISO8601(line []byte) (float64, string, bool) {
	if len(line) < 19 {
		return 0, "", false
	}
	// Check YYYY-MM-DD
	if line[4] != '-' || line[7] != '-' {
		return 0, "", false
	}
	// Check [T ]
	if line[10] != 'T' && line[10] != ' ' {
		return 0, "", false
	}
	// Check HH:MM:SS
	if line[13] != ':' || line[16] != ':' {
		return 0, "", false
	}

	// Determine end of timestamp
	end := 19
	// Scan fractional seconds
	if end < len(line) && line[end] == '.' {
		end++
		for end < len(line) && line[end] >= '0' && line[end] <= '9' {
			end++
		}
	}
	// Scan timezone
	if end < len(line) {
		if line[end] == 'Z' {
			end++
		} else if line[end] == '+' || line[end] == '-' {
			end++
			// Expect digits and colon
			for end < len(line) && ((line[end] >= '0' && line[end] <= '9') || line[end] == ':') {
				end++
			}
		}
	}

	tsStr := string(line[:end])

	// Parse
	var t time.Time
	var err error

	if line[10] == 'T' {
		t, err = time.Parse(time.RFC3339Nano, tsStr)
	} else {
		// Space separator
		// Try full layout if timezone present
		if len(tsStr) > 19 {
			t, err = time.Parse("2006-01-02 15:04:05.999999999Z07:00", tsStr)
		} else {
			t, err = time.Parse("2006-01-02 15:04:05", tsStr)
		}
	}

	if err == nil {
		return float64(t.Unix()) + float64(t.Nanosecond())/1e9, tsStr, true
	}
	return 0, "", false
}

func ParseNginxError(line []byte) (float64, string, bool) {
	// 2023/10/27 10:00:00
	if len(line) < 19 {
		return 0, "", false
	}
	if line[4] != '/' || line[7] != '/' || line[10] != ' ' || line[13] != ':' || line[16] != ':' {
		return 0, "", false
	}

	tsStr := string(line[:19])
	t, err := time.Parse("2006/01/02 15:04:05", tsStr)
	if err == nil {
		return float64(t.Unix()) + float64(t.Nanosecond())/1e9, tsStr, true
	}
	return 0, "", false
}

func ParseDmesgTimestamp(line []byte) (float64, string, bool) {
	if len(line) < 3 || line[0] != '[' {
		return 0, "", false
	}

	limit := 32
	if len(line) < limit {
		limit = len(line)
	}

	closeBracket := -1
	for i := 1; i < limit; i++ {
		if line[i] == ']' {
			closeBracket = i
			break
		}
	}
	if closeBracket == -1 {
		return 0, "", false
	}

	start := 1
	for start < closeBracket && line[start] == ' ' {
		start++
	}

	if start == closeBracket {
		return 0, "", false
	}

	numBytes := line[start:closeBracket]
	for _, b := range numBytes {
		if (b < '0' || b > '9') && b != '.' {
			return 0, "", false
		}
	}

	tsStr := string(numBytes)
	ts, err := strconv.ParseFloat(tsStr, 64)
	if err != nil {
		return 0, "", false
	}
	return ts, tsStr, true
}
