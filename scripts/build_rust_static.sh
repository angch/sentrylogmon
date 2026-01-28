#!/bin/bash
set -e

# Ensure .tools directory exists
mkdir -p .tools

# Add .tools/zig to PATH if it exists
if [ -d ".tools/zig" ]; then
    export PATH="$(pwd)/.tools/zig:$PATH"
fi

# Check if zig is available
if ! command -v zig &> /dev/null; then
    echo "Error: zig is required for static rust build but not found."
    echo "Run 'make install-prereqs' to install it."
    exit 1
fi

# Create zig-cc wrapper
cat > .tools/zig-cc << 'EOF'
#!/bin/bash
args=()
for arg in "$@"; do
  if [[ "$arg" == "--target=x86_64-unknown-linux-musl" ]]; then
    continue
  fi
  args+=("$arg")
done
exec zig cc -target x86_64-linux-musl "${args[@]}"
EOF
chmod +x .tools/zig-cc

# Create zig-c++ wrapper
cat > .tools/zig-c++ << 'EOF'
#!/bin/bash
args=()
for arg in "$@"; do
  if [[ "$arg" == "--target=x86_64-unknown-linux-musl" ]]; then
    continue
  fi
  args+=("$arg")
done
exec zig c++ -target x86_64-linux-musl "${args[@]}"
EOF
chmod +x .tools/zig-c++

# Set environment variables for cargo
export PATH="$(pwd)/.tools:$PATH"
export CC_x86_64_unknown_linux_musl="$(pwd)/.tools/zig-cc"
export CXX_x86_64_unknown_linux_musl="$(pwd)/.tools/zig-c++"
export CARGO_TARGET_X86_64_UNKNOWN_LINUX_MUSL_LINKER="$(pwd)/.tools/zig-cc"
export AR_x86_64_unknown_linux_musl="zig ar"
export RUSTFLAGS="-C link-self-contained=no"

echo "Building Rust binary (static)..."
cd rust
cargo build --release --target x86_64-unknown-linux-musl
