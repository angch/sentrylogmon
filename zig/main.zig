const std = @import("std");
const http = std.http;
const config_mod = @import("config.zig");
const queue_mod = @import("queue.zig");
const batcher_mod = @import("batcher.zig");

const Args = struct {
    dsn: []const u8,
    file: ?[]const u8 = null,
    use_dmesg: bool = false,
    command: ?[]const u8 = null,
    journalctl: ?[]const u8 = null,
    pattern: []const u8 = "Error",
    format: ?[]const u8 = null,
    environment: []const u8 = "production",
    release: []const u8 = "",
    verbose: bool = false,
    oneshot: bool = false,
    config: ?[]const u8 = null,
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
    defer if (args.config) |c| allocator.free(c);

    if (args.dsn.len == 0 and args.config == null) {
        std.debug.print("Sentry DSN is required. Set via --dsn flag or SENTRY_DSN environment variable\n", .{});
        std.process.exit(1);
    }

    if (args.verbose) {
        std.debug.print("Initialized Sentry with DSN (environment={s}, release={s})\n", .{ args.environment, args.release });
    }

    // Initialize Queue
    const queue_dir = ".sentrylogmon_queue";
    try queue_mod.Queue.init(queue_dir);
    var queue = queue_mod.Queue{ .dir_path = queue_dir };

    // Try to flush queue
    const RetryContext = struct {
        allocator: std.mem.Allocator,
        args: Args,
        fn send(self: @This(), payload: []u8) !void {
            try sendSentryPayload(self.allocator, payload, self.args);
        }
    };
    const retry_ctx = RetryContext{ .allocator = allocator, .args = args };
    queue.retryAll(allocator, retry_ctx, RetryContext.send) catch |err| {
        if (args.verbose) std.debug.print("Failed to retry queued events: {}\n", .{err});
    };

    var config: ?config_mod.Config = null;
    if (args.config) |cfg_path| {
        config = try config_mod.parseConfig(allocator, cfg_path);
    }
    defer if (config) |*c| c.deinit(allocator);

    if (config) |cfg| {
        // Multi-monitor mode
        var threads = std.ArrayList(std.Thread).empty;
        defer threads.deinit(allocator);

        for (cfg.monitors.items) |monitor| {
            var monitor_args = args;
            if (cfg.sentry.dsn.len > 0) monitor_args.dsn = cfg.sentry.dsn;
            if (cfg.sentry.environment.len > 0) monitor_args.environment = cfg.sentry.environment;
            if (cfg.sentry.release.len > 0) monitor_args.release = cfg.sentry.release;

            if (monitor_args.dsn.len == 0) {
                 std.debug.print("Skipping monitor {s}: No DSN configured\n", .{monitor.name});
                 continue;
            }

            const ctx = try allocator.create(MonitorContext);
            ctx.* = MonitorContext{
                .allocator = allocator,
                .args = monitor_args,
                .source_name = monitor.name,
                .queue = &queue,
            };

            const batcher = try allocator.create(batcher_mod.Batcher);
            batcher.* = batcher_mod.Batcher.init(allocator, ctx, MonitorContext.send);
            try batcher.startFlusher();

            if (args.verbose) {
                std.debug.print("Starting monitor: {s}\n", .{monitor.name});
            }

            if (monitor.type == .file) {
                 if (monitor.path) |p| {
                     const t = try std.Thread.spawn(.{}, monitorFile, .{p, monitor.pattern, batcher, monitor_args});
                     try threads.append(allocator, t);
                 }
            } else if (monitor.type == .journalctl) {
                 var argv = std.ArrayList([]const u8).empty;
                 try argv.append(allocator, "journalctl");
                 if (monitor.args) |a| {
                     var iter = std.mem.tokenizeScalar(u8, a, ' ');
                     while (iter.next()) |part| {
                         try argv.append(allocator, part);
                     }
                 }
                 const t = try std.Thread.spawn(.{}, monitorCommand, .{allocator, argv.items, monitor.pattern, batcher, monitor_args});
                 try threads.append(allocator, t);
            } else if (monitor.type == .dmesg) {
                 var argv = std.ArrayList([]const u8).empty;
                 try argv.append(allocator, "dmesg");
                 if (!args.oneshot) try argv.append(allocator, "-w");
                 const t = try std.Thread.spawn(.{}, monitorCommand, .{allocator, argv.items, monitor.pattern, batcher, monitor_args});
                 try threads.append(allocator, t);
            } else if (monitor.type == .command) {
                 if (monitor.args) |cmd_str| {
                     var argv = std.ArrayList([]const u8).empty;
                     var iter = std.mem.tokenizeScalar(u8, cmd_str, ' ');
                     while (iter.next()) |part| {
                         try argv.append(allocator, part);
                     }
                     const t = try std.Thread.spawn(.{}, monitorCommand, .{allocator, argv.items, monitor.pattern, batcher, monitor_args});
                     try threads.append(allocator, t);
                 }
            }
        }

        for (threads.items) |t| {
            t.join();
        }
        return;
    }

    // Legacy
    {
        // Determine log source (Legacy)

        // Prepare context and batcher
        var source_name: []const u8 = "unknown";
        if (args.use_dmesg) {
            source_name = "dmesg";
        } else if (args.file) |_| {
            source_name = "file";
        } else if (args.command) |_| {
            source_name = "command";
        } else if (args.journalctl) |_| {
            source_name = "journalctl";
        }

        // If no source specified, exit
        if (std.mem.eql(u8, source_name, "unknown") and args.config == null) {
            std.debug.print("Please specify a log source: --dmesg, --file, --command, --journalctl, or --config\n", .{});
            std.process.exit(1);
        }

        const ctx = try allocator.create(MonitorContext);
        ctx.* = MonitorContext{
            .allocator = allocator,
            .args = args,
            .source_name = source_name,
            .queue = &queue,
        };

        const batcher = try allocator.create(batcher_mod.Batcher);
        batcher.* = batcher_mod.Batcher.init(allocator, ctx, MonitorContext.send);
        try batcher.startFlusher();

        if (args.use_dmesg) {
            if (args.verbose) {
                std.debug.print("Starting dmesg monitor...\n", .{});
            }
            var argv = std.ArrayList([]const u8).empty;
            defer argv.deinit(allocator);
            try argv.append(allocator, "dmesg");
            if (!args.oneshot) try argv.append(allocator, "-w");

            try monitorCommand(allocator, argv.items, args.pattern, batcher, args);
        } else if (args.file) |file_path| {
            if (args.verbose) {
                std.debug.print("Monitoring file: {s}\n", .{file_path});
            }
            try monitorFile(file_path, args.pattern, batcher, args);
        } else if (args.command) |cmd| {
            if (args.verbose) {
                std.debug.print("Monitoring command: {s}\n", .{cmd});
            }
            var argv = std.ArrayList([]const u8).empty;
            defer argv.deinit(allocator);
            var iter = std.mem.tokenizeScalar(u8, cmd, ' ');
            while (iter.next()) |part| {
                try argv.append(allocator, part);
            }

            try monitorCommand(allocator, argv.items, args.pattern, batcher, args);
        } else if (args.journalctl) |jargs| {
            if (args.verbose) {
                std.debug.print("Monitoring journalctl: {s}\n", .{jargs});
            }
            var argv = std.ArrayList([]const u8).empty;
            defer argv.deinit(allocator);
            try argv.append(allocator, "journalctl");
            var iter = std.mem.tokenizeScalar(u8, jargs, ' ');
            while (iter.next()) |part| {
                try argv.append(allocator, part);
            }

            try monitorCommand(allocator, argv.items, args.pattern, batcher, args);
        }
    }
}

fn printUsage() void {
    std.debug.print(
        \\Usage of sentrylogmon-zig:
        \\  --config string
        \\        Path to configuration file
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
        \\  --format string
        \\        Log format (nginx, nginx-error, dmesg)
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
        } else if (std.mem.eql(u8, arg, "--format")) {
            if (arg_iter.next()) |format| {
                args.format = format;
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
        } else if (std.mem.eql(u8, arg, "--config")) {
            if (arg_iter.next()) |cfg| {
                args.config = try allocator.dupe(u8, cfg);
            }
        } else if (std.mem.eql(u8, arg, "--help") or std.mem.eql(u8, arg, "-h")) {
            printUsage();
            std.process.exit(0);
        }
    }

    return args;
}

fn readLine(reader: *std.io.Reader, buffer: []u8) !?[]u8 {
    var fbs = std.io.Writer.fixed(buffer);
    _ = reader.streamDelimiter(&fbs, '\n') catch |err| {
        if (err == error.EndOfStream) {
            if (fbs.end > 0) {
                return buffer[0..fbs.end];
            }
            return null;
        }
        return err;
    };
    return buffer[0..fbs.end];
}

fn monitorFile(file_path: []const u8, pattern: []const u8, batcher: *batcher_mod.Batcher, args: Args) !void {
    const file = try std.fs.cwd().openFile(file_path, .{});
    defer file.close();

    // If not oneshot, start from end of file (tail behavior)
    if (!args.oneshot) {
        try file.seekFromEnd(0);
    }

    var reader_buf: [4096]u8 = undefined;
    var file_reader = file.reader(&reader_buf);
    const in_stream = &file_reader.interface;

    var line_buf: [4096]u8 = undefined;

    while (true) {
        const line_or_null = try readLine(in_stream, &line_buf);

        if (line_or_null) |line| {
            // Check if line matches pattern
            if (containsPattern(line, pattern)) {
                if (args.verbose) {
                    std.debug.print("Matched line: {s}\n", .{line});
                }

                const timestamp = extractTimestamp(line);
                try batcher.add(timestamp, line);
            }
        } else {
            if (args.oneshot) {
                batcher.flush() catch {};
                break;
            }

            // Sleep and retry (simple polling)
            std.Thread.sleep(1 * std.time.ns_per_s);
        }
    }
}

fn monitorCommand(allocator: std.mem.Allocator, argv: []const []const u8, pattern: []const u8, batcher: *batcher_mod.Batcher, args: Args) !void {
    while (true) {
        if (args.verbose) {
            std.debug.print("Starting command: {s}\n", .{argv[0]});
        }

        var child = std.process.Child.init(argv, allocator);
        child.stdout_behavior = .Pipe;

        child.spawn() catch |err| {
            std.debug.print("Failed to spawn command: {}\n", .{err});
            if (args.oneshot) return err;
            std.Thread.sleep(1 * std.time.ns_per_s);
            continue;
        };

        {
            defer _ = child.kill() catch {};

            if (child.stdout) |stdout| {
                var reader_buf: [4096]u8 = undefined;
                var stdout_reader = stdout.reader(&reader_buf);
                const in_stream = &stdout_reader.interface;

                var line_buf: [4096]u8 = undefined;

                while (true) {
                    const line_or_null = readLine(in_stream, &line_buf) catch break;

                    if (line_or_null) |line| {
                        if (containsPattern(line, pattern)) {
                            if (args.verbose) {
                                std.debug.print("Matched line: {s}\n", .{line});
                            }

                            const timestamp = extractTimestamp(line);
                            batcher.add(timestamp, line) catch |err| {
                                std.debug.print("Error adding to batch: {}\n", .{err});
                            };
                        }
                    } else {
                        break;
                    }
                }
            }
        }

        _ = child.wait() catch {};

        if (args.oneshot) {
            batcher.flush() catch {};
            break;
        }

        if (args.verbose) {
            std.debug.print("Command exited, restarting in 1s...\n", .{});
        }
        std.Thread.sleep(1 * std.time.ns_per_s);
    }
}

fn shouldLog(line: []const u8, format: ?[]const u8, pattern: []const u8) bool {
    if (format) |fmt| {
        if (std.mem.eql(u8, fmt, "nginx") or std.mem.eql(u8, fmt, "nginx-error")) {
            const patterns = [_][]const u8{ "error", "critical", "crit", "alert", "emerg" };
            return containsAny(line, &patterns);
        } else if (std.mem.eql(u8, fmt, "dmesg")) {
            const patterns = [_][]const u8{ "error", "fail", "panic", "oops", "exception" };
            return containsAny(line, &patterns);
        }
    }
    return containsPattern(line, pattern);
}

fn containsAny(haystack: []const u8, needles: []const []const u8) bool {
    for (needles) |needle| {
        if (containsPattern(haystack, needle)) return true;
    }
    return false;
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

const MonitorContext = struct {
    allocator: std.mem.Allocator,
    args: Args,
    source_name: []const u8,
    queue: *queue_mod.Queue,

    fn send(ctx_opaque: *anyopaque, timestamp: []const u8, lines: []const []const u8) !void {
        const self: *@This() = @ptrCast(@alignCast(ctx_opaque));

        // Build payload
        const payload = try createSentryPayload(self.allocator, timestamp, lines, self.source_name, self.args);
        defer self.allocator.free(payload);

        // Try send
        sendSentryPayload(self.allocator, payload, self.args) catch |err| {
            std.debug.print("Failed to send to Sentry: {}. Queueing...\n", .{err});
            // Queue
            self.queue.push(self.allocator, payload) catch |qerr| {
                std.debug.print("Failed to queue event: {}\n", .{qerr});
            };
        };
    }
};

fn createSentryPayload(allocator: std.mem.Allocator, timestamp: []const u8, lines: []const []const u8, source: []const u8, args: Args) ![]u8 {
    var payload = std.ArrayList(u8).empty;
    errdefer payload.deinit(allocator);

    const writer = payload.writer(allocator);
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
            switch (c) {
                '"' => try writer.writeAll("\\\""),
                '\\' => try writer.writeAll("\\\\"),
                '\n' => try writer.writeAll("\\n"),
                '\r' => try writer.writeAll("\\r"),
                '\t' => try writer.writeAll("\\t"),
                else => {
                    if (std.ascii.isPrint(c)) {
                        try writer.writeByte(c);
                    } else {
                        try writer.print("\\u{x:0>4}", .{c});
                    }
                },
            }
        }
    }

    try writer.writeAll("\"}}}");

    return payload.toOwnedSlice(allocator);
}

fn sendSentryPayload(allocator: std.mem.Allocator, payload: []u8, args: Args) !void {
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

    var request = try client.request(.POST, uri, .{
        .extra_headers = extra_headers,
    });
    defer request.deinit();

    try request.sendBodyComplete(payload);

    // Wait for response
    var redirect_buffer: [1024]u8 = undefined;
    const response = try request.receiveHead(&redirect_buffer);

    if (args.verbose) {
        std.debug.print("Sent event to Sentry: {}\n", .{response.head.status});
    }

    // Read and discard body to allow connection reuse
    var body_buf: [1024]u8 = undefined;
    var mut_response = response;
    const reader = mut_response.reader(&body_buf);
    _ = try reader.discardRemaining();
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

test {
    std.testing.refAllDecls(@This());
    _ = config_mod;
    _ = queue_mod;
    _ = batcher_mod;
}
