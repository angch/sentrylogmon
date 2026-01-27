.PHONY: build test clean build-bench benchmark

build:
	CGO_ENABLED=0 GOARCH=amd64 go build -ldflags "-s -w" -o sentrylogmon .

test:
	go test -v ./...

clean:
	rm -f sentrylogmon loggen bench_config.yaml benchmark_output.txt

build-bench: build
	go build -o loggen ./cmd/loggen

benchmark: build-bench
	@echo "Generating benchmark config..."
	@echo "monitors:" > bench_config.yaml
	@echo "  - name: bench-nginx" >> bench_config.yaml
	@echo "    type: command" >> bench_config.yaml
	@echo "    args: ./loggen -size 100MB -format nginx" >> bench_config.yaml
	@echo "    format: nginx" >> bench_config.yaml
	@echo "sentry:" >> bench_config.yaml
	@echo "  dsn: https://examplePublicKey@o0.ingest.sentry.io/0" >> bench_config.yaml
	@echo ""
	@echo "Running benchmark..."
	@/bin/bash -c "time ./sentrylogmon --config bench_config.yaml --oneshot" 2> benchmark_output.txt || true
	@cat benchmark_output.txt
	@echo "Benchmark complete."
