# Testing Guidelines

- **Non-Destructive Testing**: All tests must be non-destructive. If deletes are to be used for testing, create a copy of test data/databases first.
- **Cleanup**: A clean up step can be run idempotently to undo all changes in tests.
- **Go Tests**: Use table-driven tests with subtests (`t.Run(...)`). Run tests with `go test ./...`.

### Unit Tests

- Test each LogSource implementation independently
- Mock Sentry client for Monitor tests
- Test pattern matching edge cases in Detector
- Aim for >80% code coverage

### Data-Driven Tests

Integration tests for detectors are located in `testdata/<detector>/`.
- Each test case consists of an input file (`*.txt`) and an expected output file (`*.expect.txt`).
- **Important**: Files not ending in `.txt` are ignored by the test runner. This allows keeping backup files or other artifacts in the directory without breaking tests.
- When adding new test cases, ensure you provide both the input `.txt` file and the corresponding `.expect.txt` file.

### Integration Tests

- End-to-end tests with real log files
- Test error handling and recovery
- Verify Sentry integration with test DSN

### Performance Tests

- Benchmark memory usage with large log files
- Test CPU usage under sustained load
- Verify resource usage stays within "lightweight" criteria (<50MB RAM, <5% CPU)
