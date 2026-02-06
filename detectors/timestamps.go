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
	ts, err := parseFloatFromBytes(numBytes)
	if err != nil {
		return 0, "", false
	}
	return ts, tsStr, true
}

func parseFloatFromBytes(b []byte) (float64, error) {
	var integerPart uint64
	var fractionalPart uint64
	var divisor float64 = 1
	var dotSeen bool

	if len(b) == 0 {
		return 0, strconv.ErrSyntax
	}

	for _, c := range b {
		if c >= '0' && c <= '9' {
			if !dotSeen {
				integerPart = integerPart*10 + uint64(c-'0')
			} else {
				fractionalPart = fractionalPart*10 + uint64(c-'0')
				divisor *= 10
			}
		} else if c == '.' {
			if dotSeen {
				return 0, strconv.ErrSyntax
			}
			dotSeen = true
		} else {
			return 0, strconv.ErrSyntax
		}
	}
	return float64(integerPart) + float64(fractionalPart)/divisor, nil
}

func ParseSyslogTimestamp(line []byte) (float64, string, bool) {
	if len(line) < 15 {
		return 0, "", false
	}

	// Skip priority
	offset := 0
	if line[0] == '<' {
		// Find '>'
		// PRI is 1-3 digits. So '>' can be at index 2, 3, or 4.
		// Start searching from index 2
		found := false
		for i := 2; i <= 4 && i < len(line); i++ {
			if line[i] == '>' {
				offset = i + 1
				found = true
				break
			}
		}
		if !found {
			return 0, "", false
		}
	}

	if len(line)-offset < 15 {
		return 0, "", false
	}

	tsBytes := line[offset : offset+15]

	// Check Mmm (Month)
	// Must be capitalized: A-Z
	if tsBytes[0] < 'A' || tsBytes[0] > 'Z' {
		return 0, "", false
	}

	m0, m1, m2 := tsBytes[0], tsBytes[1], tsBytes[2]
	var month time.Month

	switch m0 {
	case 'J':
		if m1 == 'a' && m2 == 'n' {
			month = time.January
		} else if m1 == 'u' && m2 == 'n' {
			month = time.June
		} else if m1 == 'u' && m2 == 'l' {
			month = time.July
		}
	case 'F':
		if m1 == 'e' && m2 == 'b' {
			month = time.February
		}
	case 'M':
		if m1 == 'a' && m2 == 'r' {
			month = time.March
		} else if m1 == 'a' && m2 == 'y' {
			month = time.May
		}
	case 'A':
		if m1 == 'p' && m2 == 'r' {
			month = time.April
		} else if m1 == 'u' && m2 == 'g' {
			month = time.August
		}
	case 'S':
		if m1 == 'e' && m2 == 'p' {
			month = time.September
		}
	case 'O':
		if m1 == 'c' && m2 == 't' {
			month = time.October
		}
	case 'N':
		if m1 == 'o' && m2 == 'v' {
			month = time.November
		}
	case 'D':
		if m1 == 'e' && m2 == 'c' {
			month = time.December
		}
	}

	if month == 0 {
		return 0, "", false
	}

	if tsBytes[3] != ' ' {
		return 0, "", false
	}

	// Day
	var day int
	d1, d2 := tsBytes[4], tsBytes[5]
	if d1 == ' ' {
		if d2 < '0' || d2 > '9' {
			return 0, "", false
		}
		day = int(d2 - '0')
	} else {
		if d1 < '0' || d1 > '9' || d2 < '0' || d2 > '9' {
			return 0, "", false
		}
		day = int(d1-'0')*10 + int(d2-'0')
	}

	if tsBytes[6] != ' ' {
		return 0, "", false
	}

	// Time HH:MM:SS
	h1, h2 := tsBytes[7], tsBytes[8]
	if h1 < '0' || h1 > '9' || h2 < '0' || h2 > '9' {
		return 0, "", false
	}
	hour := int(h1-'0')*10 + int(h2-'0')

	if tsBytes[9] != ':' {
		return 0, "", false
	}

	min1, min2 := tsBytes[10], tsBytes[11]
	if min1 < '0' || min1 > '9' || min2 < '0' || min2 > '9' {
		return 0, "", false
	}
	minute := int(min1-'0')*10 + int(min2-'0')

	if tsBytes[12] != ':' {
		return 0, "", false
	}

	s1, s2 := tsBytes[13], tsBytes[14]
	if s1 < '0' || s1 > '9' || s2 < '0' || s2 > '9' {
		return 0, "", false
	}
	sec := int(s1-'0')*10 + int(s2-'0')

	// Validation
	if day < 1 || day > 31 || hour > 23 || minute > 59 || sec > 60 {
		return 0, "", false
	}

	// Year Inference
	now := time.Now()
	// Use UTC to match time.Parse(time.Stamp) behavior which defaults to UTC
	currentYear := now.Year()
	t := time.Date(currentYear, month, day, hour, minute, sec, 0, time.UTC)

	// Simple heuristic for year boundary
	if t.Sub(now) > 30*24*time.Hour {
		t = t.AddDate(-1, 0, 0)
	}

	return float64(t.Unix()) + float64(t.Nanosecond())/1e9, string(tsBytes), true
}
