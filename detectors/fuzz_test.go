package detectors

import (
	"bytes"
	"testing"
)

func FuzzDmesgDetector(f *testing.F) {
	// Seed corpus
	f.Add([]byte(`[    0.000000] Linux version 5.4.0-100-generic`))
	f.Add([]byte(`[  123.456789] ata1.00: exception Emask 0x0 SAct 0x0 SErr 0x0 action 0x0`))
	f.Add([]byte("error\n[ 123.457] related header"))

	f.Fuzz(func(t *testing.T, data []byte) {
		d := NewDmesgDetector()

		// Split input into lines to test state transitions
		lines := bytes.Split(data, []byte{'\n'})
		for _, line := range lines {
			d.Detect(line)
			d.TransformMessage(line)
		}
	})
}

func FuzzGenericDetector(f *testing.F) {
	// Seed corpus
	f.Add([]byte("error"), "error")
	f.Add([]byte("fail"), "(?i)fail")
	f.Add([]byte("nothing"), "abc")

	f.Fuzz(func(t *testing.T, line []byte, pattern string) {
		if pattern == "" {
			return
		}
		d, err := NewGenericDetector(pattern)
		if err != nil {
			return
		}
		d.Detect(line)
	})
}

func FuzzJsonDetector(f *testing.F) {
	// Seed corpus
	f.Add([]byte(`{"level":"error"}`), "level:error")
	f.Add([]byte(`{"msg":"failed"}`), "msg:fail")
	f.Add([]byte(`not json`), "foo:bar")

	f.Fuzz(func(t *testing.T, line []byte, pattern string) {
		if pattern == "" {
			return
		}
		d, err := NewJsonDetector(pattern)
		if err != nil {
			return
		}
		d.Detect(line)
		d.GetContext(line)
	})
}
