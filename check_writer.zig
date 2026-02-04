const std = @import("std");
pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    const allocator = gpa.allocator();
    var list = std.ArrayList(u8).empty;
    const w = list.writer(allocator);
    try w.writeAll("hello");
    std.debug.print("{s}\n", .{list.items});
}
