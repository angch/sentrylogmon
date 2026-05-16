## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-01-27 - Manual Timestamp Parsing Optimization
**Learning:** Go's `time.Parse` involves heavy allocation and reflection overhead. When the log format is extremely strict (like Nginx Error logs at exactly 19 bytes), manually extracting digit components with custom helpers (`atoi2`, `atoi4`) and initializing `time.Date` completely avoids the string allocation from `time.Parse` while massively speeding up processing time.
**Action:** For strict format log parsing on hot paths, replace `time.Parse` with manual index-based extraction and basic digit math if the layout is completely predictable.
