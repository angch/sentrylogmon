## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2024-05-15 - Optimizing Fixed Format Timestamp Parsing
**Learning:** In Go, replacing `time.Parse` with manual byte extraction (e.g., custom `atoi2`, `atoi4` helpers) and `time.Date` for fixed-format timestamps avoids reflection and string allocation overhead, providing significant performance improvements in hot paths.
**Action:** When a fixed-format timestamp parsing is identified in a hot path, consider replacing `time.Parse` with manual parsing helpers. Ensure strict bounds validation (e.g., `m < 1 || m > 12`) to match `time.Parse` correctness and avoid regressions.
