## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-03-02 - Fast-Path Rejection for Expensive Operations
**Learning:** `json.Unmarshal` is very expensive to call on every log line when the specified JSON key doesn't even exist in the string. Using `bytes.Contains(line, searchBytes)` where `searchBytes` is the JSON-marshalled key name acts as a highly effective and very fast fast-path rejection that reduces ~3500ns execution time to ~35ns (~100x improvement) for non-matching lines.
**Action:** When working with expensive unmarshalling functions like `json.Unmarshal`, evaluate if a quick string/byte search can pre-qualify the payload to avoid the expensive call altogether for irrelevant payloads.
