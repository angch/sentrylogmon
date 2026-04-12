## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.
## 2026-01-27 - Fast-path rejection for log parsing
**Learning:** In log processing hot paths (like `JsonDetector`), applying fast-path byte sequence rejection (e.g., `bytes.Contains` in Go, `line.windows().any()` in Rust) before heavy parsing (like JSON unmarshaling) dramatically reduces overhead for non-matching lines.
**Action:** Always check if a quick string/byte search can pre-filter lines before allocating and parsing complex structures.
## 2026-01-27 - Precomputing search bytes for zero-allocation fast-paths
**Learning:** Adding a fast-path search like `bytes.Contains` inside a hot loop (e.g., `Detect` method) is completely defeated if the search string itself is allocated or formatted on every invocation (e.g., `[]byte("\"" + field + "\"")` or `format!("\"{}\"").into_bytes()`).
**Action:** Always precompute static search strings during constructor/initialization and store them in the struct (as `[]byte` or `Vec<u8>`) to ensure the fast path operates with zero allocations.
