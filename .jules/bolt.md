## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-03-11 - Manual Byte Scanning Beats `time.Parse`
**Learning:** `time.Parse` has significant overhead and allocates memory. By replacing `time.Parse` with manual byte scanning and integer conversion helpers (`atoi2`, `atoi4`) on log formats with rigid structures (like Nginx Error logs), processing speed improves dramatically (>3.4x) while eliminating intermediate string allocation overheads.
**Action:** When parsing timestamps that have a fixed layout, prefer manual byte verification and offset extraction over `time.Parse`. Be sure to strictly validate byte offsets (`/`, ` `, `:`) and value bounds before calling `time.Date`.
