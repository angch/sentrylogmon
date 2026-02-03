#!/bin/bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
COMPOSE_FILE="$DIR/docker-compose.yml"

# Check for docker-compose
if command -v docker-compose >/dev/null 2>&1; then
    DOCKER_COMPOSE="docker-compose"
elif docker compose version >/dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
else
    echo "Error: docker-compose not found."
    exit 1
fi

echo "Cleaning up any previous run..."
$DOCKER_COMPOSE -f "$COMPOSE_FILE" down -v || true

trap "$DOCKER_COMPOSE -f '$COMPOSE_FILE' down -v" EXIT

echo "Starting E2E tests..."
$DOCKER_COMPOSE -f "$COMPOSE_FILE" up -d --build

echo "Waiting for services to initialize and process logs..."
# Wait loop
MAX_RETRIES=12
for i in $(seq 1 $MAX_RETRIES); do
    sleep 5
    echo "Checking events (attempt $i/$MAX_RETRIES)..."
    RESPONSE=$(curl -s http://localhost:8080/events)

    # Check if we got a valid JSON array response and it's not empty (length > 2 for "[]")
    if [ "${#RESPONSE}" -gt 5 ] && echo "$RESPONSE" | grep -q "nginx"; then
        echo "SUCCESS: Events received."
        echo "Sample event data: ${RESPONSE:0:100}..."
        exit 0
    fi
done

echo "FAILURE: No expected events found after $(($MAX_RETRIES * 5)) seconds."
echo "Response from mock: $RESPONSE"
echo "--------------------------------"
echo "Sentrylogmon Logs:"
$DOCKER_COMPOSE -f "$COMPOSE_FILE" logs sentrylogmon
echo "--------------------------------"
echo "Loggen Logs:"
$DOCKER_COMPOSE -f "$COMPOSE_FILE" logs loggen
exit 1
