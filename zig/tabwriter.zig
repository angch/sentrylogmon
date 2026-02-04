const std = @import("std");

/// A simplified TabWriter implementation similar to Go's text/tabwriter.
/// It buffers input, calculates column widths, and prints aligned output.
pub fn TabWriter(comptime WriterType: type) type {
    return struct {
        allocator: std.mem.Allocator,
        rows: std.ArrayList(std.ArrayList([]const u8)),
        padding: usize,
        writer: WriterType,

        const Self = @This();

        pub fn init(allocator: std.mem.Allocator, writer: WriterType) Self {
            return Self{
                .allocator = allocator,
                .rows = std.ArrayList(std.ArrayList([]const u8)).empty,
                .padding = 2, // Default padding of 2 spaces
                .writer = writer,
            };
        }

        pub fn deinit(self: *Self) void {
            for (self.rows.items) |*row| {
                for (row.items) |cell| {
                    self.allocator.free(cell);
                }
                row.deinit(self.allocator);
            }
            self.rows.deinit(self.allocator);
        }

        /// Formats string and adds to buffer. Splits by '\n' for rows and '\t' for columns.
        pub fn print(self: *Self, comptime fmt: []const u8, args: anytype) !void {
            const str = try std.fmt.allocPrint(self.allocator, fmt, args);
            defer self.allocator.free(str);

            var line_iter = std.mem.splitScalar(u8, str, '\n');
            while (line_iter.next()) |line| {
                // std.mem.splitScalar yields empty string at end if string ends with delimiter.
                // e.g. "a\n" -> "a", ""
                // We typically want to ignore that last empty line if it came from a trailing newline.
                // But we should be careful about "a\n\n" -> "a", "", "" (middle empty line is valid).
                // For simplified status output, skipping empty lines at the end is fine.
                if (line.len == 0 and line_iter.peek() == null) continue;

                try self.addRow(line);
            }
        }

        fn addRow(self: *Self, line: []const u8) !void {
            var row = std.ArrayList([]const u8).empty;
            errdefer row.deinit(self.allocator);

            var col_iter = std.mem.splitScalar(u8, line, '\t');
            while (col_iter.next()) |col| {
                const cell = try self.allocator.dupe(u8, col);
                try row.append(self.allocator, cell);
            }
            try self.rows.append(self.allocator, row);
        }

        /// Flush buffered data to the underlying writer with alignment.
        pub fn flush(self: *Self) !void {
            if (self.rows.items.len == 0) return;

            // Calculate max widths
            var max_widths = std.ArrayList(usize).empty;
            defer max_widths.deinit(self.allocator);

            // Find max columns
            var max_cols: usize = 0;
            for (self.rows.items) |row| {
                if (row.items.len > max_cols) max_cols = row.items.len;
            }

            try max_widths.appendNTimes(self.allocator, 0, max_cols);

            for (self.rows.items) |row| {
                for (row.items, 0..) |cell, i| {
                    // Check for unicode length vs byte length?
                    // For now assume ASCII/UTF-8 byte length is approximation, or use std.unicode
                    // Go's tabwriter handles runes. Zig's string is bytes.
                    // For PID, STARTED, etc, it's mostly ASCII.
                    // For parity with current implementation (byte slices), len is fine.
                    if (cell.len > max_widths.items[i]) {
                        max_widths.items[i] = cell.len;
                    }
                }
            }

            // Write
            for (self.rows.items) |row| {
                for (row.items, 0..) |cell, i| {
                    // Access interface explicitly if needed, but writer usually handles it via dot syntax
                    // in newer Zig versions if it's the Writer type.
                    // However, we are using WriterType which might be the struct (e.g. Writer(Context...)).
                    // Standard Writer struct has writeAll.
                    try self.writer.writeAll(cell);

                    // Add padding if not the last column
                    if (i < row.items.len - 1) {
                        const width = max_widths.items[i];
                        const pad = width - cell.len + self.padding;
                        var p: usize = 0;
                        while (p < pad) : (p += 1) {
                            try self.writer.writeByte(' ');
                        }
                    }
                }
                try self.writer.writeByte('\n');
            }

            // Clear rows after flush? Or user calls deinit?
            // Usually flush implies emptying the buffer.
            for (self.rows.items) |*row| {
                for (row.items) |cell| {
                    self.allocator.free(cell);
                }
                row.deinit(self.allocator);
            }
            self.rows.clearRetainingCapacity();
        }
    };
}

test "TabWriter basic alignment" {
    const allocator = std.testing.allocator;
    var out_buf = std.ArrayList(u8).empty;
    defer out_buf.deinit(allocator);

    var tw = TabWriter(std.ArrayList(u8).Writer).init(allocator, out_buf.writer(allocator));
    defer tw.deinit();

    try tw.print("Name\tAge\tCity\n", .{});
    try tw.print("Alice\t30\tNew York\n", .{});
    try tw.print("Bob\t25\tLos Angeles\n", .{});
    try tw.flush();

    const expected =
        "Name   Age  City\n" ++
        "Alice  30   New York\n" ++
        "Bob    25   Los Angeles\n";

    try std.testing.expectEqualStrings(expected, out_buf.items);
}

test "TabWriter padding" {
    const allocator = std.testing.allocator;
    var out_buf = std.ArrayList(u8).empty;
    defer out_buf.deinit(allocator);

    var tw = TabWriter(std.ArrayList(u8).Writer).init(allocator, out_buf.writer(allocator));
    defer tw.deinit();
    tw.padding = 4;

    try tw.print("A\tB\n", .{});
    try tw.print("AA\tBB\n", .{});
    try tw.flush();

    const expected =
        "A     B\n" ++
        "AA    BB\n";

    try std.testing.expectEqualStrings(expected, out_buf.items);
}
