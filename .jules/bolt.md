## 2026-02-09 - Prometheus Metrics for Monitor Lag
**Learning:** Understanding parsing and monitoring lag for lines is useful to assess system performance.
**Action:** Always extract timestamps correctly to calculate lags if needed for logging.
## 2026-05-22 - Prometheus Metrics for Monitor Lag
**Learning:** Calculating processing lag accurately requires matching time precision. When lag values can be sub-second (especially with Prometheus DefBuckets), `float64(time.Now().UnixNano()) / 1e9` should be used instead of `time.Now().Unix()` to avoid loss of precision and unintuitive histogram clustering.
**Action:** Always verify timestamp granularity when generating metrics comparing timestamps, avoiding accidental truncation to whole seconds.
