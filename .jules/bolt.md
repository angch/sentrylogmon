## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.
## 2026-02-27 - Manual Timestamp Parsing in Hot Paths
**Learning:** Parsing fixed-format timestamps with `time.Parse` has significant overhead due to reflection, validation overhead, and memory allocation inside standard libraries. Manually parsing byte slices using basic index boundaries and local conversion functions like `atoi2` and `atoi4` dramatically reduces execution time (~70% improvement for Nginx log format).
**Action:** When working on very hot execution paths (like log processing agents), consider avoiding `time.Parse` for known fixed-width formats and instead manually extract integers for `time.Date`. Always explicitly include timezone handling (e.g. `time.UTC`) if mirroring `time.Parse` without a layout timezone.
