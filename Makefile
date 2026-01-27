.PHONY: build test clean build-bench benchmark check-prereqs install-prereqs build-go build-rust build-all test-go test-rust test-all clean-go clean-rust clean-all

# Default target builds both
all: build-all

# Check prerequisites
check-prereqs:
	@echo "Checking prerequisites..."
	@echo -n "Go: "
	@which go >/dev/null 2>&1 && go version || echo "NOT FOUND"
	@echo -n "Rust/Cargo: "
	@which cargo >/dev/null 2>&1 && cargo --version || echo "NOT FOUND"
	@echo ""
	@echo "Prerequisites check complete."
	@echo "If any are missing, run 'make install-prereqs' to install them."

# Install prerequisites (requires sudo for system packages)
install-prereqs:
	@echo "Installing prerequisites..."
	@echo "Note: This requires internet access and may require sudo privileges."
	@echo ""
	@# Check and install Go
	@if ! which go >/dev/null 2>&1; then \
		echo "Installing Go..."; \
		if [ -f /etc/debian_version ]; then \
			echo "Detected Debian/Ubuntu system"; \
			echo "Please install Go manually from https://golang.org/dl/ or run:"; \
			echo "  wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz"; \
			echo "  sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz"; \
			echo "  export PATH=\$$PATH:/usr/local/go/bin"; \
		else \
			echo "Please install Go from https://golang.org/dl/"; \
		fi; \
	else \
		echo "Go is already installed: $$(go version)"; \
	fi
	@# Check and install Rust
	@if ! which cargo >/dev/null 2>&1; then \
		echo "Installing Rust..."; \
		curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y; \
		echo "Rust installed. Please run: source $$HOME/.cargo/env"; \
	else \
		echo "Rust is already installed: $$(cargo --version)"; \
	fi
	@echo ""
	@echo "Prerequisites installation complete."

# Build Go binary
build-go:
	@echo "Building Go binary..."
	CGO_ENABLED=0 GOARCH=amd64 go build -ldflags "-s -w" -o sentrylogmon .
	@echo "Go binary built: sentrylogmon"

# Build Rust binary
build-rust:
	@echo "Building Rust binary..."
	cd rust && cargo build --release
	@echo "Rust binary built: rust/target/release/sentrylogmon"

# Build both binaries
build-all: build-go build-rust
	@echo ""
	@echo "Both binaries built successfully!"
	@ls -lh sentrylogmon rust/target/release/sentrylogmon

# Alias for backward compatibility
build: build-go

# Test Go code
test-go:
	go test -v ./...

# Test Rust code
test-rust:
	cd rust && cargo test

# Test both
test-all: test-go test-rust

# Alias for backward compatibility
test: test-go

# Clean Go artifacts
clean-go:
	rm -f sentrylogmon loggen bench_config.yaml benchmark_output.txt

# Clean Rust artifacts
clean-rust:
	cd rust && cargo clean

# Clean all artifacts
clean-all: clean-go clean-rust

# Alias for backward compatibility
clean: clean-go

build-bench: build-go
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
