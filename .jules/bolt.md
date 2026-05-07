## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-05-07 - Nginx Error Timestamp Extraction Fast-path
**Learning:** Using `time.Parse` to parse fixed-format timestamps in log hot paths incurs a significant runtime cost (approx 252 ns/op) compared to a manual extraction and `time.Date` combination (approx 70 ns/op) due to format string processing overhead.
**Action:** When extracting time out of logs with strict structural guarantees, use `atoi2`/`atoi4` helper functions to pull the individual time parts directly from the `[]byte` slice and compile using `time.Date(..., time.UTC)` for near zero-allocation hot paths.
