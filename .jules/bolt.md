## 2026-01-27 - Benchmarking Allocation Optimizations
**Learning:** When benchmarking optimizations that avoid allocations (like using `[]byte` instead of `string`), ensure the benchmark includes the cost of the allocation being removed. Benchmarking only the processing function might show a regression if the allocation happened in the caller.
**Action:** Always benchmark the full path or simulate the inputs realistically (e.g. including conversions) to prove the benefit of reducing allocations.

## 2026-01-27 - Regexp Allocation Limits
**Learning:** Go's `regexp.FindSubmatchIndex` still allocates the `[]int` result slice. While it reduces memory usage compared to `FindSubmatch` (which allocates `[][]byte`), it doesn't eliminate allocations entirely. Zero-alloc regex capturing requires different libraries or manual parsing.
**Action:** For hot paths requiring zero allocations, prefer manual parsing (`bytes.Index`, etc.) over `regexp` if feasible, otherwise accept the reduced but non-zero allocation of `FindSubmatchIndex`.

## 2026-03-03 - Nginx Error Timestamp Parsing
**Learning:** `time.Parse` incurs significant performance overhead (approx 640ns/op) when parsing timestamps because of layout string processing and potentially heap allocations. Replacing it with manual byte-level arithmetic parser (`atoi2`, `atoi4`) drastically decreases overhead.
**Action:** When creating high-throughput detectors, always use custom byte scanning logic and `time.Date()` instead of relying on generic standard library format string parsers like `time.Parse`.

## 2026-03-03 - Nginx Error Timestamp Validation Caveat
**Learning:** `time.Date()` will silently normalize and mask invalid inputs like negative time values (e.g. `time.Date(2023, 10, 27, -5, 0, 0)`), wrapping them to previous hours/days without returning an error. When replacing `time.Parse` with manual byte parsing, bounds checking must strictly enforce both upper (e.g., `> 23`) AND lower bounds (e.g., `< 0`) for *all* integer parts.
**Action:** When manually parsing bytes to integers for `time.Date`, explicitly validate `field >= 0` across all fields before instantiation to ensure corrupt logs are correctly rejected.
