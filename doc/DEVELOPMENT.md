# Development & Troubleshooting

## Common Pitfalls for Future Development

### 1. File Monitoring

**Issue**: Using polling instead of inotify/fsnotify can be inefficient

**Solution**: Consider `github.com/fsnotify/fsnotify` for production deployments, but keep polling as fallback for compatibility

### 2. Memory Leaks with Long-Running Processes

**Issue**: Go applications can leak memory with improper goroutine management

**Solution**:
- Always defer `Close()` calls
- Use context for cancellation
- Profile with pprof regularly: `import _ "net/http/pprof"`

### 3. Regex Performance

**Issue**: Complex regexes can be slow on high-volume logs

**Solution**:
- Pre-compile regexes once
- Use simple patterns when possible
- Consider Boyer-Moore or other algorithms for literal string matching

### 4. Sentry Rate Limiting

**Issue**: Sending too many messages can hit Sentry rate limits

**Solution**:
- Implement client-side rate limiting
- Batch similar messages
- Use sampling for high-frequency issues

### 5. Timezone Handling

**Issue**: Log timestamps may be in different timezones

**Solution**:
- Always normalize to UTC
- Preserve original timezone in metadata
- Use Go's `time.Time` type throughout
