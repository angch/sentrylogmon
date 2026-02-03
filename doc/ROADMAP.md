# Roadmap & Future Enhancements

## Planned Features

1. **Configuration File Support** (Implemented)
   - YAML/TOML format for defining multiple monitors
   - Hot-reload capability to update config without restart (Future)
   - Validate config before applying (Future)

2. **Metrics and Health Monitoring**
   - Expose Prometheus metrics (lines processed, issues detected, etc.)
   - Health check endpoint for monitoring the monitor itself
   - Self-monitoring: detect if log sources stop producing output

3. **Buffering and Batching**
   - Buffer messages during Sentry outages
   - Batch similar messages to reduce API calls
   - Persistent queue for guaranteed delivery

4. **Advanced Filtering**
   - Whitelist patterns (exclude matches from reporting)
   - Rate limiting per pattern to avoid spam
   - Time-based filtering (only monitor during business hours)
   - Severity levels

5. **Structured Log Support**
   - Parse JSON logs and match on fields
   - Extract metadata from structured logs for Sentry context
   - Support for common formats (Logrus, Zap, etc.)

6. **Multi-tenancy**
   - Support multiple Sentry projects/DSNs
   - Route different log sources to different Sentry projects

## Non-Goals

The following are explicitly **not** goals for this project:

1. **Log Aggregation Storage**: Use Elasticsearch, Loki, or similar for this
2. **Complex Analytics**: Use dedicated log analysis tools
3. **Log Transformation**: Keep the tool focused on monitoring and forwarding
4. **GUI**: Remain a CLI tool; use Sentry's web interface for visualization
5. **Plugin System**: Keep the codebase simple; fork if extensive customization needed
