const std = @import("std");
const utils = @import("utils.zig");
const batcher_mod = @import("batcher.zig");
const detectors = @import("detectors.zig");

const Protocol = enum {
    udp,
    tcp,
};

const ParsedAddress = struct {
    protocol: Protocol,
    address: std.net.Address,
};

fn parseAddress(allocator: std.mem.Allocator, address_str: []const u8) !ParsedAddress {
    var protocol = Protocol.udp;
    var host_port = address_str;

    if (std.mem.startsWith(u8, address_str, "tcp:")) {
        protocol = .tcp;
        host_port = address_str[4..];
    } else if (std.mem.startsWith(u8, address_str, "udp:")) {
        protocol = .udp;
        host_port = address_str[4..];
    }

    // Split host and port
    var host: []const u8 = "0.0.0.0";
    var port: u16 = 514;

    if (std.mem.lastIndexOf(u8, host_port, ":")) |idx| {
        host = host_port[0..idx];
        const port_str = host_port[idx + 1 ..];
        port = std.fmt.parseInt(u16, port_str, 10) catch 514;
    } else {
        // No port specified? use default 514
        // But if host_port is just a number, it might be port?
        // Let's assume input is "ip:port" or "ip".
        host = host_port;
    }

    // Resolve address
    // We use getAddressList to support hostnames and IPs
    const list = try std.net.getAddressList(allocator, host, port);
    defer list.deinit();

    if (list.addrs.len == 0) return error.AddressNotFound;

    return ParsedAddress{
        .protocol = protocol,
        .address = list.addrs[0],
    };
}

pub fn monitorSyslog(allocator: std.mem.Allocator, address_str: []const u8, pattern: []const u8, exclude_pattern: ?[]const u8, batcher: *batcher_mod.Batcher, verbose: bool, format: ?[]const u8) !void {
    const parsed = try parseAddress(allocator, address_str);

    if (verbose) {
        std.debug.print("Starting Syslog monitor on {any} ({s})\n", .{ parsed.address, @tagName(parsed.protocol) });
    }

    switch (parsed.protocol) {
        .udp => try runUdp(allocator, parsed.address, pattern, exclude_pattern, batcher, verbose, format),
        .tcp => try runTcp(allocator, parsed.address, pattern, exclude_pattern, batcher, verbose, format),
    }
}

fn runUdp(allocator: std.mem.Allocator, address: std.net.Address, pattern: []const u8, exclude_pattern: ?[]const u8, batcher: *batcher_mod.Batcher, verbose: bool, format: ?[]const u8) !void {
    const socket = try std.posix.socket(address.any.family, std.posix.SOCK.DGRAM, std.posix.IPPROTO.UDP);
    defer std.posix.close(socket);

    try std.posix.bind(socket, &address.any, address.getOsSockLen());

    var buf: [65536]u8 = undefined;

    // Detectors
    const detector = try detectors.createDetector(allocator, format, pattern);
    var exclude_detector: ?detectors.Detector = null;
    if (exclude_pattern) |ep| {
        exclude_detector = try detectors.createDetector(allocator, format, ep);
    }

    var arena = std.heap.ArenaAllocator.init(allocator);
    defer arena.deinit();

    while (true) {
        const len = std.posix.recvfrom(socket, &buf, 0, null, null) catch |err| {
            if (verbose) std.debug.print("UDP recv error: {}\n", .{err});
            continue;
        };

        if (len == 0) continue;

        const data = buf[0..len];
        // Split by newline if present (RFC says syslog might be one packet per message, but sometimes batched)
        // Rust implementation appends newline if missing.
        // We will just process lines.

        var iter = std.mem.splitScalar(u8, data, '\n');
        while (iter.next()) |line_raw| {
            const line = std.mem.trim(u8, line_raw, "\r");
            if (line.len == 0) continue;

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
                    if (verbose) std.debug.print("Excluded syslog: {s}\n", .{line});
                    continue;
                }

                if (verbose) std.debug.print("Matched syslog: {s}\n", .{line});

                const timestamp = utils.extractTimestamp(line);
                batcher.add(timestamp, line) catch |err| {
                    if (verbose) std.debug.print("Batcher add error: {}\n", .{err});
                };
            }
        }
    }
}

fn runTcp(allocator: std.mem.Allocator, address: std.net.Address, pattern: []const u8, exclude_pattern: ?[]const u8, batcher: *batcher_mod.Batcher, verbose: bool, format: ?[]const u8) !void {
    var server = try address.listen(.{ .reuse_address = true });
    defer server.deinit();

    while (true) {
        const conn = server.accept() catch |err| {
            if (verbose) std.debug.print("TCP accept error: {}\n", .{err});
            continue;
        };

        const thread_args = .{ allocator, conn, pattern, exclude_pattern, batcher, verbose, format };
        const thread = std.Thread.spawn(.{}, handleTcpConnection, thread_args) catch |err| {
            if (verbose) std.debug.print("Failed to spawn TCP thread: {}\n", .{err});
            conn.stream.close();
            continue;
        };
        thread.detach();
    }
}

fn handleTcpConnection(allocator: std.mem.Allocator, conn: std.net.Server.Connection, pattern: []const u8, exclude_pattern: ?[]const u8, batcher: *batcher_mod.Batcher, verbose: bool, format: ?[]const u8) void {
    defer conn.stream.close();

    var reader_buf: [4096]u8 = undefined;
    var raw_reader = conn.stream.reader(&reader_buf);
    const reader = raw_reader.interface();

    // Detectors
    const detector = detectors.createDetector(allocator, format, pattern) catch return;
    var exclude_detector: ?detectors.Detector = null;
    if (exclude_pattern) |ep| {
        exclude_detector = detectors.createDetector(allocator, format, ep) catch null;
    }

    var arena = std.heap.ArenaAllocator.init(allocator);
    defer arena.deinit();

    while (true) {
        const line = reader.takeDelimiterExclusive('\n') catch |err| {
            if (err == error.EndOfStream) break;
            if (verbose) std.debug.print("TCP read error: {}\n", .{err});
            break;
        };

        // Trim CR if present
        const trimmed = std.mem.trimRight(u8, line, "\r");

        if (trimmed.len == 0) continue;

        const arena_alloc = arena.allocator();
        defer _ = arena.reset(.retain_capacity);

        if (detector.match(arena_alloc, trimmed)) {
            var excluded = false;
            if (exclude_detector) |ed| {
                if (ed.match(arena_alloc, trimmed)) {
                    excluded = true;
                }
            }

            if (excluded) {
                if (verbose) std.debug.print("Excluded syslog: {s}\n", .{trimmed});
                continue;
            }

            if (verbose) std.debug.print("Matched syslog: {s}\n", .{trimmed});

            const timestamp = utils.extractTimestamp(trimmed);
            batcher.add(timestamp, trimmed) catch {};
        }
    }
}
