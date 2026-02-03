#!/bin/bash
set -e

# Cleanup function
cleanup() {
    echo "Cleaning up..."
    kill $(jobs -p) 2>/dev/null || true
    rm -f sentry-mock loggen sentrylogmon access.log
}
trap cleanup EXIT

echo "Building binaries..."
make build-e2e
# make build-e2e builds sentry-mock and loggen. We also need sentrylogmon (built by make build-go)
make build-go

echo "Starting Sentry Mock..."
./sentry-mock > mock.log 2>&1 &
MOCK_PID=$!
sleep 2

echo "Starting SentryLogMon..."
# Use absolute path for file or ensure PWD is correct
touch access.log
export SENTRY_DSN="http://testkey@localhost:8080/1"
export SENTRY_ENVIRONMENT="local-test"
./sentrylogmon --file access.log --format nginx --verbose > mon.log 2>&1 &
MON_PID=$!
sleep 2

echo "Generating Logs..."
./loggen --size 1MB --format nginx >> access.log

echo "Waiting for processing..."
sleep 5

echo "Checking events..."
RESPONSE=$(curl -s http://localhost:8080/events)

if [ "${#RESPONSE}" -gt 5 ] && echo "$RESPONSE" | grep -q "nginx"; then
    echo "SUCCESS: Events received."
    echo "Sample event data: ${RESPONSE:0:100}..."
else
    echo "FAILURE: No expected events found."
    echo "Mock Logs:"
    cat mock.log
    echo "Monitor Logs:"
    cat mon.log
    exit 1
fi

echo "Local E2E tests passed!"
