# Builder stage
FROM golang:1.24 AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binaries
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/sentrylogmon .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/loggen ./cmd/loggen
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/sentry-mock ./cmd/sentry-mock

# Runtime stage
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /bin/sentrylogmon /bin/sentrylogmon
COPY --from=builder /bin/loggen /bin/loggen
COPY --from=builder /bin/sentry-mock /bin/sentry-mock

# Create directory for logs
RUN mkdir -p /logs && chmod 777 /logs

CMD ["/bin/sentrylogmon"]
