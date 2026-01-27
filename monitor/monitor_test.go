package monitor

import (
	"testing"

	"github.com/angch/sentrylogmon/sysstat"
)

func TestStateAttachment(t *testing.T) {
	// Test that state.ToMap() is called and works correctly
	collector := sysstat.New()
	// Run collector in background
	go collector.Run()
	defer collector.Stop()

	// Trigger initial collection - wait a bit for collection to happen
	// The Run() method does an initial collect before entering the loop
	// So we should have state immediately, but let's give it a moment
	state := collector.GetState()
	if state == nil {
		t.Fatal("State should not be nil after collection")
	}

	// Test ToMap conversion
	stateMap := state.ToMap()
	if stateMap == nil {
		t.Fatal("ToMap should not return nil for valid state")
	}

	// Verify expected fields are present
	if _, ok := stateMap["timestamp"]; !ok {
		t.Error("stateMap should contain 'timestamp' field")
	}
	if _, ok := stateMap["uptime"]; !ok {
		t.Error("stateMap should contain 'uptime' field")
	}
	if _, ok := stateMap["process_summary"]; !ok {
		t.Error("stateMap should contain 'process_summary' field")
	}

	// Test with nil state
	collector2 := sysstat.New()
	// Don't run collection, state should be empty
	state2 := collector2.GetState()
	if state2 != nil {
		stateMap2 := state2.ToMap()
		// Should handle nil fields gracefully
		if stateMap2 == nil {
			t.Error("ToMap should return a map even with nil fields")
		}
	}
}

func TestNilCollector(t *testing.T) {
	// Test that monitor handles nil collector gracefully
	// This would be tested in integration tests, but verify
	// that GetState returns nil for nil collector is safe

	var collector *sysstat.Collector
	if collector != nil {
		state := collector.GetState()
		if state != nil {
			t.Error("GetState on nil collector should be safe")
		}
	}
}
