## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-03-16 - Manual byte parsing vs time.Parse
**Learning:** `time.Parse` has significant overhead and allocates memory even when the layout string and target string match exactly. By replacing it with manual byte parsing (e.g., `atoi4` and `atoi2`) and using `time.Date`, we can avoid the allocation and achieve a ~3x performance improvement for simple timestamp formats like Nginx error logs.
**Action:** When a high-throughput path parses logs, manually extract time components using simple byte scanning and string to int conversions rather than using regex and `time.Parse`.
