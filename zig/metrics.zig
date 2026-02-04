const std = @import("std");
const sysstat = @import("sysstat.zig");

pub fn startServer(allocator: std.mem.Allocator, port: u16, collector: *sysstat.Collector, cmdline: []const []const u8) !void {
    const address = std.net.Address.initIp4(.{ 0, 0, 0, 0 }, port);
    var server = try address.listen(.{ .kernel_backlog = 128 });
    defer server.deinit();

    std.debug.print("Metrics server listening on port {d}\n", .{port});

    while (true) {
        var connection = server.accept() catch |err| {
            std.debug.print("Metrics accept error: {}\n", .{err});
            continue;
        };
        defer connection.stream.close();

        handleConnection(allocator, connection.stream, collector, cmdline) catch |err| {
            std.debug.print("Metrics handle error: {}\n", .{err});
        };
    }
}

fn handleConnection(allocator: std.mem.Allocator, stream: std.net.Stream, collector: *sysstat.Collector, cmdline: []const []const u8) !void {
    var buf: [1024]u8 = undefined;
    const bytes_read = try stream.read(&buf);
    if (bytes_read == 0) return;

    const request = buf[0..bytes_read];

    var iter = std.mem.tokenizeScalar(u8, request, ' ');
    const method = iter.next() orelse return;
    const full_path = iter.next() orelse return;

    // Strip query params for routing
    const path_end = std.mem.indexOfScalar(u8, full_path, '?') orelse full_path.len;
    const path = full_path[0..path_end];

    if (std.mem.eql(u8, method, "GET")) {
        if (std.mem.eql(u8, path, "/healthz")) {
            try stream.writeAll("HTTP/1.1 200 OK\r\nConnection: close\r\n\r\n");
            return;
        } else if (std.mem.eql(u8, path, "/metrics")) {
            const response = try generateMetrics(allocator, collector);
            defer allocator.free(response);

            try stream.writeAll("HTTP/1.1 200 OK\r\nContent-Type: text/plain; version=0.0.4\r\nConnection: close\r\n\r\n");
            try stream.writeAll(response);
            return;
        } else if (std.mem.startsWith(u8, path, "/debug/pprof")) {
             try handlePprof(stream, path, cmdline);
             return;
        }
    }

    try stream.writeAll("HTTP/1.1 404 Not Found\r\nConnection: close\r\n\r\n");
}

fn handlePprof(stream: std.net.Stream, path: []const u8, cmdline: []const []const u8) !void {
    if (std.mem.eql(u8, path, "/debug/pprof")) {
        try stream.writeAll("HTTP/1.1 301 Moved Permanently\r\nLocation: /debug/pprof/\r\nConnection: close\r\n\r\n");
        return;
    }

    if (std.mem.eql(u8, path, "/debug/pprof/")) {
        const index_html =
            \\<html>
            \\<head>
            \\<title>/debug/pprof/</title>
            \\</head>
            \\<body>
            \\/debug/pprof/<br>
            \\<br>
            \\<a href="cmdline">cmdline</a><br>
            \\<a href="profile">profile</a><br>
            \\<a href="symbol">symbol</a><br>
            \\<a href="trace">trace</a><br>
            \\</body>
            \\</html>
        ;
        try stream.writeAll("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nConnection: close\r\n\r\n");
        try stream.writeAll(index_html);
        return;
    }

    if (std.mem.eql(u8, path, "/debug/pprof/cmdline")) {
         try stream.writeAll("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\n");
         for (cmdline, 0..) |arg, i| {
             if (i > 0) try stream.writeAll("\x00");
             try stream.writeAll(arg);
         }
         return;
    }

    // For other endpoints, return 501
    const not_implemented = "Not Implemented. Zig implementation does not support runtime profiling via HTTP. Please use perf, Valgrind, or Massif.\n";
    try stream.writeAll("HTTP/1.1 501 Not Implemented\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\n");
    try stream.writeAll(not_implemented);
}

fn generateMetrics(allocator: std.mem.Allocator, collector: *sysstat.Collector) ![]u8 {
    _ = collector; // Potentially used for app-specific counters in future
    var list = std.ArrayList(u8).empty;
    errdefer list.deinit(allocator);

    const writer = list.writer(allocator);

    try writer.writeAll("# HELP sentrylogmon_build_info Build information\n");
    try writer.writeAll("# TYPE sentrylogmon_build_info gauge\n");
    // Hardcoded version matching main.zig or TODO
    try writer.writeAll("sentrylogmon_build_info{version=\"0.1.0\"} 1\n");

    if (readSelfStat(allocator)) |stats| {
        try writer.writeAll("# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.\n");
        try writer.writeAll("# TYPE process_cpu_seconds_total counter\n");
        try writer.print("process_cpu_seconds_total {d:.2}\n", .{stats.cpu_seconds});

        try writer.writeAll("# HELP process_resident_memory_bytes Resident memory size in bytes.\n");
        try writer.writeAll("# TYPE process_resident_memory_bytes gauge\n");
        try writer.print("process_resident_memory_bytes {d}\n", .{stats.rss_bytes});

        try writer.writeAll("# HELP process_virtual_memory_bytes Virtual memory size in bytes.\n");
        try writer.writeAll("# TYPE process_virtual_memory_bytes gauge\n");
        try writer.print("process_virtual_memory_bytes {d}\n", .{stats.vsz_bytes});
    } else |_| {}

    return list.toOwnedSlice(allocator);
}

const SelfStat = struct {
    cpu_seconds: f64,
    rss_bytes: u64,
    vsz_bytes: u64,
};

fn readSelfStat(allocator: std.mem.Allocator) !SelfStat {
    const file = try std.fs.openFileAbsolute("/proc/self/stat", .{});
    defer file.close();
    const content = try file.readToEndAlloc(allocator, 1024);
    defer allocator.free(content);

    const closing_paren = std.mem.lastIndexOf(u8, content, ")") orelse return error.InvalidFormat;
    const rest = content[closing_paren + 2 ..];

    var iter = std.mem.tokenizeScalar(u8, rest, ' ');
    // Skip 3..13 (11 fields)
    for (0..11) |_| _ = iter.next();

    const utime_str = iter.next() orelse return error.InvalidFormat;
    const stime_str = iter.next() orelse return error.InvalidFormat;

    // Skip 16..22 (7 fields)
    for (0..7) |_| _ = iter.next();

    const vsz_str = iter.next() orelse return error.InvalidFormat;
    const rss_str = iter.next() orelse return error.InvalidFormat;

    const utime = try std.fmt.parseFloat(f64, utime_str);
    const stime = try std.fmt.parseFloat(f64, stime_str);
    const vsz = try std.fmt.parseInt(u64, vsz_str, 10);
    const rss_pages = try std.fmt.parseInt(u64, rss_str, 10);

    const clk_tck = 100.0;
    const page_size = 4096;

    return SelfStat{
        .cpu_seconds = (utime + stime) / clk_tck,
        .rss_bytes = rss_pages * page_size,
        .vsz_bytes = vsz,
    };
}
