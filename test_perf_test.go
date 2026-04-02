package main

import (
	"fmt"
	"testing"
)

func BenchmarkSprintf(b *testing.B) {
	var val interface{} = "error"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("%v", val)
	}
}

func BenchmarkTypeAssert(b *testing.B) {
	var val interface{} = "error"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if s, ok := val.(string); ok {
			_ = s
		} else {
			_ = fmt.Sprintf("%v", val)
		}
	}
}
