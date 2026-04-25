#!/bin/bash
echo "Looking for socket_dir occurrences in rust and zig files..."
grep -r "/tmp/sentrylogmon" rust/ zig/
