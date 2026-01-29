# Performance Optimization Checklist

This document tracks potential performance improvements for the `sentrylogmon` project.

## Go Implementation

- [x] **Lazy Load Process Command Lines in `sysstat`**
  - **Current Behavior:** The `getProcessStats` function iterates over all processes and calls `p.CmdLine()` (which reads `/proc/<pid>/cmdline`) for *every* process before sorting.
  - **Proposed Change:** Collect only lightweight stats (PID, CPU, Memory) first, sort to find the top K processes, and *then* fetch the command line only for those top K processes.
  - **Expected Impact:** Significant reduction in I/O operations (from N reads to ~10 reads per collection cycle).

- [x] **Optimize Buffering in `Monitor`**
  - **Current Behavior:** Uses `[]string` to buffer log lines and `strings.Join` to concatenate them before sending to Sentry. This causes extra allocations.
  - **Proposed Change:** Use `strings.Builder` or `bytes.Buffer` to accumulate log lines directly.
  - **Expected Impact:** Reduced memory allocations and GC pressure.

- [x] **Optimize `DmesgDetector` Allocations**
  - **Current Behavior:** Uses `FindSubmatch` which allocates slices of byte slices, and frequently converts `[]byte` to `string`.
  - **Proposed Change:** Use `FindSubmatchIndex` to work with indices and avoid slice allocation. Minimize string conversions by checking bytes directly where possible.
  - **Expected Impact:** Reduced allocations in the detection hot path.
  - **Result:** 65% reduction in bytes allocated (891→313 B/op), 40% fewer allocations (15→9 allocs/op).

## Rust Implementation

- [x] **Optimize `sysstat` System Refresh**
  - **Current Behavior:** Calls `sys.refresh_all()` which updates all system information including all processes and their details.
  - **Proposed Change:** Use `sysinfo`'s more granular refresh methods (e.g., `refresh_cpu`, `refresh_memory`, `refresh_processes_specifics`) to only update what is necessary.
  - **Expected Impact:** Lower CPU usage during system stats collection.
  - **Result:** Replaced `refresh_all()` with `refresh_memory()` and `refresh_processes()` to skip disk, network, and sensor refreshes.

## General

- [ ] **Profile Memory Usage**
  - Run the application with `pprof` enabled under load to identify any other memory bottlenecks.
