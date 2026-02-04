const std = @import("std");
const http = std.http;
const config_mod = @import("config.zig");
const queue_mod = @import("queue.zig");
const batcher_mod = @import("batcher.zig");
const detectors = @import("detectors.zig");
const sysstat = @import("sysstat.zig");
const ipc = @import("ipc.zig");
const syslog = @import("syslog.zig");
const utils = @import("utils.zig");

const RateLimiter = struct {
    limit: usize,
    window: u64, // nanoseconds
    count: usize,
    window_start: i64,

    fn allow(self: *RateLimiter) bool {
        if (self.limit == 0) return true;

        const now = std.time.milliTimestamp();
        // Convert window from ns to ms
        const window_ms = @divTrunc(self.window, std.time.ns_per_ms);

        if (now - self.window_start > window_ms) {
            self.window_start = now;
            self.count = 0;
        }

        if (self.count < self.limit) {
            self.count += 1;
            return true;
        }
        return false;
    }
};

fn formatDuration(allocator: std.mem.Allocator, seconds: i64) ![]u8 {
    if (seconds < 0) return allocator.dupe(u8, "0s");

    const days = @divTrunc(seconds, 86400);
    const rem_days = @rem(seconds, 86400);
    const hours = @divTrunc(rem_days, 3600);
    const rem_hours = @rem(rem_days, 3600);
    const minutes = @divTrunc(rem_hours, 60);
    const sec = @rem(rem_hours, 60);

    if (days > 0) {
        return std.fmt.allocPrint(allocator, "{d}d {d}h {d}m", .{days, hours, minutes});
    } else if (hours > 0) {
        return std.fmt.allocPrint(allocator, "{d}h {d}m {d}s", .{hours, minutes, sec});
    } else if (minutes > 0) {
        return std.fmt.allocPrint(allocator, "{d}m {d}s", .{minutes, sec});
    } else {
        return std.fmt.allocPrint(allocator, "{d}s", .{sec});
    }
}

fn formatDate(allocator: std.mem.Allocator, timestamp: i64) ![]u8 {
    const epoch_seconds = std.time.epoch.EpochSeconds{ .secs = @intCast(timestamp) };
    const year_day = epoch_seconds.getEpochDay();
    const day_seconds = epoch_seconds.getDaySeconds();

    const year = year_day.calculateYearDay();
    const month_day = year.calculateMonthDay();

    const hours = day_seconds.getHoursIntoDay();
    const minutes = day_seconds.getMinutesIntoHour();
    const seconds = day_seconds.getSecondsIntoMinute();

    return std.fmt.allocPrint(allocator, "{d:0>4}-{d:0>2}-{d:0>2} {d:0>2}:{d:0>2}:{d:0>2}", .{
        year.year, month_day.month.numeric(), month_day.day_index + 1,
        hours, minutes, seconds
    });
}

fn getDetails(allocator: std.mem.Allocator, inst: ipc.StatusResponse) ![]u8 {
    if (inst.config) |cfg| {
        var details = std.ArrayList(u8).empty;
        defer details.deinit(allocator);

        const count = cfg.monitors.items.len;
        try details.writer(allocator).print("{d} monitors: ", .{count});

        for (cfg.monitors.items, 0..) |m, i| {
            if (i > 0) try details.writer(allocator).writeAll(", ");
            try details.writer(allocator).print("{s}", .{m.name});

            const type_str = switch (m.type) {
                .file => "file",
                .journalctl => "journalctl",
                .dmesg => "dmesg",
                .command => "command",
                .syslog => "syslog",
                .unknown => "unknown",
            };
            try details.writer(allocator).print("({s})", .{type_str});

            if (details.items.len > 100) {
                 try details.writer(allocator).writeAll("...");
                 break;
            }
        }
        return details.toOwnedSlice(allocator);
    } else {
        if (inst.command_line.len > 0) {
             var details = std.ArrayList(u8).empty;
             defer details.deinit(allocator);

             var iter = std.mem.tokenizeScalar(u8, inst.command_line, ' ');
             _ = iter.next();

             while (iter.next()) |arg| {
                 if (std.mem.eql(u8, arg, "--dmesg")) {
                     if (details.items.len > 0) try details.writer(allocator).writeAll(", ");
                     try details.writer(allocator).writeAll("dmesg");
                 } else if (std.mem.eql(u8, arg, "--file")) {
                     if (iter.next()) |val| {
                         if (details.items.len > 0) try details.writer(allocator).writeAll(", ");
                         try details.writer(allocator).print("file: {s}", .{val});
                     }
                 } else if (std.mem.eql(u8, arg, "--journalctl")) {
                     if (details.items.len > 0) try details.writer(allocator).writeAll(", ");
                     if (iter.next()) |val| {
                         try details.writer(allocator).print("journalctl: {s}", .{val});
                     } else {
                         try details.writer(allocator).writeAll("journalctl");
                     }
                 } else if (std.mem.eql(u8, arg, "--syslog")) {
                      if (iter.next()) |val| {
                         if (details.items.len > 0) try details.writer(allocator).writeAll(", ");
                         try details.writer(allocator).print("syslog: {s}", .{val});
                     }
                 }
             }

             if (details.items.len == 0) {
                 return allocator.dupe(u8, "legacy mode");
             }
             return details.toOwnedSlice(allocator);
        }
        return allocator.dupe(u8, "unknown mode");
    }
}

fn parseDuration(s: []const u8) u64 {
    if (s.len < 2) return 0;

    const unit = s[s.len - 1];
    const val_str = s[0 .. s.len - 1];
    const val = std.fmt.parseInt(u64, val_str, 10) catch return 0;

    switch (unit) {
        's' => return val * std.time.ns_per_s,
        'm' => return val * std.time.ns_per_min,
        'h' => return val * std.time.ns_per_hour,
        else => return 0,
    }
}

const Args = struct {
    dsn: []const u8,
    file: ?[]const u8 = null,
    use_dmesg: bool = false,
    command: ?[]const u8 = null,
    journalctl: ?[]const u8 = null,
    syslog: ?[]const u8 = null,
    pattern: []const u8 = "Error",
    exclude_pattern: ?[]const u8 = null,
    format: ?[]const u8 = null,
    environment: []const u8 = "production",
    release: []const u8 = "",
    verbose: bool = false,
    oneshot: bool = false,
    config: ?[]const u8 = null,
    status: bool = false,
    update: bool = false,
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
    defer if (args.exclude_pattern) |e| allocator.free(e);
    defer if (args.release.len > 0) allocator.free(args.release);
    defer if (args.config) |c| allocator.free(c);

    // IPC Commands
    const socket_dir = "/tmp/sentrylogmon";
    if (args.status) {
        var instances = ipc.listInstances(allocator, socket_dir) catch |err| {
            std.debug.print("Error listing instances: {}\n", .{err});
            std.process.exit(1);
        };
        defer {
            for (instances.items) |*inst| {
                inst.deinit(allocator);
            }
            instances.deinit(allocator);
        }

        const stdout = std.fs.File.stdout();
        if (stdout.isTty()) {
            std.debug.print("PID\tSTARTED\tUPTIME\tVERSION\tDETAILS\n", .{});
            for (instances.items) |inst| {
                const start_ts = std.fmt.parseInt(i64, inst.start_time, 10) catch 0;
                const now = std.time.timestamp();
                const uptime_sec = if (now > start_ts) now - start_ts else 0;

                const started_str = formatDate(allocator, start_ts) catch "unknown";
                defer if (!std.mem.eql(u8, started_str, "unknown")) allocator.free(started_str);

                const uptime_str = formatDuration(allocator, uptime_sec) catch "unknown";
                defer if (!std.mem.eql(u8, uptime_str, "unknown")) allocator.free(uptime_str);

                const details = getDetails(allocator, inst) catch "error";
                defer if (!std.mem.eql(u8, details, "error")) allocator.free(details);

                std.debug.print("{d}\t{s}\t{s}\t{s}\t{s}\n", .{inst.pid, started_str, uptime_str, inst.version, details});
            }
        } else {
            var buf: [4096]u8 = undefined;
            var w = stdout.writer(&buf);

            try w.interface.writeAll("[");
            for (instances.items, 0..) |inst, i| {
                if (i > 0) try w.interface.writeAll(",");

                const JsonConfig = struct {
                    sentry: config_mod.SentryConfig,
                    monitors: []const config_mod.MonitorConfig,
                };
                const JsonStatus = struct {
                    pid: i32,
                    start_time: []const u8,
                    version: []const u8,
                    config: ?JsonConfig,
                    command_line: []const u8,
                };

                var json_config: ?JsonConfig = null;
                if (inst.config) |c| {
                    json_config = .{
                        .sentry = c.sentry,
                        .monitors = c.monitors.items,
                    };
                }

                const json_inst = JsonStatus{
                    .pid = inst.pid,
                    .start_time = inst.start_time,
                    .version = inst.version,
                    .config = json_config,
                    .command_line = inst.command_line,
                };

                try std.json.Stringify.value(json_inst, .{}, &w.interface);
            }
            try w.interface.writeAll("]\n");
            try w.end();
        }
        std.process.exit(0);
    }

    if (args.update) {
        var instances = ipc.listInstances(allocator, socket_dir) catch |err| {
            std.debug.print("Error listing instances: {}\n", .{err});
            std.process.exit(1);
        };
        defer instances.deinit(allocator);

        for (instances.items) |inst| {
            const socket_path = try std.fmt.allocPrint(allocator, "{s}/sentrylogmon.{d}.sock", .{socket_dir, inst.pid});
            defer allocator.free(socket_path);

            std.debug.print("Requesting update for PID {d}...\n", .{inst.pid});
            ipc.requestUpdate(allocator, socket_path) catch |err| {
                std.debug.print("Failed to update PID {d}: {}\n", .{inst.pid, err});
                continue;
            };
            std.debug.print("Update requested for PID {d}\n", .{inst.pid});
        }
        std.process.exit(0);
    }

    if (args.dsn.len == 0 and args.config == null) {
        std.debug.print("Sentry DSN is required. Set via --dsn flag or SENTRY_DSN environment variable\n", .{});
        std.process.exit(1);
    }

    // Initialize IPC Server
    ipc.ensureSecureDirectory(socket_dir) catch |err| {
        std.debug.print("Failed to ensure secure IPC directory: {}\n", .{err});
    };
    const my_pid = std.os.linux.getpid();
    const socket_path = try std.fmt.allocPrint(allocator, "{s}/sentrylogmon.{d}.sock", .{socket_dir, my_pid});
    // We leak socket_path (used by thread)

    // Capture raw args for restart
    var raw_args = std.ArrayList([]const u8).empty;
    var raw_iter = try std.process.argsWithAllocator(allocator);
    while (raw_iter.next()) |arg| {
        try raw_args.append(allocator, try allocator.dupe(u8, arg));
    }
    const raw_args_slice = try raw_args.toOwnedSlice(allocator); // leak for thread

    const start_time = std.time.timestamp();

    if (args.verbose) {
        std.debug.print("Initialized Sentry with DSN (environment={s}, release={s})\n", .{ args.environment, args.release });
    }

    // Initialize System Statistics Collector
    const collector = try allocator.create(sysstat.Collector);
    collector.* = sysstat.Collector.init(allocator);
    defer {
        collector.deinit();
        allocator.destroy(collector);
    }

    // Start collector thread
    const collector_thread = try std.Thread.spawn(.{}, sysstat.Collector.run, .{collector});
    // We detach the collector thread as it runs indefinitely (unless we add stop mechanism, which we skip for now)
    collector_thread.detach();

    // Initialize Queue
    const queue_dir = ".sentrylogmon_queue";
    try queue_mod.Queue.init(queue_dir);
    var queue = queue_mod.Queue{ .dir_path = queue_dir };

    // Try to flush queue
    const RetryContext = struct {
        allocator: std.mem.Allocator,
        args: Args,
        collector: *sysstat.Collector,
        fn send(self: @This(), payload: []u8) !void {
            try sendSentryPayload(self.allocator, payload, self.args);
        }
    };
    // Note: Queue retry currently doesn't add system stats to old payloads (they are already built strings).
    // This is fine.
    const retry_ctx = RetryContext{ .allocator = allocator, .args = args, .collector = collector };
    queue.retryAll(allocator, retry_ctx, RetryContext.send) catch |err| {
        if (args.verbose) std.debug.print("Failed to retry queued events: {}\n", .{err});
    };

    var config: ?config_mod.Config = null;
    if (args.config) |cfg_path| {
        config = try config_mod.parseConfig(allocator, cfg_path);
    }
    defer if (config) |*c| c.deinit(allocator);

    // Start IPC Server
    const ipc_thread = try std.Thread.spawn(.{}, ipc.startServer, .{allocator, socket_path, config, raw_args_slice, start_time});
    ipc_thread.detach();

    if (config) |cfg| {
        // Multi-monitor mode
        var threads = std.ArrayList(std.Thread).empty;
        defer threads.deinit(allocator);

        for (cfg.monitors.items) |monitor| {
            var monitor_args = args;
            if (cfg.sentry.dsn.len > 0) monitor_args.dsn = cfg.sentry.dsn;
            if (cfg.sentry.environment.len > 0) monitor_args.environment = cfg.sentry.environment;
            if (cfg.sentry.release.len > 0) monitor_args.release = cfg.sentry.release;
            if (monitor.format) |fmt| monitor_args.format = fmt;

            if (monitor_args.dsn.len == 0) {
                 std.debug.print("Skipping monitor {s}: No DSN configured\n", .{monitor.name});
                 continue;
            }

            const limit = monitor.rate_limit_burst;
            var window: u64 = 0;
            if (monitor.rate_limit_window) |w| {
                window = parseDuration(w);
            }

            const ctx = try allocator.create(MonitorContext);
            ctx.* = MonitorContext{
                .allocator = allocator,
                .args = monitor_args,
                .source_name = monitor.name,
                .queue = &queue,
                .rate_limiter = RateLimiter{
                    .limit = limit,
                    .window = window,
                    .count = 0,
                    .window_start = std.time.milliTimestamp(),
                },
                .collector = collector,
            };

            const batcher = try allocator.create(batcher_mod.Batcher);
            batcher.* = batcher_mod.Batcher.init(allocator, ctx, MonitorContext.send);
            try batcher.startFlusher();

            if (args.verbose) {
                std.debug.print("Starting monitor: {s}\n", .{monitor.name});
            }

            if (monitor.type == .file) {
                 if (monitor.path) |p| {
                     const t = try std.Thread.spawn(.{}, monitorFile, .{allocator, p, monitor.pattern, monitor.exclude_pattern, batcher, monitor_args});
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
                 const t = try std.Thread.spawn(.{}, monitorCommand, .{allocator, argv.items, monitor.pattern, monitor.exclude_pattern, batcher, monitor_args});
                 try threads.append(allocator, t);
            } else if (monitor.type == .dmesg) {
                 var argv = std.ArrayList([]const u8).empty;
                 try argv.append(allocator, "dmesg");
                 if (!args.oneshot) try argv.append(allocator, "-w");
                 const t = try std.Thread.spawn(.{}, monitorCommand, .{allocator, argv.items, monitor.pattern, monitor.exclude_pattern, batcher, monitor_args});
                 try threads.append(allocator, t);
            } else if (monitor.type == .command) {
                 if (monitor.args) |cmd_str| {
                     var argv = std.ArrayList([]const u8).empty;
                     var iter = std.mem.tokenizeScalar(u8, cmd_str, ' ');
                     while (iter.next()) |part| {
                         try argv.append(allocator, part);
                     }
                     const t = try std.Thread.spawn(.{}, monitorCommand, .{allocator, argv.items, monitor.pattern, monitor.exclude_pattern, batcher, monitor_args});
                     try threads.append(allocator, t);
                 }
            } else if (monitor.type == .syslog) {
                 if (monitor.path) |addr| {
                     const t = try std.Thread.spawn(.{}, syslog.monitorSyslog, .{allocator, addr, monitor.pattern, monitor.exclude_pattern, batcher, monitor_args.verbose, monitor.format});
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
        } else if (args.syslog) |_| {
            source_name = "syslog";
        }

        // If no source specified, exit
        if (std.mem.eql(u8, source_name, "unknown") and args.config == null) {
            std.debug.print("Please specify a log source: --dmesg, --file, --command, --journalctl, --syslog, or --config\n", .{});
            std.process.exit(1);
        }

        const ctx = try allocator.create(MonitorContext);
        ctx.* = MonitorContext{
            .allocator = allocator,
            .args = args,
            .source_name = source_name,
            .queue = &queue,
            .rate_limiter = RateLimiter{
                .limit = 0,
                .window = 0,
                .count = 0,
                .window_start = 0,
            },
            .collector = collector,
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

            try monitorCommand(allocator, argv.items, args.pattern, args.exclude_pattern, batcher, args);
        } else if (args.file) |file_path| {
            if (args.verbose) {
                std.debug.print("Monitoring file: {s}\n", .{file_path});
            }
            try monitorFile(allocator, file_path, args.pattern, args.exclude_pattern, batcher, args);
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

            try monitorCommand(allocator, argv.items, args.pattern, args.exclude_pattern, batcher, args);
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

            try monitorCommand(allocator, argv.items, args.pattern, args.exclude_pattern, batcher, args);
        } else if (args.syslog) |saddr| {
            if (args.verbose) {
                std.debug.print("Monitoring syslog: {s}\n", .{saddr});
            }
            try syslog.monitorSyslog(allocator, saddr, args.pattern, args.exclude_pattern, batcher, args.verbose, args.format);
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
        \\  --syslog string
        \\        Monitor syslog (udp/tcp address e.g. udp:127.0.0.1:514)
        \\  --pattern string
        \\        Regex pattern to match (default "Error")
        \\  --exclude string
        \\        Regex pattern to exclude (default null)
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
        } else if (std.mem.eql(u8, arg, "--syslog")) {
            if (arg_iter.next()) |saddr| {
                args.syslog = try allocator.dupe(u8, saddr);
            }
        } else if (std.mem.eql(u8, arg, "--pattern")) {
            if (arg_iter.next()) |pattern| {
                args.pattern = pattern;
            }
        } else if (std.mem.eql(u8, arg, "--exclude")) {
            if (arg_iter.next()) |ep| {
                args.exclude_pattern = try allocator.dupe(u8, ep);
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
        } else if (std.mem.eql(u8, arg, "--status")) {
            args.status = true;
        } else if (std.mem.eql(u8, arg, "--update")) {
            args.update = true;
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

fn monitorFile(allocator: std.mem.Allocator, file_path: []const u8, pattern: []const u8, exclude_pattern: ?[]const u8, batcher: *batcher_mod.Batcher, args: Args) !void {
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

    // Initialize detectors
    const detector = try detectors.createDetector(allocator, args.format, pattern);
    var exclude_detector: ?detectors.Detector = null;
    if (exclude_pattern) |ep| {
        // For exclusion, we generally use string matching, but let's assume if format is JSON, exclusion might also be JSON?
        // To be safe and simple: use createDetector with the same format.
        // But if format is JSON, pattern is "key:val". Exclude pattern might be "val" or "key:val".
        // If exclude pattern is just a string, createDetector will fail if format is JSON?
        // Detectors.createDetector("json", "foo") -> JsonDetector(key="message", val="foo").
        // Detectors.createDetector("json", "k:v") -> JsonDetector(key="k", val="v").
        // This seems reasonable for exclude too.
        exclude_detector = try detectors.createDetector(allocator, args.format, ep);
    }

    var arena = std.heap.ArenaAllocator.init(allocator);
    defer arena.deinit();

    while (true) {
        const line_or_null = try readLine(in_stream, &line_buf);

        if (line_or_null) |line| {
            // Reset arena for per-line allocations (e.g. JSON parsing)
            const arena_alloc = arena.allocator();
            defer _ = arena.reset(.retain_capacity);

            // Check if line matches pattern
            if (detector.match(arena_alloc, line)) {
                var excluded = false;
                if (exclude_detector) |ed| {
                    if (ed.match(arena_alloc, line)) {
                        excluded = true;
                    }
                }

                if (excluded) {
                    if (args.verbose) {
                        std.debug.print("Excluded line: {s}\n", .{line});
                    }
                    continue;
                }

                if (args.verbose) {
                    std.debug.print("Matched line: {s}\n", .{line});
                }

                const timestamp = utils.extractTimestamp(line);
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

fn monitorCommand(allocator: std.mem.Allocator, argv: []const []const u8, pattern: []const u8, exclude_pattern: ?[]const u8, batcher: *batcher_mod.Batcher, args: Args) !void {
    // Initialize detectors
    const detector = try detectors.createDetector(allocator, args.format, pattern);
    var exclude_detector: ?detectors.Detector = null;
    if (exclude_pattern) |ep| {
        exclude_detector = try detectors.createDetector(allocator, args.format, ep);
    }

    var arena = std.heap.ArenaAllocator.init(allocator);
    defer arena.deinit();

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
                        // Reset arena for per-line allocations
                        const arena_alloc = arena.allocator();
                        defer _ = arena.reset(.retain_capacity);

                        if (detector.match(arena_alloc, line)) {
                            var excluded = false;
                            if (exclude_detector) |ed| {
                                if (ed.match(arena_alloc, line)) {
                                    excluded = true;
                                }
                            }

                            if (excluded) {
                                if (args.verbose) {
                                    std.debug.print("Excluded line: {s}\n", .{line});
                                }
                                continue;
                            }

                            if (args.verbose) {
                                std.debug.print("Matched line: {s}\n", .{line});
                            }

                            const timestamp = utils.extractTimestamp(line);
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


const MonitorContext = struct {
    allocator: std.mem.Allocator,
    args: Args,
    source_name: []const u8,
    queue: *queue_mod.Queue,
    rate_limiter: RateLimiter,
    collector: *sysstat.Collector,

    fn send(ctx_opaque: *anyopaque, timestamp: []const u8, lines: []const []const u8) !void {
        const self: *@This() = @ptrCast(@alignCast(ctx_opaque));

        if (!self.rate_limiter.allow()) {
            if (self.args.verbose) {
                std.debug.print("[{s}] Rate limited, dropping event.\n", .{self.source_name});
            }
            return;
        }

        // Build payload
        const payload = try createSentryPayload(self.allocator, timestamp, lines, self.source_name, self.args, self.collector);
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

fn createSentryPayload(allocator: std.mem.Allocator, timestamp: []const u8, lines: []const []const u8, source: []const u8, args: Args, collector: *sysstat.Collector) ![]u8 {
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

    try writer.writeAll("\"}}");

    // Contexts (System State)
    try writer.writeAll(",\"contexts\":{\"Server State\":");

    // Get stats json
    const stats_json = try collector.getJson(allocator);
    defer allocator.free(stats_json);

    // We assume stats_json is valid JSON object
    try writer.writeAll(stats_json);

    try writer.writeAll("}}");

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
    _ = detectors;
    _ = sysstat;
    _ = syslog;
    _ = utils;
}

