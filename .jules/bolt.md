## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-04-29 - time.Parse overhead in Hot Paths
**Learning:** In log processing hot paths, `time.Parse` incurs significant allocation overhead due to formatting setup and internal allocations. We found that replacing `time.Parse` with manual extraction of timestamp parts directly from the `[]byte` slice to construct a `time.Date` is nearly 4x faster and avoids allocations.
**Action:** Always favor manual date extraction and `time.Date` over `time.Parse` when parsing strictly defined timestamp formats in hot paths.
