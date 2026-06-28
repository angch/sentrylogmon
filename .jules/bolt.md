## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.
## 2024-06-28 - Optimizing Nginx Error Timestamp Parsing
**Learning:** `time.Parse` has significant overhead for simple, fixed-format timestamp parsing like those in Nginx error logs (`2006/01/02 15:04:05`). Replacing it with manual byte extraction and passing the components directly to `time.Date(..., time.UTC)` can drastically improve parsing performance and eliminate allocations.
**Action:** When a known, fixed-length log format exists and its parsing represents a bottleneck (e.g., regex/reflect-based time.Parse), replace it with a manual byte extraction implementation utilizing helpers like `atoi4`/`atoi2` and `time.Date`. Make sure to explicitly define the `time.UTC` location since `time.Parse` assumes UTC when no timezone is provided in the format layout.
