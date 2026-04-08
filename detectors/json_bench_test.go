package detectors

import (
	"fmt"
	"testing"
)

func BenchmarkExtractStringSlow(b *testing.B) {
	var val interface{} = "this is a test string"
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("%v", val)
	}
}

func BenchmarkExtractStringFast(b *testing.B) {
	var val interface{} = "this is a test string"
	for i := 0; i < b.N; i++ {
		var valStr string
		if s, ok := val.(string); ok {
			valStr = s
		} else {
			valStr = fmt.Sprintf("%v", val)
		}
		_ = valStr
	}
}
