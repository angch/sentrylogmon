## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-02-23 - JSON Unmarshalling Performance Optimization
**Learning:** JSON unmarshalling is computationally expensive. When attempting to detect values associated with specific keys in structured logs, unmarshalling every log line just to find it doesn't match the required key/pattern incurs unnecessary overhead.
**Action:** Implement a fast-path rejection mechanism by pre-compiling the expected JSON key into a byte slice representing its serialized form (e.g., `[]byte("\"key\"")` in Go, `b"\"key\""` in Rust/Zig) and using `bytes.Contains` (or equivalent) to quickly discard lines that do not contain the key before attempting full JSON unmarshalling. Ensure to replicate this cross-language parity across Go, Rust, and Zig implementations.
