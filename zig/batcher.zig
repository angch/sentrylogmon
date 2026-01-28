const std = @import("std");

pub const Batcher = struct {
    allocator: std.mem.Allocator,
    mutex: std.Thread.Mutex,
    // Buffer: Timestamp -> List of lines
    buffer: std.StringHashMap(std.ArrayList([]const u8)),

    // Callback: fn(context, timestamp, lines)
    context: *anyopaque,
    sendFn: *const fn (*anyopaque, []const u8, []const []const u8) anyerror!void,

    pub fn init(
        allocator: std.mem.Allocator,
        context: *anyopaque,
        sendFn: *const fn (*anyopaque, []const u8, []const []const u8) anyerror!void
    ) Batcher {
        return Batcher{
            .allocator = allocator,
            .mutex = .{},
            .buffer = std.StringHashMap(std.ArrayList([]const u8)).init(allocator),
            .context = context,
            .sendFn = sendFn,
        };
    }

    pub fn deinit(self: *Batcher) void {
        self.mutex.lock();
        defer self.mutex.unlock();

        var iter = self.buffer.iterator();
        while (iter.next()) |entry| {
            self.allocator.free(entry.key_ptr.*);
            for (entry.value_ptr.items) |line| {
                self.allocator.free(line);
            }
            entry.value_ptr.deinit(self.allocator);
        }
        self.buffer.deinit();
    }

    pub fn add(self: *Batcher, timestamp: []const u8, line: []const u8) !void {
        self.mutex.lock();
        defer self.mutex.unlock();

        const result = try self.buffer.getOrPut(timestamp);
        if (!result.found_existing) {
             // New entry, dupe key
             result.key_ptr.* = try self.allocator.dupe(u8, timestamp);
             result.value_ptr.* = .empty;
        }

        const line_copy = try self.allocator.dupe(u8, line);
        errdefer self.allocator.free(line_copy);
        try result.value_ptr.append(self.allocator, line_copy);
    }

    pub fn flush(self: *Batcher) !void {
        // Swap buffer
        self.mutex.lock();
        var old_buffer = self.buffer;
        self.buffer = std.StringHashMap(std.ArrayList([]const u8)).init(self.allocator);
        self.mutex.unlock();

        // Process old buffer
        var iter = old_buffer.iterator();
        while (iter.next()) |entry| {
            const timestamp = entry.key_ptr.*;
            const lines = entry.value_ptr.items;

            // Send
            self.sendFn(self.context, timestamp, lines) catch |err| {
                std.debug.print("Error sending batch for {s}: {}\n", .{timestamp, err});
            };

            // Cleanup
            self.allocator.free(timestamp);
            for (lines) |line| {
                self.allocator.free(line);
            }
            entry.value_ptr.deinit(self.allocator);
        }
        old_buffer.deinit();
    }

    pub fn startFlusher(self: *Batcher) !void {
        const thread = try std.Thread.spawn(.{}, flusherLoop, .{self});
        thread.detach();
    }

    fn flusherLoop(self: *Batcher) void {
        while (true) {
            std.Thread.sleep(5 * std.time.ns_per_s);
            self.flush() catch |err| {
                std.debug.print("Error in flusher: {}\n", .{err});
            };
        }
    }
};

test "batcher" {
    const allocator = std.testing.allocator;

    const Context = struct {
        count: usize,
        fn send(ctx_opaque: *anyopaque, timestamp: []const u8, lines: []const []const u8) !void {
            const self: *@This() = @ptrCast(@alignCast(ctx_opaque));
            self.count += lines.len;
            _ = timestamp;
        }
    };
    var ctx = Context{ .count = 0 };

    var batcher = Batcher.init(allocator, &ctx, Context.send);
    defer batcher.deinit();

    try batcher.add("t1", "line1");
    try batcher.add("t1", "line2");
    try batcher.add("t2", "line3");

    try batcher.flush();

    try std.testing.expectEqual(@as(usize, 3), ctx.count);
}
