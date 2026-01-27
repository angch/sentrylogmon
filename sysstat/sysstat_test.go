package sysstat

import (
	"testing"
	"time"
)

func TestCollect(t *testing.T) {
	c := New()
	c.collect()

	state := c.GetState()
	if state == nil {
		t.Fatal("State should not be nil")
	}

	if state.Uptime == 0 {
		t.Log("Uptime is 0, possibly expected in some environments or very fresh boot")
	}

	if state.Load == nil {
		t.Error("Load should not be nil")
	}

	if state.Memory == nil {
		t.Error("Memory should not be nil")
	}

	if len(state.TopCPU) == 0 {
		t.Log("TopCPU is empty, possibly due to missing ps command or permissions")
	} else {
		if len(state.TopCPU) > 5 {
			t.Error("TopCPU has too many entries")
		}
	}

	if len(state.TopMem) == 0 {
		t.Log("TopMem is empty, possibly due to missing ps command or permissions")
	} else {
		if len(state.TopMem) > 5 {
			t.Error("TopMem has too many entries")
		}
	}

	if time.Since(state.Timestamp) > 1*time.Minute {
		t.Error("Timestamp is too old")
	}
}
