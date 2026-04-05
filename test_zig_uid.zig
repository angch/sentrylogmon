const std = @import("std");
pub fn main() void {
    if (@import("builtin").os.tag == .windows) {
        std.debug.print("Windows lacks getuid\n", .{});
    } else {
        std.debug.print("UID: {d}\n", .{std.os.linux.getuid()});
    }
}
