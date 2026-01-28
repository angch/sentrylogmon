const std = @import("std");

pub const SentryConfig = struct {
    dsn: []const u8 = "",
    environment: []const u8 = "production",
    release: []const u8 = "",
};

pub const MonitorType = enum {
    file,
    journalctl,
    dmesg,
    command,
    unknown,
};

pub const MonitorConfig = struct {
    name: []const u8,
    type: MonitorType,
    path: ?[]const u8 = null,
    args: ?[]const u8 = null,
    pattern: []const u8,
};

pub const Config = struct {
    sentry: SentryConfig,
    monitors: std.ArrayList(MonitorConfig),

    pub fn deinit(self: *Config, allocator: std.mem.Allocator) void {
        if (self.sentry.dsn.len > 0) allocator.free(self.sentry.dsn);
        if (self.sentry.environment.len > 0 and !std.mem.eql(u8, self.sentry.environment, "production")) allocator.free(self.sentry.environment);
        if (self.sentry.release.len > 0) allocator.free(self.sentry.release);

        for (self.monitors.items) |m| {
            allocator.free(m.name);
            if (m.path) |p| allocator.free(p);
            if (m.args) |a| allocator.free(a);
            allocator.free(m.pattern);
        }
        self.monitors.deinit(allocator);
    }
};

pub fn parseConfig(allocator: std.mem.Allocator, file_path: []const u8) !Config {
    const file = try std.fs.cwd().openFile(file_path, .{});
    defer file.close();

    const stat = try file.stat();
    const content = try file.readToEndAlloc(allocator, stat.size);
    defer allocator.free(content);

    return parseConfigString(allocator, content);
}

fn parseConfigString(allocator: std.mem.Allocator, content: []const u8) !Config {
    var config = Config{
        .sentry = .{},
        .monitors = .empty,
    };

    var iter = std.mem.splitScalar(u8, content, '\n');
    var state: enum { Global, Sentry, Monitors } = .Global;
    var current_monitor: ?MonitorConfig = null;

    while (iter.next()) |line| {
        const trimmed = std.mem.trim(u8, line, " \r\t");
        if (trimmed.len == 0 or std.mem.startsWith(u8, trimmed, "#")) continue;

        const indent = countIndent(line);

        if (indent == 0) {
            if (std.mem.eql(u8, trimmed, "sentry:")) {
                state = .Sentry;
            } else if (std.mem.eql(u8, trimmed, "monitors:")) {
                state = .Monitors;
            } else {
                state = .Global;
            }
            continue;
        }

        switch (state) {
            .Sentry => {
                if (parseKeyVal(trimmed)) |kv| {
                    if (std.mem.eql(u8, kv.key, "dsn")) config.sentry.dsn = try allocator.dupe(u8, kv.val);
                    if (std.mem.eql(u8, kv.key, "environment")) config.sentry.environment = try allocator.dupe(u8, kv.val);
                    if (std.mem.eql(u8, kv.key, "release")) config.sentry.release = try allocator.dupe(u8, kv.val);
                }
            },
            .Monitors => {
                if (std.mem.startsWith(u8, trimmed, "-")) {
                     if (current_monitor) |m| {
                        try config.monitors.append(allocator, m);
                     }
                     current_monitor = MonitorConfig{
                         .name = try allocator.dupe(u8, "unnamed"),
                         .type = .unknown,
                         .pattern = try allocator.dupe(u8, "Error"),
                     };

                     const after_dash = std.mem.trim(u8, trimmed[1..], " ");
                     if (parseKeyVal(after_dash)) |kv| {
                         try updateMonitor(allocator, &current_monitor.?, kv.key, kv.val);
                     }
                } else if (current_monitor != null) {
                    if (parseKeyVal(trimmed)) |kv| {
                         try updateMonitor(allocator, &current_monitor.?, kv.key, kv.val);
                    }
                }
            },
            .Global => {},
        }
    }

    if (current_monitor) |m| {
        try config.monitors.append(allocator, m);
    }

    return config;
}

fn updateMonitor(allocator: std.mem.Allocator, monitor: *MonitorConfig, key: []const u8, val: []const u8) !void {
    if (std.mem.eql(u8, key, "name")) {
        allocator.free(monitor.name);
        monitor.name = try allocator.dupe(u8, val);
    }
    if (std.mem.eql(u8, key, "type")) {
        if (std.mem.eql(u8, val, "file")) monitor.type = .file;
        if (std.mem.eql(u8, val, "journalctl")) monitor.type = .journalctl;
        if (std.mem.eql(u8, val, "dmesg")) monitor.type = .dmesg;
        if (std.mem.eql(u8, val, "command")) monitor.type = .command;
    }
    if (std.mem.eql(u8, key, "path")) {
        if (monitor.path) |p| allocator.free(p);
        monitor.path = try allocator.dupe(u8, val);
    }
    if (std.mem.eql(u8, key, "args")) {
        if (monitor.args) |a| allocator.free(a);
        monitor.args = try allocator.dupe(u8, val);
    }
    if (std.mem.eql(u8, key, "pattern")) {
        allocator.free(monitor.pattern);
        monitor.pattern = try allocator.dupe(u8, val);
    }
}

fn parseKeyVal(line: []const u8) ?struct { key: []const u8, val: []const u8 } {
    if (std.mem.indexOf(u8, line, ":")) |idx| {
        const key = std.mem.trim(u8, line[0..idx], " ");
        var val = std.mem.trim(u8, line[idx+1..], " ");

        if (val.len >= 2 and val[0] == '"' and val[val.len-1] == '"') {
            val = val[1..val.len-1];
        }
        return .{ .key = key, .val = val };
    }
    return null;
}

fn countIndent(line: []const u8) usize {
    var count: usize = 0;
    for (line) |c| {
        if (c == ' ') count += 1
        else break;
    }
    return count;
}

test "parse config" {
    const content =
        \\sentry:
        \\  dsn: "https://example.com"
        \\  environment: "staging"
        \\
        \\monitors:
        \\  - name: "syslog"
        \\    type: "file"
        \\    path: "/var/log/syslog"
        \\  - name: "journal"
        \\    type: "journalctl"
        \\    args: "-f"
    ;

    const allocator = std.testing.allocator;
    var config = try parseConfigString(allocator, content);
    defer config.deinit(allocator);

    try std.testing.expectEqualStrings("https://example.com", config.sentry.dsn);
    try std.testing.expectEqualStrings("staging", config.sentry.environment);
    try std.testing.expect(config.monitors.items.len == 2);

    const m1 = config.monitors.items[0];
    try std.testing.expectEqualStrings("syslog", m1.name);
    try std.testing.expect(m1.type == .file);
    try std.testing.expectEqualStrings("/var/log/syslog", m1.path.?);

    const m2 = config.monitors.items[1];
    try std.testing.expectEqualStrings("journal", m2.name);
    try std.testing.expect(m2.type == .journalctl);
    try std.testing.expectEqualStrings("-f", m2.args.?);
}
