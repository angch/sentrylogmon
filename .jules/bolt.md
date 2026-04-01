## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2024-04-01 - Fast-Path JSON Parsing & Zero-Alloc Type Checking
**Learning:** Full JSON parsing (`json.Unmarshal`) on every incoming log line is prohibitively expensive (~2500ns, 18 allocs). Pre-calculating a target key (`json.Marshal(field)`) and executing a raw `bytes.Contains` check creates a massive fast-path optimization (rejects non-matching lines in ~24ns with 0 allocs). Additionally, avoiding `fmt.Sprintf` for standard type conversion by explicitly prioritizing type assertions (e.g., `val.(string)`) prevents further latency and heap allocations on hot code paths.
**Action:** In high-throughput environments like logging monitors, always evaluate raw byte scanning for fast-path rejection before allocating structured decoding processes, and substitute generic type coercion routines with explicit type assertions wherever the type is known or highly predictable.
