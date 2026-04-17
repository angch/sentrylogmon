## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-04-17 - Fast-path rejection for heavy operations
**Learning:** In log processing hot paths, `json.Unmarshal` (or equivalent parsing) introduces massive allocation and CPU overhead. If the target field doesn't exist, we parse JSON unnecessarily.
**Action:** Implemented a pre-parsing byte sequence check (`bytes.Contains`, `line.windows().any()`, `std.mem.indexOf`) for the field name before attempting to unmarshal the payload. This reduces misses' latency from ~2800ns to ~130ns (95% speedup) and is safe when field names map accurately to strings. This optimization must be maintained across all language implementations (Go, Rust, Zig).
