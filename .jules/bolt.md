## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-05-04 - Fast-path byte sequence rejection for JSON detection
**Learning:** Parsing JSON (`json.Unmarshal`, `serde_json`, etc.) is incredibly expensive and allocates significant memory. In log processing, most log lines will not match the target criteria.
**Action:** Always apply a fast-path byte sequence rejection (e.g., `bytes.Contains`, `line.windows().any()`, `std.mem.indexOf`) before heavy JSON parsing to dramatically reduce overhead for non-matching lines. Ensure this optimization is replicated across all language ports.
