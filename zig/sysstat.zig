const std = @import("std");

pub const ProcessInfo = struct {
    pid: []const u8,
    rss: []const u8,
    cpu: []const u8,
    mem: []const u8,
    command: []const u8,

    // Internal fields for sorting (not serialized)
    cpu_usage: f64,
    mem_usage: f64,

    pub fn deinit(self: *ProcessInfo, allocator: std.mem.Allocator) void {
        allocator.free(self.pid);
        allocator.free(self.rss);
        allocator.free(self.cpu);
        allocator.free(self.mem);
        allocator.free(self.command);
    }
};

pub const PressureInfo = struct {
    avg10: f64,
    avg60: f64,
    avg300: f64,
    total: f64,
};

pub const LoadAvg = struct {
    load1: f64,
    load5: f64,
    load15: f64,
};

pub const MemInfo = struct {
    total: u64,
    available: u64,
    used_percent: f64,
};

pub const SystemState = struct {
    timestamp: i64,
    uptime: u64,
    load: ?LoadAvg,
    memory: ?MemInfo,
    disk_pressure: ?PressureInfo,
    top_cpu: []ProcessInfo,
    top_mem: []ProcessInfo,
    process_summary: []const u8,

    pub fn deinit(self: *SystemState, allocator: std.mem.Allocator) void {
        for (self.top_cpu) |*p| p.deinit(allocator);
        allocator.free(self.top_cpu);
        for (self.top_mem) |*p| p.deinit(allocator);
        allocator.free(self.top_mem);
        if (self.process_summary.len > 0) allocator.free(self.process_summary);
    }
};

pub const Collector = struct {
    mutex: std.Thread.Mutex,
    state: ?SystemState,
    allocator: std.mem.Allocator,

    pub fn init(allocator: std.mem.Allocator) Collector {
        return Collector{
            .mutex = std.Thread.Mutex{},
            .state = null,
            .allocator = allocator,
        };
    }

    pub fn deinit(self: *Collector) void {
        self.mutex.lock();
        defer self.mutex.unlock();
        if (self.state) |*s| {
            s.deinit(self.allocator);
        }
    }

    pub fn getJson(self: *Collector, allocator: std.mem.Allocator) ![]const u8 {
        self.mutex.lock();
        defer self.mutex.unlock();

        if (self.state) |state| {
            var list = std.ArrayList(u8).empty;
            errdefer list.deinit(allocator);

            const writer = list.writer(allocator);
            try writer.writeAll("{");
            try writer.print("\"timestamp\":{d},", .{state.timestamp});
            try writer.print("\"uptime\":{d},", .{state.uptime});

            if (state.load) |l| {
                try writer.print("\"load\":{{\"load1\":{d:.2},\"load5\":{d:.2},\"load15\":{d:.2}}},", .{l.load1, l.load5, l.load15});
            } else {
                try writer.writeAll("\"load\":null,");
            }

            if (state.memory) |m| {
                try writer.print("\"memory\":{{\"total\":{d},\"available\":{d},\"used_percent\":{d:.1}}},", .{m.total, m.available, m.used_percent});
            } else {
                try writer.writeAll("\"memory\":null,");
            }

            if (state.disk_pressure) |p| {
                try writer.print("\"disk_pressure\":{{\"avg10\":{d:.2},\"avg60\":{d:.2},\"avg300\":{d:.2},\"total\":{d:.2}}},", .{p.avg10, p.avg60, p.avg300, p.total});
            }

            try writer.writeAll("\"top_cpu\":[");
            for (state.top_cpu, 0..) |p, i| {
                if (i > 0) try writer.writeAll(",");
                try serializeProcess(writer, p);
            }
            try writer.writeAll("],");

            try writer.writeAll("\"top_mem\":[");
            for (state.top_mem, 0..) |p, i| {
                if (i > 0) try writer.writeAll(",");
                try serializeProcess(writer, p);
            }
            try writer.writeAll("],");

            try writer.writeAll("\"process_summary\":\"");
            try writeEscaped(writer, state.process_summary);
            try writer.writeAll("\"}");

            return list.toOwnedSlice(allocator);
        }
        return allocator.dupe(u8, "{}");
    }

    fn writeEscaped(writer: anytype, s: []const u8) !void {
        for (s) |c| {
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

    fn serializeProcess(writer: anytype, p: ProcessInfo) !void {
        try writer.writeAll("{");
        try writer.print("\"pid\":\"{s}\",", .{p.pid});
        try writer.print("\"rss\":\"{s}\",", .{p.rss});
        try writer.print("\"cpu\":\"{s}\",", .{p.cpu});
        try writer.print("\"mem\":\"{s}\",", .{p.mem});
        try writer.writeAll("\"command\":\"");
        try writeEscaped(writer, p.command);
        try writer.writeAll("\"}");
    }

    pub fn collect(self: *Collector) !void {
        var new_state = SystemState{
            .timestamp = std.time.timestamp(),
            .uptime = 0,
            .load = null,
            .memory = null,
            .disk_pressure = null,
            .top_cpu = &[_]ProcessInfo{},
            .top_mem = &[_]ProcessInfo{},
            .process_summary = "",
        };
        errdefer new_state.deinit(self.allocator);

        // 1. Uptime
        if (readUptime(self.allocator)) |u| {
            new_state.uptime = u;
        } else |_| {}

        // 2. Load Average
        if (readLoadAvg(self.allocator)) |l| {
            new_state.load = l;
        } else |_| {}

        // 3. Memory
        if (readMemInfo(self.allocator)) |m| {
            new_state.memory = m;
        } else |_| {}

        // 4. Disk Pressure
        if (readDiskPressure(self.allocator)) |p| {
            new_state.disk_pressure = p;
        } else |_| {}

        // 5. Processes
        // This is the heavy part.
        // We'll skip it for now in the first pass to ensure basic things compile and run.
        // Or implement a simplified version.

        // Let's implement full process stats collection.
        if (getProcessStats(self.allocator, new_state.uptime, if (new_state.memory) |m| m.total else 0)) |stats| {
            new_state.top_cpu = stats.top_cpu;
            new_state.top_mem = stats.top_mem;
            new_state.process_summary = stats.summary;
        } else |err| {
            std.debug.print("Error collecting processes: {}\n", .{err});
        }

        self.mutex.lock();
        defer self.mutex.unlock();
        if (self.state) |*s| {
            s.deinit(self.allocator);
        }
        self.state = new_state;
    }

    pub fn run(self: *Collector) !void {
        // Initial collection
        self.collect() catch |err| {
             std.debug.print("Initial collection failed: {}\n", .{err});
        };

        while (true) {
            std.Thread.sleep(60 * std.time.ns_per_s);
            self.collect() catch |err| {
                std.debug.print("Collection failed: {}\n", .{err});
            };
        }
    }
};

fn readFile(allocator: std.mem.Allocator, path: []const u8) ![]u8 {
    const file = try std.fs.openFileAbsolute(path, .{});
    defer file.close();
    // /proc files are usually small, but let's limit to 64KB
    return file.readToEndAlloc(allocator, 64 * 1024);
}

fn readUptime(allocator: std.mem.Allocator) !u64 {
    const content = try readFile(allocator, "/proc/uptime");
    defer allocator.free(content);

    var iter = std.mem.tokenizeScalar(u8, content, ' ');
    if (iter.next()) |part| {
        const val = try std.fmt.parseFloat(f64, part);
        return @intFromFloat(val);
    }
    return error.InvalidFormat;
}

fn readLoadAvg(allocator: std.mem.Allocator) !LoadAvg {
    const content = try readFile(allocator, "/proc/loadavg");
    defer allocator.free(content);

    var iter = std.mem.tokenizeScalar(u8, content, ' ');
    const l1 = try std.fmt.parseFloat(f64, iter.next() orelse return error.InvalidFormat);
    const l2 = try std.fmt.parseFloat(f64, iter.next() orelse return error.InvalidFormat);
    const l3 = try std.fmt.parseFloat(f64, iter.next() orelse return error.InvalidFormat);

    return LoadAvg{ .load1 = l1, .load5 = l2, .load15 = l3 };
}

fn readMemInfo(allocator: std.mem.Allocator) !MemInfo {
    const content = try readFile(allocator, "/proc/meminfo");
    defer allocator.free(content);

    var total: u64 = 0;
    var available: u64 = 0;

    var lines = std.mem.splitScalar(u8, content, '\n');
    while (lines.next()) |line| {
        if (std.mem.startsWith(u8, line, "MemTotal:")) {
            total = try parseMemVal(line);
        } else if (std.mem.startsWith(u8, line, "MemAvailable:")) {
            available = try parseMemVal(line);
        }
    }

    if (total == 0) return error.InvalidFormat;

    // kB to Bytes
    total *= 1024;
    available *= 1024;

    const used = total - available;
    const used_percent = @as(f64, @floatFromInt(used)) / @as(f64, @floatFromInt(total)) * 100.0;

    return MemInfo{ .total = total, .available = available, .used_percent = used_percent };
}

fn parseMemVal(line: []const u8) !u64 {
    var iter = std.mem.tokenizeScalar(u8, line, ' ');
    _ = iter.next(); // Key:
    const val_str = iter.next() orelse return error.InvalidFormat;
    return std.fmt.parseInt(u64, val_str, 10);
}

fn readDiskPressure(allocator: std.mem.Allocator) !PressureInfo {
    const content = try readFile(allocator, "/proc/pressure/io");
    defer allocator.free(content);

    var p = PressureInfo{ .avg10 = 0, .avg60 = 0, .avg300 = 0, .total = 0 };

    var lines = std.mem.splitScalar(u8, content, '\n');
    while (lines.next()) |line| {
        if (std.mem.startsWith(u8, line, "some")) {
            var iter = std.mem.tokenizeScalar(u8, line, ' ');
            _ = iter.next(); // some
            while (iter.next()) |part| {
                var kv = std.mem.splitScalar(u8, part, '=');
                const key = kv.first();
                const val_str = kv.next() orelse continue;
                const val = try std.fmt.parseFloat(f64, val_str);

                if (std.mem.eql(u8, key, "avg10")) p.avg10 = val
                else if (std.mem.eql(u8, key, "avg60")) p.avg60 = val
                else if (std.mem.eql(u8, key, "avg300")) p.avg300 = val
                else if (std.mem.eql(u8, key, "total")) p.total = val;
            }
            return p;
        }
    }
    return error.NotFound;
}

const ProcessStatsResult = struct {
    top_cpu: []ProcessInfo,
    top_mem: []ProcessInfo,
    summary: []const u8,
};

fn getProcessStats(allocator: std.mem.Allocator, uptime: u64, total_mem: u64) !ProcessStatsResult {
    var proc_dir = try std.fs.openDirAbsolute("/proc", .{ .iterate = true });
    defer proc_dir.close();

    // Use iterator instead.
    var iter = proc_dir.iterate();

    var procs = std.ArrayList(ProcessInfo).empty;
    defer {
        // If we error out or return, we need to handle ownership.
        // If success, we transfer ownership of some items to result.
        // For now, let's just use errdefer to cleanup all.
        // Logic below handles moving items.
    }
    errdefer {
        for (procs.items) |*p| p.deinit(allocator);
        procs.deinit(allocator);
    }

    const clk_tck = 100.0; // Assume 100Hz
    const page_size = 4096; // Assume 4KB

    while (try iter.next()) |entry| {
        if (entry.kind != .directory) continue;

        // Check if name is numeric
        const pid = std.fmt.parseInt(i32, entry.name, 10) catch continue;

        // Read stat
        const stat_path = try std.fmt.allocPrint(allocator, "/proc/{d}/stat", .{pid});
        defer allocator.free(stat_path);

        const stat_content = readFile(allocator, stat_path) catch continue;
        defer allocator.free(stat_content);

        // Parse stat
        // PID (comm) state ppid ...
        // We need fields: 14(utime), 15(stime), 22(starttime), 24(rss)
        // Note: comm can contain spaces and parenthesis. Find last ')'
        const closing_paren = std.mem.lastIndexOf(u8, stat_content, ")") orelse continue;
        const rest = stat_content[closing_paren + 2 ..]; // space after )

        var stat_iter = std.mem.tokenizeScalar(u8, rest, ' ');
        // After comm:
        // 3: state, 4: ppid, 5: pgrp, 6: session, 7: tty_nr, 8: tpgid, 9: flags,
        // 10: minflt, 11: cminflt, 12: majflt, 13: cmajflt
        // 14: utime, 15: stime
        // ...
        // 22: starttime
        // ...
        // 24: rss

        // Skip 3..13 (11 fields)
        for (0..11) |_| _ = stat_iter.next();

        const utime_str = stat_iter.next() orelse continue;
        const stime_str = stat_iter.next() orelse continue;

        // Skip 16..21 (6 fields)
        for (0..6) |_| _ = stat_iter.next();

        const starttime_str = stat_iter.next() orelse continue;

        // Skip 23 (1 field)
        _ = stat_iter.next();

        const rss_str = stat_iter.next() orelse continue;

        const utime = std.fmt.parseFloat(f64, utime_str) catch 0;
        const stime = std.fmt.parseFloat(f64, stime_str) catch 0;
        const starttime = std.fmt.parseFloat(f64, starttime_str) catch 0;
        const rss_pages = std.fmt.parseFloat(f64, rss_str) catch 0;

        // Calculate CPU
        const total_ticks = utime + stime;
        const starttime_seconds = starttime / clk_tck;
        var cpu_usage: f64 = 0;
        if (@as(f64, @floatFromInt(uptime)) > starttime_seconds) {
            const seconds_active = @as(f64, @floatFromInt(uptime)) - starttime_seconds;
             cpu_usage = (total_ticks / clk_tck) / seconds_active * 100.0;
        }

        // Calculate Mem
        const rss_bytes = rss_pages * page_size;
        var mem_usage: f64 = 0;
        if (total_mem > 0) {
            mem_usage = (rss_bytes / @as(f64, @floatFromInt(total_mem))) * 100.0;
        }

        // We delay command reading until we know it's a top process to save IO
        // Store partial info
        try procs.append(allocator, ProcessInfo{
            .pid = try allocator.dupe(u8, entry.name),
            .rss = try std.fmt.allocPrint(allocator, "{d:.0}", .{rss_bytes}),
            .cpu = try std.fmt.allocPrint(allocator, "{d:.1}", .{cpu_usage}),
            .mem = try std.fmt.allocPrint(allocator, "{d:.1}", .{mem_usage}),
            .command = try allocator.dupe(u8, ""), // Empty for now
            .cpu_usage = cpu_usage,
            .mem_usage = mem_usage,
        });
    }

    const summary = try std.fmt.allocPrint(allocator, "Total Processes: {d}", .{procs.items.len});

    // Sort for CPU
    std.sort.block(ProcessInfo, procs.items, {}, struct {
        fn lessThan(_: void, lhs: ProcessInfo, rhs: ProcessInfo) bool {
            return lhs.cpu_usage > rhs.cpu_usage;
        }
    }.lessThan);

    var top_cpu = std.ArrayList(ProcessInfo).empty;
    const count = @min(procs.items.len, 5);
    for (0..count) |i| {
        var p = procs.items[i];
        try populateCommand(allocator, &p);
        // We need to duplicate because procs will be freed, but we can just steal ownership if we are careful.
        // But procs.deinit() will free strings.
        // Let's Deep Copy.
        try top_cpu.append(allocator, try deepCopyProcess(allocator, p));
    }

    // Sort for Mem
    std.sort.block(ProcessInfo, procs.items, {}, struct {
        fn lessThan(_: void, lhs: ProcessInfo, rhs: ProcessInfo) bool {
            return lhs.mem_usage > rhs.mem_usage;
        }
    }.lessThan);

    var top_mem = std.ArrayList(ProcessInfo).empty;
    for (0..count) |i| {
        var p = procs.items[i];
        // Command might be populated already if it was in top_cpu
        try populateCommand(allocator, &p);
        try top_mem.append(allocator, try deepCopyProcess(allocator, p));
    }

    // Clean up temporary list
    for (procs.items) |*p| p.deinit(allocator);
    procs.deinit(allocator);

    return ProcessStatsResult{
        .top_cpu = try top_cpu.toOwnedSlice(allocator),
        .top_mem = try top_mem.toOwnedSlice(allocator),
        .summary = summary,
    };
}

fn deepCopyProcess(allocator: std.mem.Allocator, p: ProcessInfo) !ProcessInfo {
    return ProcessInfo{
        .pid = try allocator.dupe(u8, p.pid),
        .rss = try allocator.dupe(u8, p.rss),
        .cpu = try allocator.dupe(u8, p.cpu),
        .mem = try allocator.dupe(u8, p.mem),
        .command = try allocator.dupe(u8, p.command),
        .cpu_usage = p.cpu_usage,
        .mem_usage = p.mem_usage,
    };
}

fn populateCommand(allocator: std.mem.Allocator, p: *ProcessInfo) !void {
    if (p.command.len > 0) return;

    // Free the empty string we allocated
    allocator.free(p.command);

    const cmd_path = try std.fmt.allocPrint(allocator, "/proc/{s}/cmdline", .{p.pid});
    defer allocator.free(cmd_path);

    if (readFile(allocator, cmd_path)) |content| {
        defer allocator.free(content);
        // cmdline is null-separated
        var args = std.ArrayList([]const u8).empty;
        defer args.deinit(allocator);

        var iter = std.mem.splitScalar(u8, content, 0);
        while (iter.next()) |arg| {
            if (arg.len > 0) try args.append(allocator, arg);
        }

        // Sanitize
        p.command = try sanitizeCommand(allocator, args.items);
    } else |_| {
         // Fallback to comm from stat? Accessing /proc again...
         // For now just "unknown"
         p.command = try allocator.dupe(u8, "unknown");
    }
}

fn sanitizeCommand(allocator: std.mem.Allocator, args: []const []const u8) ![]u8 {
    if (args.len == 0) return allocator.dupe(u8, "");

    var out = std.ArrayList(u8).empty;
    errdefer out.deinit(allocator);

    var skip_next = false;

    const sensitive_flags = std.StaticStringMap(bool).initComptime(.{
        .{ "--password", true },
        .{ "--token", true },
        .{ "--api-key", true },
        .{ "--apikey", true },
        .{ "--secret", true },
        .{ "--client-secret", true },
        .{ "--access-token", true },
        .{ "--auth-token", true },
    });

    const sensitive_suffixes = [_][]const u8{
        "password", "token", "secret", "_key",
    };

    for (args, 0..) |arg, i| {
        if (i > 0) try out.append(allocator, ' ');

        if (skip_next) {
            try out.appendSlice(allocator, "[REDACTED]");
            skip_next = false;
            continue;
        }

        // Check key=value
        if (std.mem.indexOf(u8, arg, "=")) |eq_idx| {
            const key = arg[0..eq_idx];

            // Trim left dashes
            const clean_key = std.mem.trimLeft(u8, key, "-");

            var sensitive = false;
            const lower_key = try std.ascii.allocLowerString(allocator, clean_key);
            defer allocator.free(lower_key);

            if (std.mem.eql(u8, lower_key, "password") or
                std.mem.eql(u8, lower_key, "token") or
                std.mem.eql(u8, lower_key, "secret") or
                std.mem.eql(u8, lower_key, "key") or
                std.mem.eql(u8, lower_key, "auth")) {
                sensitive = true;
            } else {
                 for (sensitive_suffixes) |suffix| {
                     if (std.mem.endsWith(u8, lower_key, suffix)) {
                         sensitive = true;
                         break;
                     }
                 }
            }

            if (sensitive or sensitive_flags.has(key)) {
                try out.appendSlice(allocator, key);
                try out.appendSlice(allocator, "=[REDACTED]");
                continue;
            }
        }

        if (sensitive_flags.has(arg)) {
            try out.appendSlice(allocator, arg);
            skip_next = true;
            continue;
        }

        try out.appendSlice(allocator, arg);
    }

    return out.toOwnedSlice(allocator);
}

test "sanitizeCommand" {
    const allocator = std.testing.allocator;
    const args = [_][]const u8{ "curl", "--user", "user:pass", "--token", "123", "--url=http://example.com?key=secret" };
    // Note: our logic redacts next arg for --token, but key=value for --url (if key is sensitive).
    // wait, --url=... key is --url. --url is not sensitive.
    // user:pass is not flagged by simple logic unless it matches something.
    // The current Go implementation handles --flag=value.

    const res = try sanitizeCommand(allocator, &args);
    defer allocator.free(res);

    // std.debug.print("Sanitized: {s}\n", .{res});
    // Expected: curl --user user:pass --token [REDACTED] --url=http://example.com?key=secret
}
