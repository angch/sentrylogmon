## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-01-27 - Zero-Allocation Timestamp Parsing
**Learning:** In Go, replacing `time.Parse` with manual byte scanning (e.g., extracting year, month, day directly from `[]byte` using custom `atoi` functions) combined with `time.Date()` significantly improves parsing speed and reduces allocations in hot loops. Allocations drop from 1 alloc/op to 0 until `string()` conversion is performed, and CPU time is reduced by ~60%.
**Action:** For highly predictable log timestamp formats (like Nginx Error logs), avoid `time.Parse`. Validate structure first (e.g., `line[4] == '/'`), extract fields as integers without allocations, and construct the timestamp via `time.Date(..., time.UTC)`. Only convert `[]byte` to `string` if absolutely necessary and only after full validation succeeds to fail fast and avoid garbage generation on non-matching lines.
