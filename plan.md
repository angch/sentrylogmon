1. Modify `metrics/metrics.go` to add `ProcessingLagSeconds` Prometheus histogram metric and register it.
2. Modify `monitor/monitor.go`:
   - Add `metricProcessingLag` to `Monitor` struct.
   - Initialize it in `New`.
   - In `processLoop`, calculate lag using the timestamp extracted (if > 0). `lag := float64(time.Now().UnixNano())/1e9 - timestamp`.
   - Note: From memory `In Go, when computing sub-second precision elapsed time or lag metrics (e.g., for Prometheus histograms), use float64(time.Now().UnixNano()) / 1e9 instead of float64(time.Now().Unix()). The .Unix() method truncates to whole seconds, causing fast sub-second operations to incorrectly report zero lag.`
3. Add a test for this metric in `monitor/observability_test.go` checking the histogram values. Note from memory: `When testing Prometheus vector metrics (e.g., GaugeVec, HistogramVec) in Go subtests, use unique label values per subtest to prevent state leakage and ensure clean, isolated metric assertions.`
4. Update `TODO.md` to check off the issue.
