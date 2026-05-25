## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-05-25 - Sub-second Precision Log Lag Metrics
**Learning:** In Go, measuring high-precision time differences for observability requires correct representation of sub-second intervals. `time.Now().Unix()` truncates to the nearest second, causing precision loss in lag calculations.
**Action:** Use `float64(time.Now().UnixNano()) / 1e9` instead of `time.Now().Unix()` when calculating time differences between log generation timestamps (which often have micro/nanosecond precision) and current processing time.
