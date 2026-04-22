## 2026-04-22 - Scanner Buffer Allocation in Loop
**Learning:** In Go, `bufio.Scanner.Buffer(buf, max)` does not take ownership of the `buf` slice's lifecycle. Reallocating a large buffer (e.g., 1MB `MaxScanTokenSize`) inside a reader loop creates significant memory churn and GC pressure on every source reconnect or benchmark iteration.
**Action:** Allocate large buffers (`make([]byte, 0, MaxScanTokenSize)`) once outside the `for` loop that creates new scanners. The `bufio.Scanner` will safely use the provided buffer.

## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.
