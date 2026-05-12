## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-05-12 - Nginx Error Timestamp Parsing Allocation
**Learning:** Converting a timestamp to a string using time.Parse with a custom layout allocates heavily and is slow. Directly parsing the known format by extracting year, month, day, hour, minute, and second into an unallocated slice via manual helpers (atoi2, atoi4) and calling time.Date reduces allocations to 0 and improves speed.
**Action:** For performance-critical hot paths, avoid time.Parse for strictly known fixed-length log formats. Extract data components directly without string allocation overhead.
