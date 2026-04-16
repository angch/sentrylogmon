## 2023-10-27 - Monitor Memory Allocation Optimization
**Learning:** In the Go `monitor` package, `MaxScanTokenSize` is 1MB. Re-allocating this buffer inside the restart loop (in `Monitor.Start`) and the benchmark `b.N` loop causes significant memory churn and garbage collection (GC) pressure.
**Action:** Always hoist large buffer allocations out of loops (like reading chunks or resetting stream readers) when the buffer can be safely reused, to reduce memory allocations and GC overhead.
