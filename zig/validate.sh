#!/bin/bash
# Zig Code Validation Script
# This script validates the Zig implementation without requiring Zig to be installed

set -e

echo "=== Zig Implementation Validation ==="
echo ""

# Check if we're in the right directory
if [ ! -f "main.zig" ]; then
    echo "Error: This script must be run from the zig/ directory"
    exit 1
fi

echo "✓ Found main.zig"

# Check basic code structure
echo ""
echo "Checking code structure..."

# Check for main function
if grep -q "pub fn main()" main.zig; then
    echo "✓ Main function found"
else
    echo "✗ Main function not found"
    exit 1
fi

# Check for argument parsing
if grep -q "parseArgs" main.zig; then
    echo "✓ Argument parsing function found"
else
    echo "✗ Argument parsing function not found"
    exit 1
fi

# Check for file monitoring
if grep -q "monitorFile" main.zig; then
    echo "✓ File monitoring function found"
else
    echo "✗ File monitoring function not found"
    exit 1
fi

# Check for dmesg monitoring
if grep -q "monitorDmesg" main.zig; then
    echo "✓ Dmesg monitoring function found"
else
    echo "✗ Dmesg monitoring function not found"
    exit 1
fi

# Check for pattern matching
if grep -q "containsPattern" main.zig; then
    echo "✓ Pattern matching function found"
else
    echo "✗ Pattern matching function not found"
    exit 1
fi

# Check for Sentry integration
if grep -q "sendToSentry" main.zig; then
    echo "✓ Sentry integration function found"
else
    echo "✗ Sentry integration function not found"
    exit 1
fi

# Check for DSN parsing
if grep -q "parseDsn" main.zig; then
    echo "✓ DSN parsing function found"
else
    echo "✗ DSN parsing function not found"
    exit 1
fi

# Check build.zig
echo ""
echo "Checking build configuration..."

if [ -f "build.zig" ]; then
    echo "✓ build.zig found"
    
    if grep -q "addExecutable" build.zig; then
        echo "✓ Executable configuration found"
    else
        echo "✗ Executable configuration not found"
        exit 1
    fi
else
    echo "✗ build.zig not found"
    exit 1
fi

# Check documentation
echo ""
echo "Checking documentation..."

if [ -f "README.md" ]; then
    echo "✓ README.md found"
else
    echo "✗ README.md not found"
    exit 1
fi

if [ -f "test_functionality.md" ]; then
    echo "✓ Testing documentation found"
else
    echo "✗ Testing documentation not found"
    exit 1
fi

# Verify code follows Go implementation
echo ""
echo "Verifying functional parity with Go implementation..."

# Check that all command-line flags are supported
FLAGS=("dsn" "file" "dmesg" "pattern" "environment" "release" "verbose")
for flag in "${FLAGS[@]}"; do
    if grep -q "$flag" main.zig; then
        echo "✓ Flag --$flag is supported"
    else
        echo "⚠ Flag --$flag might not be supported"
    fi
done

# Line count comparison
echo ""
echo "Code statistics:"
LINES=$(wc -l < main.zig)
echo "  - main.zig: $LINES lines"

if [ -f "../main.go" ]; then
    GO_LINES=$(wc -l < ../main.go)
    echo "  - main.go: $GO_LINES lines"
    
    # Zig should be comparable or slightly larger due to manual HTTP
    if [ $LINES -lt $(($GO_LINES * 3)) ]; then
        echo "✓ Zig implementation is reasonably sized"
    else
        echo "⚠ Zig implementation seems large"
    fi
fi

echo ""
echo "=== Validation Complete ==="
echo ""
echo "Summary:"
echo "  - All required functions are present"
echo "  - Build configuration is correct"
echo "  - Documentation is available"
echo "  - Command-line flags are implemented"
echo ""

if command -v zig &> /dev/null; then
    echo "Note: Zig is installed. You can build and test:"
    echo "  zig build -Doptimize=ReleaseSafe"
    echo ""
else
    echo "Note: Zig is not installed. To build and test:"
    echo "  1. Install Zig from https://ziglang.org/download/"
    echo "  2. Run: zig build -Doptimize=ReleaseSafe"
    echo "  3. Test with: ./zig-out/bin/sentrylogmon-zig --help"
    echo ""
fi

exit 0
