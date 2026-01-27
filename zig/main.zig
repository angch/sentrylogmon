const std = @import("std");
const http = std.http;

const Args = struct {
    dsn: []const u8,
    file: ?[]const u8 = null,
    use_dmesg: bool = false,
    command: ?[]const u8 = null,
    journalctl: ?[]const u8 = null,
    pattern: []const u8 = "Error",
    environment: []const u8 = "production",
    release: []const u8 = "",
    verbose: bool = false,
    oneshot: bool = false,
};

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    // Parse command-line arguments
    const args = try parseArgs(allocator);
    defer if (args.file) |f| allocator.free(f);
    defer if (args.command) |c| allocator.free(c);
    defer if (args.journalctl) |j| allocator.free(j);
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
        var argv = std.ArrayList([]const u8).init(allocator);
        defer argv.deinit();
        try argv.append("dmesg");
        if (!args.oneshot) try argv.append("-w");

        try monitorCommand(allocator, argv.items, "dmesg", args);
    } else if (args.file) |file_path| {
        if (args.verbose) {
            std.debug.print("Monitoring file: {s}\n", .{file_path});
        }
        try monitorFile(allocator, file_path, args);
    } else if (args.command) |cmd| {
        if (args.verbose) {
            std.debug.print("Monitoring command: {s}\n", .{cmd});
        }
        var argv = std.ArrayList([]const u8).init(allocator);
        defer argv.deinit();
        var iter = std.mem.tokenizeScalar(u8, cmd, ' ');
        while (iter.next()) |part| {
            try argv.append(part);
        }

        try monitorCommand(allocator, argv.items, "command", args);
    } else if (args.journalctl) |jargs| {
        if (args.verbose) {
            std.debug.print("Monitoring journalctl: {s}\n", .{jargs});
        }
        var argv = std.ArrayList([]const u8).init(allocator);
        defer argv.deinit();
        try argv.append("journalctl");
        var iter = std.mem.tokenizeScalar(u8, jargs, ' ');
        while (iter.next()) |part| {
            try argv.append(part);
        }

        try monitorCommand(allocator, argv.items, "journalctl", args);
    } else {
        std.debug.print("Please specify a log source: --dmesg, --file, --command or --journalctl\n", .{});
        std.process.exit(1);
    }
}

fn printUsage() void {
    std.debug.print(
        \\Usage of sentrylogmon-zig:
        \\  --dsn string
        \\        Sentry DSN (or set SENTRY_DSN environment variable)
        \\  --file string
        \\        Path to log file to monitor
        \\  --dmesg
        \\        Monitor dmesg output
        \\  --command string
        \\        Monitor output of a custom command
        \\  --journalctl string
        \\        Monitor journalctl output (args passed to journalctl)
        \\  --pattern string
        \\        Regex pattern to match (default "Error")
        \\  --environment string
        \\        Sentry environment (default "production")
        \\  --release string
        \\        Sentry release
        \\  --verbose
        \\        Enable verbose logging
        \\  --oneshot
        \\        Process existing logs and exit (do not follow)
        \\  --help
        \\        Show help message
        \\
    , .{});
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
        } else if (std.mem.eql(u8, arg, "--command")) {
            if (arg_iter.next()) |cmd| {
                args.command = try allocator.dupe(u8, cmd);
            }
        } else if (std.mem.eql(u8, arg, "--journalctl")) {
            if (arg_iter.next()) |jargs| {
                args.journalctl = try allocator.dupe(u8, jargs);
            }
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
        } else if (std.mem.eql(u8, arg, "--oneshot")) {
            args.oneshot = true;
        } else if (std.mem.eql(u8, arg, "--help") or std.mem.eql(u8, arg, "-h")) {
            printUsage();
            std.process.exit(0);
        }
    }

    return args;
}

fn monitorFile(allocator: std.mem.Allocator, file_path: []const u8, args: Args) !void {
    const file = try std.fs.cwd().openFile(file_path, .{});
    defer file.close();

    // If not oneshot, start from end of file (tail behavior)
    if (!args.oneshot) {
        try file.seekFromEnd(0);
    }

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

    while (true) {
        const line_or_null = try in_stream.readUntilDelimiterOrEof(&line_buf, '\n');

        if (line_or_null) |line| {
            // Check if line matches pattern
            if (containsPattern(line, args.pattern)) {
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
        } else {
            // EOF reached - flush pending groups
            var iter = timestamp_groups.iterator();
            while (iter.next()) |entry| {
                try sendToSentry(allocator, entry.key_ptr.*, entry.value_ptr.items, file_path, args);
                // Clean up sent items
                for (entry.value_ptr.items) |l| {
                    allocator.free(l);
                }
                entry.value_ptr.deinit();
            }
            timestamp_groups.clearAndFree();

            if (args.oneshot) {
                break;
            }

            // Sleep and retry (simple polling)
            std.time.sleep(1 * std.time.ns_per_s);
        }
    }
}

fn monitorCommand(allocator: std.mem.Allocator, argv: []const []const u8, source_name: []const u8, args: Args) !void {
    while (true) {
        if (args.verbose) {
            std.debug.print("Starting command: {s}\n", .{argv[0]});
        }

        var child = std.process.Child.init(argv, allocator);
        child.stdout_behavior = .Pipe;

        child.spawn() catch |err| {
            std.debug.print("Failed to spawn command: {}\n", .{err});
            if (args.oneshot) return err;
            std.time.sleep(1 * std.time.ns_per_s);
            continue;
        };

        {
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

                while (true) {
                    const line_or_null = in_stream.readUntilDelimiterOrEof(&line_buf, '\n') catch break;

                    if (line_or_null) |line| {
                        if (containsPattern(line, args.pattern)) {
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
                    } else {
                        break;
                    }
                }

                // Send grouped events to Sentry
                var iter = timestamp_groups.iterator();
                while (iter.next()) |entry| {
                    try sendToSentry(allocator, entry.key_ptr.*, entry.value_ptr.items, source_name, args);
                }
            }
        }

        _ = child.wait() catch {};

        if (args.oneshot) break;

        if (args.verbose) {
            std.debug.print("Command exited, restarting in 1s...\n", .{});
        }
        std.time.sleep(1 * std.time.ns_per_s);
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

    const auth_header = try std.fmt.allocPrint(allocator, "Sentry sentry_version=7,sentry_key={s}", .{parsed_dsn.public_key});
    defer allocator.free(auth_header);

    const extra_headers = &[_]std.http.Header{
        .{ .name = "X-Sentry-Auth", .value = auth_header },
        .{ .name = "Content-Type", .value = "application/json" },
    };

    var server_header_buffer: [4096]u8 = undefined;

    var request = try client.open(.POST, uri, .{
        .server_header_buffer = &server_header_buffer,
        .extra_headers = extra_headers,
    });
    defer request.deinit();

    request.transfer_encoding = .{ .content_length = payload.items.len };

    try request.send();
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
