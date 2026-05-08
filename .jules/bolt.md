## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-01-27 - Manual parsing for simple layouts is much faster than time.Parse
**Learning:** The standard library `time.Parse` is surprisingly slow due to reflection and general-purpose parsing overhead, taking ~250ns for a simple format like `2006/01/02 15:04:05`. Manually parsing individual digits with fixed offsets and calling `time.Date()` can reduce the time by ~3x (down to ~70ns).
**Action:** In high-throughput hot paths dealing with strictly formatted data like logs, avoid `time.Parse` for known layouts and use manual byte extraction combined with `time.Date(...)`.
