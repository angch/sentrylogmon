## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-05-26 - Optimize Timestamp Parsing
**Learning:** Using `time.Parse` inside a hot loop or frequently called log parsing function like `ParseNginxError` causes unnecessary allocations overhead, even for strict known formats.
**Action:** When the format is strict and known (e.g. `2006/01/02 15:04:05`), use manual parsing with helper functions like `atoi2` and `atoi4` directly against the `[]byte` slice, then construct the timestamp with `time.Date(..., time.UTC)` to bypass string allocation and general-purpose parsing overhead.
