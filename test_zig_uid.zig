const std = @import("std");

pub fn main() void {
    if (@import("builtin").os.tag != .windows) {
        const uid = std.posix.getuid();
        std.debug.print("UID: {d}\n", .{uid});
    }
}
