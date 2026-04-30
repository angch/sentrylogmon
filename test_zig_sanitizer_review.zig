const std = @import("std");

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    const allocator = gpa.allocator();

    const args1 = [_][]const u8{"myapp", "--PASSWORD", "mysecret"};
    const sanitized1 = try sanitizeCommand(allocator, &args1);
    std.debug.print("Sanitized1: {s}\n", .{sanitized1});
    allocator.free(sanitized1);
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
        "password", "token", "secret", "_key", "-key", ".key", "signature", "credential", "cookie", "session",
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

            const lower_full_key = try std.ascii.allocLowerString(allocator, key);
            defer allocator.free(lower_full_key);

            if (sensitive or sensitive_flags.has(lower_full_key)) {
                try out.appendSlice(allocator, key);
                try out.appendSlice(allocator, "=[REDACTED]");
                continue;
            }

            try out.appendSlice(allocator, arg);
            continue;
        }

        const lower_arg = try std.ascii.allocLowerString(allocator, arg);
        defer allocator.free(lower_arg);

        if (sensitive_flags.has(lower_arg)) {
            try out.appendSlice(allocator, arg);
            skip_next = true;
            continue;
        }

        const clean_arg = std.mem.trimLeft(u8, lower_arg, "-");
        var heuristic_match = false;
        if (std.mem.eql(u8, clean_arg, "password") or
            std.mem.eql(u8, clean_arg, "token") or
            std.mem.eql(u8, clean_arg, "secret") or
            std.mem.eql(u8, clean_arg, "key") or
            std.mem.eql(u8, clean_arg, "auth")) {
            heuristic_match = true;
        } else {
             for (sensitive_suffixes) |suffix| {
                 if (std.mem.endsWith(u8, clean_arg, suffix)) {
                     if (clean_arg.len == suffix.len) {
                         heuristic_match = true;
                         break;
                     }
                     if (suffix[0] == '-' or suffix[0] == '_' or suffix[0] == '.') {
                         heuristic_match = true;
                         break;
                     }
                     const match_idx = clean_arg.len - suffix.len;
                     if (match_idx > 0) {
                         const char_before = clean_arg[match_idx - 1];
                         if (char_before == '-' or char_before == '_' or char_before == '.') {
                             heuristic_match = true;
                             break;
                         }
                     }
                 }
             }
        }

        if (heuristic_match) {
            try out.appendSlice(allocator, arg);
            if (i + 1 < args.len and !(args[i + 1].len > 0 and args[i + 1][0] == '-')) {
                skip_next = true;
            }
            continue;
        }

        try out.appendSlice(allocator, arg);
    }

    return out.toOwnedSlice(allocator);
}
