const std = @import("std");
pub fn main() void {
    const list = std.ArrayList(u8).empty;
    _ = list;
}
