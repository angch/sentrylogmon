## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2024-05-29 - time.Parse Optimization
**Learning:** Using `time.Parse` inside high-frequency log parsing functions involves string conversions and generalized layout interpretation which introduces measurable overhead (~250ns/op). For rigid timestamp formats (like Nginx `YYYY/MM/DD HH:MM:SS`), manually extracting fields using simple character math (`atoi4`/`atoi2`) and constructing `time.Date` is significantly faster.
**Action:** When parsing well-defined static timestamp formats in hot paths, avoid `time.Parse`. Opt for manual byte slicing and fast integer parsing, ensuring all boundary and range checks are strictly enforced.
