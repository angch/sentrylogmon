const std = @import("std");

pub const Queue = struct {
    dir_path: []const u8,

    pub fn init(dir_path: []const u8) !void {
        std.fs.cwd().makeDir(dir_path) catch |err| {
            if (err != error.PathAlreadyExists) return err;
        };
    }

    pub fn push(self: Queue, allocator: std.mem.Allocator, payload: []const u8) !void {
        const timestamp = std.time.timestamp();

        var seed: [8]u8 = undefined;
        try std.posix.getrandom(&seed);
        const seed_int = std.mem.readInt(u64, &seed, .little);
        var prng = std.Random.DefaultPrng.init(seed_int);
        const random_val = prng.random().int(u32);

        const filename = try std.fmt.allocPrint(allocator, "{s}/event_{}_{}.json", .{ self.dir_path, timestamp, random_val });
        defer allocator.free(filename);

        const file = try std.fs.cwd().createFile(filename, .{});
        defer file.close();

        try file.writeAll(payload);
    }

    pub fn retryAll(self: Queue, allocator: std.mem.Allocator, context: anytype, sendFn: fn (@TypeOf(context), []const u8) anyerror!void) !void {
        var dir = std.fs.cwd().openDir(self.dir_path, .{ .iterate = true }) catch |err| {
             // If dir doesn't exist, nothing to retry
             if (err == error.FileNotFound) return;
             return err;
        };
        defer dir.close();

        var iter = dir.iterate();
        while (try iter.next()) |entry| {
            if (entry.kind == .file and std.mem.startsWith(u8, entry.name, "event_")) {
                const filepath = try std.fmt.allocPrint(allocator, "{s}/{s}", .{ self.dir_path, entry.name });
                defer allocator.free(filepath);

                // Read file
                const file = std.fs.cwd().openFile(filepath, .{}) catch continue;
                // Defer close inside block? No, better explicit close.

                const stat = file.stat() catch { file.close(); continue; };
                const content = file.readToEndAlloc(allocator, stat.size) catch { file.close(); continue; };
                file.close(); // Close before sending so we can delete if successful
                defer allocator.free(content);

                // Try to send
                sendFn(context, content) catch |err| {
                     std.debug.print("Failed to retry {s}: {}\n", .{entry.name, err});
                     continue;
                };

                // If successful, delete file
                std.fs.cwd().deleteFile(filepath) catch |err| {
                    std.debug.print("Failed to delete {s}: {}\n", .{filepath, err});
                };
            }
        }
    }
};

test "queue operations" {
    const allocator = std.testing.allocator;
    const test_dir = "test_queue";

    // Cleanup before test
    std.fs.cwd().deleteTree(test_dir) catch {};
    defer std.fs.cwd().deleteTree(test_dir) catch {};

    const queue = Queue{ .dir_path = test_dir };
    try Queue.init(test_dir);

    const payload = "{\"test\": 123}";
    try queue.push(allocator, payload);

    // Verify file exists
    var dir = try std.fs.cwd().openDir(test_dir, .{ .iterate = true });
    var iter = dir.iterate();
    var found = false;
    while (try iter.next()) |entry| {
        if (std.mem.startsWith(u8, entry.name, "event_")) {
            found = true;
            break;
        }
    }
    dir.close();
    try std.testing.expect(found);

    // Test retry
    const Context = struct {
        received: bool,
        fn send(self: *@This(), data: []const u8) !void {
            if (std.mem.eql(u8, data, "{\"test\": 123}")) {
                self.received = true;
            } else {
                return error.Mismatch;
            }
        }
    };
    var ctx = Context{ .received = false };

    try queue.retryAll(allocator, &ctx, Context.send);

    try std.testing.expect(ctx.received);

    // Verify file deleted
    dir = try std.fs.cwd().openDir(test_dir, .{ .iterate = true });
    iter = dir.iterate();
    found = false;
    while (try iter.next()) |entry| {
        if (std.mem.startsWith(u8, entry.name, "event_")) {
            found = true;
            break;
        }
    }
    dir.close();
    try std.testing.expect(!found);
}
