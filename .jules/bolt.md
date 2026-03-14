## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-03-14 - JSON Detection Fast-Path Opt
**Learning:** Parsing JSON fields for matching logic unmarshals unconditionally leading to thousands of ns per log line. Using a pre-calculated marshalled field check directly against the log slice via `bytes.Contains(line, marshalledField)` skips Unmarshaling drastically improving performance for unmatched payloads (~2500ns/19 allocs -> ~90ns/0 allocs).
**Action:** When implementing generic JSON format matching tools, establish a fast-path condition checking for specific required text or marshalled keys before parsing the log payloads.
