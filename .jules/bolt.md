## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-04-25 - Scanner Buffer Allocation in Loops
**Learning:** Allocating large buffers (e.g., `make([]byte, 0, MaxScanTokenSize)`) inside a source stream restart loop causes unnecessary memory churn and GC pressure when a stream frequently reconnects or resets. Moving the allocation outside the loop allows the buffer to be reused safely across scanner instances.
**Action:** Always verify that buffer allocations are hoisted outside of loop bodies unless explicit isolation per iteration is strictly required.
