const std = @import("std");
const http = std.http;

const Args = struct {
    dsn: []const u8,
    file: ?[]const u8 = null,
    use_dmesg: bool = false,
    pattern: []const u8 = "Error",
    environment: []const u8 = "production",
    release: []const u8 = "",
    verbose: bool = false,
};

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    // Parse command-line arguments
    var args = try parseArgs(allocator);
    defer if (args.file) |f| allocator.free(f);
    defer if (args.release.len > 0) allocator.free(args.release);

    if (args.dsn.len == 0) {
        std.debug.print("Sentry DSN is required. Set via --dsn flag or SENTRY_DSN environment variable\n", .{});
        std.process.exit(1);
    }

    if (args.verbose) {
        std.debug.print("Initialized Sentry with DSN (environment={s}, release={s})\n", .{ args.environment, args.release });
    }

    // Determine log source
    if (args.use_dmesg) {
        if (args.verbose) {
            std.debug.print("Starting dmesg monitor...\n", .{});
        }
        try monitorDmesg(allocator, args);
    } else if (args.file) |file_path| {
        if (args.verbose) {
            std.debug.print("Monitoring file: {s}\n", .{file_path});
        }
        try monitorFile(allocator, file_path, args);
    } else {
        std.debug.print("Please specify a log source: --dmesg or --file\n", .{});
        std.process.exit(1);
    }
}

fn parseArgs(allocator: std.mem.Allocator) !Args {
    var args = Args{
        .dsn = "",
    };

    // Try to get DSN from environment
    const env_dsn = std.process.getEnvVarOwned(allocator, "SENTRY_DSN") catch "";
    if (env_dsn.len > 0) {
        args.dsn = env_dsn;
    }

    var arg_iter = try std.process.argsWithAllocator(allocator);
    defer arg_iter.deinit();

    // Skip program name
    _ = arg_iter.skip();

    while (arg_iter.next()) |arg| {
        if (std.mem.eql(u8, arg, "--dsn")) {
            if (arg_iter.next()) |dsn| {
                args.dsn = dsn;
            }
        } else if (std.mem.eql(u8, arg, "--file")) {
            if (arg_iter.next()) |file| {
                args.file = try allocator.dupe(u8, file);
            }
        } else if (std.mem.eql(u8, arg, "--dmesg")) {
            args.use_dmesg = true;
        } else if (std.mem.eql(u8, arg, "--pattern")) {
            if (arg_iter.next()) |pattern| {
                args.pattern = pattern;
            }
        } else if (std.mem.eql(u8, arg, "--environment")) {
            if (arg_iter.next()) |env| {
                args.environment = env;
            }
        } else if (std.mem.eql(u8, arg, "--release")) {
            if (arg_iter.next()) |release| {
                args.release = try allocator.dupe(u8, release);
            }
        } else if (std.mem.eql(u8, arg, "--verbose")) {
            args.verbose = true;
        }
    }

    return args;
}

fn monitorFile(allocator: std.mem.Allocator, file_path: []const u8, args: Args) !void {
    const file = try std.fs.cwd().openFile(file_path, .{});
    defer file.close();

    var buf_reader = std.io.bufferedReader(file.reader());
    var in_stream = buf_reader.reader();

    var line_buf: [4096]u8 = undefined;
    var timestamp_groups = std.StringHashMap(std.ArrayList([]const u8)).init(allocator);
    defer {
        var iter = timestamp_groups.iterator();
        while (iter.next()) |entry| {
            for (entry.value_ptr.items) |line| {
                allocator.free(line);
            }
            entry.value_ptr.deinit();
        }
        timestamp_groups.deinit();
    }

    while (try in_stream.readUntilDelimiterOrEof(&line_buf, '\n')) |line| {
        // Check if line matches pattern
        if (!containsPattern(line, args.pattern)) {
            continue;
        }

        if (args.verbose) {
            std.debug.print("Matched line: {s}\n", .{line});
        }

        // Extract timestamp (simplified - just looking for [number])
        const timestamp = extractTimestamp(line);
        const timestamp_key = try allocator.dupe(u8, timestamp);

        const result = try timestamp_groups.getOrPut(timestamp_key);
        if (!result.found_existing) {
            result.value_ptr.* = std.ArrayList([]const u8).init(allocator);
        } else {
            allocator.free(timestamp_key);
        }

        const line_copy = try allocator.dupe(u8, line);
        try result.value_ptr.append(line_copy);
    }

    // Send grouped events to Sentry
    var iter = timestamp_groups.iterator();
    while (iter.next()) |entry| {
        try sendToSentry(allocator, entry.key_ptr.*, entry.value_ptr.items, file_path, args);
    }
}

fn monitorDmesg(allocator: std.mem.Allocator, args: Args) !void {
    var child = std.process.Child.init(&[_][]const u8{ "dmesg", "-w" }, allocator);
    child.stdout_behavior = .Pipe;
    try child.spawn();
    defer _ = child.kill() catch {};

    if (child.stdout) |stdout| {
        var buf_reader = std.io.bufferedReader(stdout.reader());
        var in_stream = buf_reader.reader();

        var line_buf: [4096]u8 = undefined;
        var timestamp_groups = std.StringHashMap(std.ArrayList([]const u8)).init(allocator);
        defer {
            var iter = timestamp_groups.iterator();
            while (iter.next()) |entry| {
                for (entry.value_ptr.items) |line| {
                    allocator.free(line);
                }
                entry.value_ptr.deinit();
            }
            timestamp_groups.deinit();
        }

        while (try in_stream.readUntilDelimiterOrEof(&line_buf, '\n')) |line| {
            if (!containsPattern(line, args.pattern)) {
                continue;
            }

            if (args.verbose) {
                std.debug.print("Matched line: {s}\n", .{line});
            }

            const timestamp = extractTimestamp(line);
            const timestamp_key = try allocator.dupe(u8, timestamp);

            const result = try timestamp_groups.getOrPut(timestamp_key);
            if (!result.found_existing) {
                result.value_ptr.* = std.ArrayList([]const u8).init(allocator);
            } else {
                allocator.free(timestamp_key);
            }

            const line_copy = try allocator.dupe(u8, line);
            try result.value_ptr.append(line_copy);
        }

        // Send grouped events to Sentry
        var iter = timestamp_groups.iterator();
        while (iter.next()) |entry| {
            try sendToSentry(allocator, entry.key_ptr.*, entry.value_ptr.items, "dmesg", args);
        }
    }
}

fn containsPattern(haystack: []const u8, needle: []const u8) bool {
    // Simple case-insensitive substring match (simplified from regex)
    var i: usize = 0;
    while (i + needle.len <= haystack.len) : (i += 1) {
        var match = true;
        for (needle, 0..) |c, j| {
            const h = haystack[i + j];
            const n = c;
            if (std.ascii.toLower(h) != std.ascii.toLower(n)) {
                match = false;
                break;
            }
        }
        if (match) return true;
    }
    return false;
}

fn extractTimestamp(line: []const u8) []const u8 {
    // Look for pattern like [123.456]
    if (std.mem.indexOf(u8, line, "[")) |start| {
        if (std.mem.indexOf(u8, line[start..], "]")) |end| {
            return line[start + 1 .. start + end];
        }
    }
    return "unknown";
}

fn sendToSentry(allocator: std.mem.Allocator, timestamp: []const u8, lines: []const []const u8, source: []const u8, args: Args) !void {
    if (args.verbose) {
        std.debug.print("Sending to Sentry: {} line(s) for timestamp {s}\n", .{ lines.len, timestamp });
    }

    // Construct Sentry event payload
    var payload = std.ArrayList(u8).init(allocator);
    defer payload.deinit();

    const writer = payload.writer();
    try writer.writeAll("{\"message\":\"Log errors at timestamp [");
    try writer.writeAll(timestamp);
    try writer.writeAll("]\",\"level\":\"error\",\"environment\":\"");
    try writer.writeAll(args.environment);
    try writer.writeAll("\",\"tags\":{\"timestamp\":\"");
    try writer.writeAll(timestamp);
    try writer.writeAll("\",\"source\":\"");
    try writer.writeAll(source);
    try writer.writeAll("\"},\"extra\":{\"log_lines\":{\"timestamp\":\"");
    try writer.writeAll(timestamp);
    try writer.writeAll("\",\"line_count\":");
    try writer.print("{}", .{lines.len});
    try writer.writeAll(",\"lines\":\"");

    // Escape and add lines
    for (lines, 0..) |line, i| {
        if (i > 0) try writer.writeAll("\\n");
        for (line) |c| {
            if (c == '"' or c == '\\') try writer.writeByte('\\');
            try writer.writeByte(c);
        }
    }

    try writer.writeAll("\"}}}");

    // Parse DSN to extract project info
    const parsed_dsn = try parseDsn(allocator, args.dsn);
    defer allocator.free(parsed_dsn.host);
    defer allocator.free(parsed_dsn.project_id);
    defer allocator.free(parsed_dsn.public_key);

    // Send HTTP POST to Sentry
    var client = http.Client{ .allocator = allocator };
    defer client.deinit();

    const url_buf = try std.fmt.allocPrint(allocator, "https://{s}/api/{s}/store/", .{ parsed_dsn.host, parsed_dsn.project_id });
    defer allocator.free(url_buf);

    const uri = try std.Uri.parse(url_buf);

    var headers = std.http.Headers{ .allocator = allocator };
    defer headers.deinit();

    const auth_header = try std.fmt.allocPrint(allocator, "Sentry sentry_version=7,sentry_key={s}", .{parsed_dsn.public_key});
    defer allocator.free(auth_header);

    try headers.append("X-Sentry-Auth", auth_header);
    try headers.append("Content-Type", "application/json");

    var request = try client.open(.POST, uri, headers, .{});
    defer request.deinit();

    request.transfer_encoding = .{ .content_length = payload.items.len };

    try request.send(.{});
    try request.writeAll(payload.items);
    try request.finish();

    // Wait for response
    try request.wait();

    if (args.verbose) {
        std.debug.print("Sent event to Sentry: {}\n", .{request.response.status});
    }
}

const ParsedDsn = struct {
    host: []const u8,
    project_id: []const u8,
    public_key: []const u8,
};

fn parseDsn(allocator: std.mem.Allocator, dsn: []const u8) !ParsedDsn {
    // DSN format: https://public_key@host/project_id
    const uri = try std.Uri.parse(dsn);

    const host = try allocator.dupe(u8, uri.host.?.percent_encoded);
    
    // Extract public key from userinfo
    const public_key = if (uri.user) |user|
        try allocator.dupe(u8, user.percent_encoded)
    else
        try allocator.dupe(u8, "");

    // Extract project_id from path
    const path = uri.path.percent_encoded;
    var project_id: []const u8 = "";
    if (std.mem.lastIndexOf(u8, path, "/")) |last_slash| {
        project_id = path[last_slash + 1 ..];
    }
    const project_id_copy = try allocator.dupe(u8, project_id);

    return ParsedDsn{
        .host = host,
        .project_id = project_id_copy,
        .public_key = public_key,
    };
}
