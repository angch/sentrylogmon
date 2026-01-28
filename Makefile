.PHONY: all build build-go build-zig build-rust build-all clean clean-go clean-zig clean-rust clean-all check-prereqs install-prereqs help test test-go test-zig test-rust test-all validate-zig compare-size build-bench benchmark

# Default target builds all implementations
all: build-all

# Build all three implementations
build-all: build-go build-zig build-rust
	@echo ""
	@echo "All binaries built successfully!"
	@echo "Binary sizes:"
	@ls -lh sentrylogmon 2>/dev/null || echo "  Go: not built"
	@ls -lh zig/zig-out/bin/sentrylogmon-zig 2>/dev/null || echo "  Zig: not built"
	@ls -lh rust/target/release/sentrylogmon 2>/dev/null || echo "  Rust: not built"

# Alias for backward compatibility (builds Go + Zig + Rust)
build: build-all

# Build Go binary
build-go:
	@echo "Building Go binary..."
	CGO_ENABLED=0 GOARCH=amd64 go build -ldflags "-s -w" -o sentrylogmon .
	@echo "Go binary built: sentrylogmon"

# Build Zig binary
build-zig:
	@export PATH=$(PWD)/.tools/zig:$$PATH; \
	if which zig > /dev/null 2>&1; then \
		echo "Building Zig binary..."; \
		cd zig && zig build -Doptimize=ReleaseSafe -Dstrip=true && \
		echo "Zig binary built: zig/zig-out/bin/sentrylogmon-zig"; \
	else \
		echo "Zig not found. Skipping Zig build."; \
		echo "Install Zig with 'make install-prereqs' or from https://ziglang.org/download/"; \
	fi

# Build Zig binary with maximum size optimization
build-zig-small:
	@export PATH=$(PWD)/.tools/zig:$$PATH; \
	if which zig > /dev/null 2>&1; then \
		echo "Building Zig binary (size optimized)..."; \
		cd zig && zig build -Doptimize=ReleaseSmall -Dstrip=true && \
		echo "Zig binary built: zig/zig-out/bin/sentrylogmon-zig"; \
	else \
		echo "Zig not found. Skipping Zig build."; \
		echo "Install Zig with 'make install-prereqs' or from https://ziglang.org/download/"; \
	fi

# Build Rust binary
build-rust:
	@if which cargo > /dev/null 2>&1; then \
		echo "Building Rust binary (static linking with Zig)..."; \
		chmod +x scripts/build_rust_static.sh; \
		./scripts/build_rust_static.sh && \
		echo "Rust binary built: rust/target/x86_64-unknown-linux-musl/release/sentrylogmon"; \
	else \
		echo "Rust/Cargo not found. Skipping Rust build."; \
		echo "Install Rust with 'make install-prereqs' or from https://rustup.rs/"; \
	fi

# Check if all prerequisites are installed
check-prereqs:
	@echo "Checking prerequisites..."
	@echo -n "Checking for Go... "
	@which go > /dev/null 2>&1 && echo "✓ Found: $$(go version)" || echo "✗ Not found"
	@echo -n "Checking for Zig... "
	@export PATH=$(PWD)/.tools/zig:$$PATH; \
	which zig > /dev/null 2>&1 && echo "✓ Found: $$(zig version)" || echo "✗ Not found"
	@echo -n "Checking for Rust/Cargo... "
	@which cargo > /dev/null 2>&1 && echo "✓ Found: $$(cargo --version)" || echo "✗ Not found"
	@echo -n "Checking for curl... "
	@which curl > /dev/null 2>&1 && echo "✓ Found" || echo "✗ Not found"
	@echo -n "Checking for tar... "
	@which tar > /dev/null 2>&1 && echo "✓ Found" || echo "✗ Not found"
	@echo ""
	@echo "Summary:"
	@which go > /dev/null 2>&1 || (echo "  - Go is not installed. Run 'make install-prereqs' or install manually."; exit 0)
	@export PATH=$(PWD)/.tools/zig:$$PATH; \
	which zig > /dev/null 2>&1 || (echo "  - Zig is not installed. Run 'make install-prereqs' or install manually."; exit 0)
	@which cargo > /dev/null 2>&1 || (echo "  - Rust is not installed. Run 'make install-prereqs' or install manually."; exit 0)
	@echo "Prerequisites check complete."

# Install prerequisites (Go, Zig, and Rust)
install-prereqs:
	@echo "Installing prerequisites..."
	@echo ""
	@echo "This will attempt to install Go, Zig, and Rust if not already present."
	@echo ""
	@# Check and install Go
	@if ! which go > /dev/null 2>&1; then \
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
	@echo ""
	@# Check and install Zig
	@export PATH=$(PWD)/.tools/zig:$$PATH; \
	ZIG_VERSION="0.15.2"; \
	CURRENT_ZIG=$$(zig version 2>/dev/null || echo "none"); \
	if [ "$$CURRENT_ZIG" != "$$ZIG_VERSION" ]; then \
		echo "Installing Zig $$ZIG_VERSION (found $$CURRENT_ZIG)..."; \
		rm -rf .tools/zig; \
		mkdir -p .tools; \
		echo "Attempting to download Zig $$ZIG_VERSION..."; \
		cd .tools && \
		(curl -sL https://ziglang.org/download/$$ZIG_VERSION/zig-x86_64-linux-$$ZIG_VERSION.tar.xz -o zig.tar.xz && \
		tar -xf zig.tar.xz && \
		mv zig-x86_64-linux-$$ZIG_VERSION zig && \
		rm zig.tar.xz && \
		echo "" && \
		echo "Zig downloaded to $$(pwd)/zig" && \
		echo "Add to PATH: export PATH=$$(pwd)/zig:\$$PATH" && \
		echo "Or install system-wide: sudo cp -r $$(pwd)/zig /usr/local/") || \
		(echo "" && \
		echo "Failed to download Zig automatically." && \
		echo "Please install Zig manually from: https://ziglang.org/download/" && \
		echo "Or use your package manager if available."); \
	else \
		echo "Zig is already installed: $$CURRENT_ZIG"; \
	fi
	@echo ""
	@# Check and install Rust
	@if ! which cargo > /dev/null 2>&1; then \
		echo "Installing Rust..."; \
		curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y; \
		echo "Rust installed. Please run: source $$HOME/.cargo/env"; \
	else \
		echo "Rust is already installed: $$(cargo --version)"; \
	fi
	@# Install Rust target for static linking
	@echo "Installing Rust musl target..."
	@rustup target add x86_64-unknown-linux-musl || echo "Failed to add target, please ensure rustup is installed"
	@echo ""
	@echo "Installation complete!"
	@echo "If tools were installed locally, add them to your PATH:"
	@echo "  export PATH=$$(pwd)/.tools/zig:\$$PATH"
	@echo "If Rust was installed, run: source \$$HOME/.cargo/env"

# Clean build artifacts
clean-all: clean-go clean-zig clean-rust
	@echo "All build artifacts cleaned"

# Clean Go artifacts
clean-go:
	@echo "Cleaning Go build artifacts..."
	rm -f sentrylogmon loggen bench_config.yaml benchmark_output.txt
	@echo "Go artifacts cleaned"

# Clean Zig artifacts
clean-zig:
	@echo "Cleaning Zig build artifacts..."
	rm -rf zig/zig-out
	rm -rf zig/.zig-cache
	@echo "Zig artifacts cleaned"

# Clean Rust artifacts
clean-rust:
	@if which cargo > /dev/null 2>&1; then \
		echo "Cleaning Rust build artifacts..."; \
		cd rust && cargo clean && \
		echo "Rust artifacts cleaned"; \
	else \
		echo "Cargo not found, skipping Rust clean"; \
	fi

# Alias for backward compatibility (clean Go only)
clean: clean-go

# Run all tests
test-all: test-go test-zig test-rust
	@echo "All tests completed"

# Run Go tests
test-go:
	@echo "Running Go tests..."
	go test -v ./...

# Run Zig tests
test-zig:
	@export PATH=$(PWD)/.tools/zig:$$PATH; \
	if which zig > /dev/null 2>&1; then \
		echo "Running Zig tests..."; \
		cd zig && zig build test; \
	else \
		echo "Running Zig validation (Zig not installed)..."; \
		cd zig && ./validate.sh; \
	fi

# Run Rust tests
test-rust:
	@if which cargo > /dev/null 2>&1; then \
		echo "Running Rust tests (static target with Zig)..."; \
		chmod +x scripts/test_rust.sh; \
		./scripts/test_rust.sh; \
	else \
		echo "Cargo not found, skipping Rust tests"; \
	fi

# Alias for backward compatibility (test Go only)
test: test-go

# Validate Zig implementation
validate-zig:
	@echo "Validating Zig implementation..."
	@cd zig && ./validate.sh

# Compare binary sizes
compare-size: build-all
	@echo "Binary size comparison:"
	@echo "=========================================="
	@if [ -f sentrylogmon ]; then \
		ls -lh sentrylogmon | awk '{print "Go binary:   " $$5 " (" $$9 ")"}'; \
	fi
	@if [ -f zig/zig-out/bin/sentrylogmon-zig ]; then \
		ls -lh zig/zig-out/bin/sentrylogmon-zig | awk '{print "Zig binary:  " $$5}'; \
	fi
	@if [ -f rust/target/release/sentrylogmon ]; then \
		ls -lh rust/target/release/sentrylogmon | awk '{print "Rust binary: " $$5}'; \
	fi
	@echo "=========================================="

# Build benchmark tool
build-bench: build-go
	go build -o loggen ./cmd/loggen

# Run benchmark
benchmark: build-bench
	@echo "Running benchmark..."
	@echo "This will test the log monitoring performance"

# Help target
help:
	@echo "sentrylogmon Makefile"
	@echo "====================="
	@echo ""
	@echo "Available targets:"
	@echo "  make                    - Build all implementations (Go, Zig, Rust)"
	@echo "  make build-all          - Build all implementations"
	@echo "  make build              - Alias for build-all"
	@echo "  make build-go           - Build only the Go binary"
	@echo "  make build-zig          - Build only the Zig binary (ReleaseSafe)"
	@echo "  make build-zig-small    - Build Zig binary with maximum size optimization"
	@echo "  make build-rust         - Build only the Rust binary"
	@echo "  make check-prereqs      - Check if Go, Zig, and Rust are installed"
	@echo "  make install-prereqs    - Download and install prerequisites"
	@echo "  make clean-all          - Remove all build artifacts"
	@echo "  make clean-go           - Remove Go build artifacts"
	@echo "  make clean-zig          - Remove Zig build artifacts"
	@echo "  make clean-rust         - Remove Rust build artifacts"
	@echo "  make clean              - Alias for clean-go"
	@echo "  make test-all           - Run all tests"
	@echo "  make test-go            - Run Go tests"
	@echo "  make test-zig           - Run Zig tests (or validation if Zig not installed)"
	@echo "  make test-rust          - Run Rust tests"
	@echo "  make test               - Alias for test-go"
	@echo "  make validate-zig       - Validate Zig code structure without building"
	@echo "  make compare-size       - Compare binary sizes of all implementations"
	@echo "  make build-bench        - Build benchmark tool"
	@echo "  make benchmark          - Run benchmark"
	@echo "  make help               - Show this help message"
	@echo ""
	@echo "Prerequisites:"
	@echo "  - Go 1.19 or later"
	@echo "  - Zig 0.13.0 or later (optional)"
	@echo "  - Rust/Cargo (optional)"
	@echo ""
	@echo "Examples:"
	@echo "  make check-prereqs      # Check if tools are installed"
	@echo "  make install-prereqs    # Download and install tools"
	@echo "  make build-all          # Build all implementations"
	@echo "  make compare-size       # Compare binary sizes"
