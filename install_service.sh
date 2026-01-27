#!/bin/bash
set -e

if [ "$EUID" -ne 0 ]; then
  echo "Please run as root"
  exit 1
fi

echo "Installing sentrylogmon..."

# Build the binary
echo "Building binary..."
if command -v go >/dev/null 2>&1; then
    go build -o sentrylogmon .
else
    echo "Go not found. Assuming 'sentrylogmon' binary exists in current directory."
    if [ ! -f sentrylogmon ]; then
        echo "Error: sentrylogmon binary not found."
        exit 1
    fi
fi

# Install binary
echo "Copying binary to /usr/local/bin/"
cp sentrylogmon /usr/local/bin/
chmod +x /usr/local/bin/sentrylogmon

# Install config example if not exists
if [ ! -f /etc/sentrylogmon.yaml ]; then
    echo "Creating default configuration at /etc/sentrylogmon.yaml"
    mkdir -p /etc
    cp examples/config.yaml /etc/sentrylogmon.yaml
    echo "IMPORTANT: Please edit /etc/sentrylogmon.yaml with your Sentry DSN."
fi

# Install service
echo "Installing systemd service..."
cp sentrylogmon.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable sentrylogmon

echo "Installation complete."
echo "1. Edit config: sudo nano /etc/sentrylogmon.yaml"
echo "2. Start service: sudo systemctl start sentrylogmon"
echo "3. Check status: sudo systemctl status sentrylogmon"
