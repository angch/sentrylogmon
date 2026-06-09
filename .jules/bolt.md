## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-01-27 - Zero-Allocation Parsing via Custom Conversion
**Learning:** Utilizing Go's `time.Parse` incurs significant allocation overhead due to formatting analysis and potential heap allocations. Replacing it with targeted, index-based character matching and custom integer conversion helpers (`atoi2`, `atoi4`) paired with `time.Date` drastically reduces execution time (by ~70%) without altering correctness for strict timestamp formats.
**Action:** When a timestamp follows a highly deterministic format, bypass standard library parsers in favor of manual byte slicing and validation to minimize GC pressure on critical logging pathways.
