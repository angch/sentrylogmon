## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-03-21 - [Timestamp Parsing Bottleneck in Go]
**Learning:** `time.Parse` in Go incurs significant overhead and memory allocations (even for exact format matches) because it handles generalized string parsing and layout interpretation dynamically. When processing high-throughput log files like Nginx logs where the format and length are strictly known, this becomes a major bottleneck.
**Action:** In high-frequency log stream ingestion paths (like Nginx Error log parsing), always replace `time.Parse` with manual byte slice extraction using zero-allocation integer parsers (e.g., `atoi4`, `atoi2`), followed by explicitly creating a timestamp with `time.Date`. This can yield a >10x performance improvement (~1100ns to ~85ns) and drop memory allocations to zero.
