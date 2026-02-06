const std = @import("std");
const config_mod = @import("config.zig");

pub const StatusResponse = struct {
    pid: i32,
    start_time: []const u8,
    version: []const u8,
    config: ?config_mod.Config,
    command_line: []const u8 = "",
    memory_alloc: u64 = 0,

    pub fn deinit(self: *StatusResponse, allocator: std.mem.Allocator) void {
        allocator.free(self.start_time);
        allocator.free(self.version);
        if (self.command_line.len > 0) allocator.free(self.command_line);
        if (self.config) |*c| {
            c.deinit(allocator);
        }
    }
};

pub fn ensureSecureDirectory(path: []const u8) !void {
    // Try to create directory
    std.fs.makeDirAbsolute(path) catch |err| {
        if (err != error.PathAlreadyExists) return err;
    };

    // Set permissions to 0700 (rwx------)
    if (@import("builtin").os.tag != .windows) {
        const file = try std.fs.openFileAbsolute(path, .{ .mode = .read_only });
        defer file.close();
        try file.chmod(0o700);
    }
}

pub fn startServer(allocator: std.mem.Allocator, socket_path: []const u8, config: ?config_mod.Config, args_list: []const []const u8, start_time: i64) !void {
    // Unlink existing socket
    std.fs.deleteFileAbsolute(socket_path) catch {};

    const address = try std.net.Address.initUnix(socket_path);
    var server = try address.listen(.{ .kernel_backlog = 1 });
    defer server.deinit();
    // Ensure socket file has correct permissions (0600)
    if (@import("builtin").os.tag != .windows) {
        const socket_file = std.fs.File{ .handle = server.stream.handle };
        try socket_file.chmod(0o600);
    }

    while (true) {
        var connection = server.accept() catch |err| {
            std.debug.print("Accept error: {}\n", .{err});
            continue;
        };
        defer connection.stream.close();

        handleConnection(allocator, connection.stream, config, args_list, start_time) catch |err| {
            std.debug.print("Handle connection error: {}\n", .{err});
        };
    }
}

fn handleConnection(allocator: std.mem.Allocator, stream: std.net.Stream, config: ?config_mod.Config, args_list: []const []const u8, start_time: i64) !void {
    var buf: [4096]u8 = undefined;
    const bytes_read = try stream.read(&buf);
    if (bytes_read == 0) return;

    const request = buf[0..bytes_read];

    // Simple HTTP parsing
    // Look for method and path
    var iter = std.mem.tokenizeScalar(u8, request, ' ');
    const method = iter.next() orelse return;
    const path = iter.next() orelse return;

    if (std.mem.eql(u8, method, "GET") and std.mem.eql(u8, path, "/status")) {
        // Serialize status
        const time_str = try std.fmt.allocPrint(allocator, "{d}", .{start_time});
        defer allocator.free(time_str);

        const command_line = try std.mem.join(allocator, " ", args_list);
        defer allocator.free(command_line);

        const pid: i32 = if (@import("builtin").os.tag == .linux) @intCast(std.os.linux.getpid()) else 0;

        const mem_alloc = readProcessMemory(allocator) catch 0;

        const status = StatusResponse{
            .pid = pid,
            .start_time = time_str,
            .version = if (config) |c| c.sentry.release else "unknown",
            .config = config,
            .command_line = command_line,
            .memory_alloc = mem_alloc,
        };

        var json_buf = std.ArrayList(u8).empty; // Unmanaged
        defer json_buf.deinit(allocator); // Unmanaged

        try json_buf.writer(allocator).print("{f}", .{std.json.fmt(status, .{})});

        const response_headers = "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nConnection: close\r\n\r\n";
        try stream.writeAll(response_headers);
        try stream.writeAll(json_buf.items);

    } else if (std.mem.eql(u8, method, "POST") and std.mem.eql(u8, path, "/update")) {
        const response = "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\nRestarting...";
        try stream.writeAll(response);

        std.Thread.sleep(100 * std.time.ns_per_ms);

        return std.process.execv(allocator, args_list);
    } else {
        const response = "HTTP/1.1 404 Not Found\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\nNot Found";
        try stream.writeAll(response);
    }
}

pub fn listInstances(allocator: std.mem.Allocator, socket_dir: []const u8) !std.ArrayList(StatusResponse) {
    var list = std.ArrayList(StatusResponse).empty;
    errdefer list.deinit(allocator);

    var dir = std.fs.openDirAbsolute(socket_dir, .{ .iterate = true }) catch |err| {
        if (err == error.FileNotFound) return list;
        return err;
    };
    defer dir.close();
    var iter = dir.iterate();
    while (try iter.next()) |entry| {
        if (entry.kind != .unix_domain_socket and entry.kind != .file) continue;
        if (!std.mem.endsWith(u8, entry.name, ".sock")) continue;

        const socket_path = try std.fs.path.join(allocator, &[_][]const u8{ socket_dir, entry.name });
        defer allocator.free(socket_path);

        const status = getStatus(allocator, socket_path) catch |err| {
             // Maybe stale socket
             std.debug.print("Failed to get status from {s}: {}\n", .{entry.name, err});
             continue;
        };
        try list.append(allocator, status);
    }
    return list;
}

fn getStatus(allocator: std.mem.Allocator, socket_path: []const u8) !StatusResponse {
    const stream = try std.net.connectUnixSocket(socket_path);
    defer stream.close();

    const request = "GET /status HTTP/1.0\r\n\r\n";
    try stream.writeAll(request);

    var buf: [4096]u8 = undefined;
    var total_read: usize = 0;
    while (true) {
        const n = try stream.read(buf[total_read..]);
        if (n == 0) break;
        total_read += n;
        if (total_read == buf.len) break;
    }

    const response = buf[0..total_read];

    if (std.mem.indexOf(u8, response, "\r\n\r\n")) |body_start_idx| {
        const body = response[body_start_idx + 4 ..];

        const parsed = try std.json.parseFromSlice(StatusResponse, allocator, body, .{ .ignore_unknown_fields = true });
        defer parsed.deinit();

        return deepCopyStatus(allocator, parsed.value);
    }

    return error.InvalidResponse;
}

fn readProcessMemory(allocator: std.mem.Allocator) !u64 {
    const file = try std.fs.openFileAbsolute("/proc/self/statm", .{});
    defer file.close();
    const content = try file.readToEndAlloc(allocator, 1024);
    defer allocator.free(content);

    var iter = std.mem.tokenizeScalar(u8, content, ' ');
    _ = iter.next(); // size
    const resident_str = iter.next() orelse return error.InvalidFormat;
    const resident_pages = try std.fmt.parseInt(u64, resident_str, 10);
    const page_size = 4096; // Assume 4KB for now
    return resident_pages * page_size;
}

fn deepCopyStatus(allocator: std.mem.Allocator, src: StatusResponse) !StatusResponse {
    var dst = src;
    dst.start_time = try allocator.dupe(u8, src.start_time);
    dst.version = try allocator.dupe(u8, src.version);
    dst.command_line = try allocator.dupe(u8, src.command_line);
    dst.memory_alloc = src.memory_alloc;
    if (src.config) |c| {
        dst.config = try deepCopyConfig(allocator, c);
    }
    return dst;
}

fn deepCopyConfig(allocator: std.mem.Allocator, src: config_mod.Config) !config_mod.Config {
    var dst = src;
    dst.sentry.dsn = try allocator.dupe(u8, src.sentry.dsn);
    dst.sentry.environment = try allocator.dupe(u8, src.sentry.environment);
    dst.sentry.release = try allocator.dupe(u8, src.sentry.release);

    var monitors = std.ArrayList(config_mod.MonitorConfig).empty;
    for (src.monitors.items) |m| {
        var new_m = m;
        new_m.name = try allocator.dupe(u8, m.name);
        new_m.pattern = try allocator.dupe(u8, m.pattern);
        if (m.path) |p| new_m.path = try allocator.dupe(u8, p);
        if (m.args) |a| new_m.args = try allocator.dupe(u8, a);
        if (m.exclude_pattern) |p| new_m.exclude_pattern = try allocator.dupe(u8, p);
        if (m.format) |f| new_m.format = try allocator.dupe(u8, f);
        if (m.rate_limit_window) |w| new_m.rate_limit_window = try allocator.dupe(u8, w);
        try monitors.append(allocator, new_m);
    }
    dst.monitors = monitors;
    return dst;
}

pub fn requestUpdate(allocator: std.mem.Allocator, socket_path: []const u8) !void {
    _ = allocator;
    const stream = try std.net.connectUnixSocket(socket_path);
    defer stream.close();

    const request = "POST /update HTTP/1.0\r\n\r\n";
    try stream.writeAll(request);

    var buf: [128]u8 = undefined;
    _ = try stream.read(&buf);
}

test "IPC server and client" {
    const allocator = std.testing.allocator;
    const socket_path = "/tmp/test_sentrylogmon.sock";

    const args = &[_][]const u8{"program", "arg1"};

    const server_thread = try std.Thread.spawn(.{}, startServer, .{allocator, socket_path, null, args, 1234567890});
    server_thread.detach();

    std.Thread.sleep(100 * std.time.ns_per_ms);

    var status = try getStatus(allocator, socket_path);
    defer status.deinit(allocator);

    const expected_pid = if (@import("builtin").os.tag == .linux) @as(i32, @intCast(std.os.linux.getpid())) else 0;
    try std.testing.expectEqual(expected_pid, status.pid);
    try std.testing.expectEqualStrings("1234567890", status.start_time);
}
