## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-02-09 - Nginx Error Timestamp Parsing Optimized
**Learning:** Manual byte-parsing with `time.Date` for timestamp parsing achieves zero allocations (excluding the slice operation which escapes to heap) and is up to 3.5x faster than using `time.Parse` because it avoids string creation overhead inside time.Parse and format string processing.
**Action:** Always prefer manual byte parsing and `time.Date` for known fixed-width timestamp formats in hot loops to dramatically improve performance and eliminate internal allocation overhead of time.Parse.
