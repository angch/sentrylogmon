.PHONY: all build-go build-zig build clean check-prereqs install-prereqs help test

# Default target
all: build

# Build both Go and Zig binaries
build: build-go build-zig

# Build Go binary
build-go:
	@echo "Building Go binary..."
	go build -o sentrylogmon main.go
	@echo "Go binary built: sentrylogmon"

# Build Zig binary
build-zig:
	@if which zig > /dev/null 2>&1; then \
		echo "Building Zig binary..."; \
		cd zig && zig build -Doptimize=ReleaseSafe && \
		echo "Zig binary built: zig/zig-out/bin/sentrylogmon-zig"; \
	else \
		echo "Zig not found. Skipping Zig build."; \
		echo "Install Zig with 'make install-prereqs' or from https://ziglang.org/download/"; \
	fi

# Build Zig binary with maximum size optimization
build-zig-small:
	@if which zig > /dev/null 2>&1; then \
		echo "Building Zig binary (size optimized)..."; \
		cd zig && zig build -Doptimize=ReleaseSmall && \
		echo "Zig binary built: zig/zig-out/bin/sentrylogmon-zig"; \
	else \
		echo "Zig not found. Skipping Zig build."; \
		echo "Install Zig with 'make install-prereqs' or from https://ziglang.org/download/"; \
	fi

# Check if all prerequisites are installed
check-prereqs:
	@echo "Checking prerequisites..."
	@echo -n "Checking for Go... "
	@which go > /dev/null 2>&1 && echo "✓ Found: $$(go version)" || echo "✗ Not found"
	@echo -n "Checking for Zig... "
	@which zig > /dev/null 2>&1 && echo "✓ Found: $$(zig version)" || echo "✗ Not found"
	@echo -n "Checking for curl... "
	@which curl > /dev/null 2>&1 && echo "✓ Found" || echo "✗ Not found"
	@echo -n "Checking for tar... "
	@which tar > /dev/null 2>&1 && echo "✓ Found" || echo "✗ Not found"
	@echo ""
	@echo "Summary:"
	@which go > /dev/null 2>&1 || (echo "  - Go is not installed. Run 'make install-prereqs' or install manually."; exit 0)
	@which zig > /dev/null 2>&1 || (echo "  - Zig is not installed. Run 'make install-prereqs' or install manually."; exit 0)
	@echo "All prerequisites are installed!"

# Install prerequisites (Go and Zig)
install-prereqs:
	@echo "Installing prerequisites..."
	@echo ""
	@echo "This will attempt to download Go and Zig if not already present."
	@echo "Installation will be done in /tmp/sentrylogmon-tools"
	@echo ""
	@mkdir -p /tmp/sentrylogmon-tools
	@# Check and install Go
	@if ! which go > /dev/null 2>&1; then \
		echo "Installing Go..."; \
		cd /tmp/sentrylogmon-tools && \
		curl -sL https://go.dev/dl/go1.24.12.linux-amd64.tar.gz -o go.tar.gz && \
		tar -xzf go.tar.gz && \
		rm go.tar.gz && \
		echo ""; \
		echo "Go downloaded to /tmp/sentrylogmon-tools/go"; \
		echo "Add to PATH: export PATH=/tmp/sentrylogmon-tools/go/bin:\$$PATH"; \
		echo "Or install system-wide: sudo tar -C /usr/local -xzf /tmp/sentrylogmon-tools/go.tar.gz"; \
	else \
		echo "Go is already installed: $$(go version)"; \
	fi
	@# Check and install Zig
	@if ! which zig > /dev/null 2>&1; then \
		echo ""; \
		echo "Installing Zig..."; \
		echo "Attempting to download Zig 0.11.0..."; \
		cd /tmp/sentrylogmon-tools && \
		(curl -sL https://ziglang.org/download/0.11.0/zig-linux-x86_64-0.11.0.tar.xz -o zig.tar.xz && \
		tar -xf zig.tar.xz && \
		mv zig-linux-x86_64-0.11.0 zig && \
		rm zig.tar.xz && \
		echo "" && \
		echo "Zig downloaded to /tmp/sentrylogmon-tools/zig" && \
		echo "Add to PATH: export PATH=/tmp/sentrylogmon-tools/zig:\$$PATH" && \
		echo "Or install system-wide: sudo cp -r /tmp/sentrylogmon-tools/zig /usr/local/") || \
		(echo "" && \
		echo "Failed to download Zig automatically." && \
		echo "Please install Zig manually from: https://ziglang.org/download/" && \
		echo "Or use your package manager if available."); \
	else \
		echo "Zig is already installed: $$(zig version)"; \
	fi
	@echo ""
	@echo "Installation complete!"
	@echo "If tools were installed to /tmp, add them to your PATH:"
	@echo "  export PATH=/tmp/sentrylogmon-tools/go/bin:/tmp/sentrylogmon-tools/zig:\$$PATH"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f sentrylogmon
	rm -rf zig/zig-out
	rm -rf zig/.zig-cache
	@echo "Clean complete"

# Run tests
test: test-go

# Run Go tests
test-go:
	@echo "Running Go tests..."
	go test -v ./...

# Run Zig tests
test-zig:
	@echo "Running Zig tests..."
	cd zig && zig build test

# Compare binary sizes
compare-size: build
	@echo "Binary size comparison:"
	@echo "----------------------------------------"
	@ls -lh sentrylogmon | awk '{print "Go binary:  " $$5 " (" $$9 ")"}'
	@if [ -f zig/zig-out/bin/sentrylogmon-zig ]; then \
		ls -lh zig/zig-out/bin/sentrylogmon-zig | awk '{print "Zig binary: " $$5 " (" $$9 ")"}'; \
	fi
	@echo "----------------------------------------"

# Help target
help:
	@echo "sentrylogmon Makefile"
	@echo "====================="
	@echo ""
	@echo "Available targets:"
	@echo "  make                    - Build both Go and Zig binaries (same as 'make build')"
	@echo "  make build              - Build both Go and Zig binaries"
	@echo "  make build-go           - Build only the Go binary"
	@echo "  make build-zig          - Build only the Zig binary (ReleaseSafe)"
	@echo "  make build-zig-small    - Build Zig binary with maximum size optimization"
	@echo "  make check-prereqs      - Check if Go and Zig are installed"
	@echo "  make install-prereqs    - Download and install Go and Zig to /tmp"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make test               - Run tests"
	@echo "  make test-go            - Run Go tests"
	@echo "  make test-zig           - Run Zig tests"
	@echo "  make compare-size       - Compare binary sizes of Go and Zig versions"
	@echo "  make help               - Show this help message"
	@echo ""
	@echo "Prerequisites:"
	@echo "  - Go 1.19 or later"
	@echo "  - Zig 0.11.0 or later"
	@echo ""
	@echo "Examples:"
	@echo "  make check-prereqs      # Check if tools are installed"
	@echo "  make install-prereqs    # Download and install tools"
	@echo "  make build              # Build both binaries"
	@echo "  make compare-size       # Compare binary sizes"
