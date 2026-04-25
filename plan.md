1. **Add `get_socket_dir` function to Rust `ipc.rs`**
   - Add a public function `get_socket_dir` that conditionally checks the OS. If it is unix-based, it returns `/tmp/sentrylogmon-<uid>` using `libc::getuid()`. Otherwise, it defaults to `/tmp/sentrylogmon`.

2. **Update Rust `main.rs` to use `get_socket_dir`**
   - Replace the 3 hardcoded `let socket_dir = PathBuf::from("/tmp/sentrylogmon");` lines with `let socket_dir = ipc::get_socket_dir();` and pass it to functions correctly.

3. **Add `getSocketDir` function to Zig `ipc.zig`**
   - Add a public function `getSocketDir` that takes an allocator. If it is unix-based, it returns a formatted string `/tmp/sentrylogmon-{d}` using `std.posix.getuid()`. Otherwise, it returns `/tmp/sentrylogmon`.

4. **Update Zig `main.zig` to use `getSocketDir`**
   - Replace the hardcoded `const socket_dir = "/tmp/sentrylogmon";` with `const socket_dir = try ipc.getSocketDir(allocator);` and `defer allocator.free(socket_dir);`.

5. **Run tests to verify the fix**
   - Run tests in both Rust and Zig to ensure everything builds and passes.

6. **Pre-commit and submit**
   - Run the pre-commit instructions to make sure everything passes.
   - Submit the changes as PR 🛡️ Sentinel: [CRITICAL/HIGH] Fix Local Denial of Service via Hardcoded IPC Path.
