## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-03-09 - JSON Fast-Path Rejection Optimization
**Learning:** Calling `json.Unmarshal` is extremely slow and dominates execution time in the `JsonDetector.Detect` loop (~2500ns+ per call). However, most log lines won't contain the specific key the detector is searching for.
**Action:** When searching for a specific key in JSON strings, pre-calculate the JSON-marshalled representation of the key (e.g., `json.Marshal("level")`) and use `bytes.Contains(line, searchBytes)` as a fast-path rejection before invoking `json.Unmarshal`. This reduces non-matching line checks to ~100ns (a ~25x improvement) and skips parsing entirely.
