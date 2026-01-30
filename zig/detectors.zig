const std = @import("std");

pub const DetectorType = enum {
    string,
    json,
};

pub const Detector = union(DetectorType) {
    string: StringDetector,
    json: JsonDetector,

    pub fn match(self: Detector, allocator: std.mem.Allocator, line: []const u8) bool {
        switch (self) {
            .string => |d| return d.match(line),
            .json => |d| return d.match(allocator, line),
        }
    }
};

pub const StringDetector = struct {
    pattern: []const u8,

    pub fn match(self: StringDetector, line: []const u8) bool {
        return std.ascii.indexOfIgnoreCase(line, self.pattern) != null;
    }
};

pub const JsonDetector = struct {
    key: []const u8,
    value_pattern: []const u8,

    pub fn init(pattern: []const u8) JsonDetector {
        // Pattern format: "key:value_pattern"
        // If no colon, treat entire pattern as key and look for any value?
        // Or default to matching "message" or similar?
        // Go implementation convention: key:regex.
        // If no colon, we might fall back to string matching or error.
        // For now, let's assume if no colon, we try to match "message" field or just behave like string detector?
        // Let's implement stricly "key:value". If no colon, key is pattern, value is empty (match existence)?
        // Let's go with: if no colon, key="message", value=pattern (common case).

        if (std.mem.indexOf(u8, pattern, ":")) |idx| {
            return JsonDetector{
                .key = pattern[0..idx],
                .value_pattern = pattern[idx + 1 ..],
            };
        } else {
             // Default to checking "message" field if no key specified, or fallback behavior.
             // But for safety/predictability, let's treat the whole string as key? No, that's weird.
             // Let's assume pattern is the value to search in "message" field.
             return JsonDetector{
                 .key = "message",
                 .value_pattern = pattern,
             };
        }
    }

    pub fn match(self: JsonDetector, allocator: std.mem.Allocator, line: []const u8) bool {
        // Parse JSON
        const parsed = std.json.parseFromSlice(std.json.Value, allocator, line, .{}) catch return false;
        defer parsed.deinit();

        const root = parsed.value;
        if (root != .object) return false;

        if (root.object.get(self.key)) |val| {
            switch (val) {
                .string => |s| {
                    return std.ascii.indexOfIgnoreCase(s, self.value_pattern) != null;
                },
                else => return false, // Only match strings for now
            }
        }

        return false;
    }
};

pub fn createDetector(allocator: std.mem.Allocator, format: ?[]const u8, pattern: []const u8) !Detector {
    _ = allocator; // Not used currently for creation, but might be needed if we allocated detector
    if (format) |fmt| {
        if (std.mem.eql(u8, fmt, "json")) {
            return Detector{ .json = JsonDetector.init(pattern) };
        }
    }
    return Detector{ .string = StringDetector{ .pattern = pattern } };
}

test "StringDetector" {
    const d = Detector{ .string = StringDetector{ .pattern = "error" } };
    const allocator = std.testing.allocator;
    try std.testing.expect(d.match(allocator, "This is an error"));
    try std.testing.expect(!d.match(allocator, "This is fine"));
}

test "JsonDetector" {
    const allocator = std.testing.allocator;

    // Case 1: key:value match
    const d1 = Detector{ .json = JsonDetector.init("level:error") };
    try std.testing.expect(d1.match(allocator, "{\"level\":\"error\",\"msg\":\"something\"}"));
    try std.testing.expect(d1.match(allocator, "{\"level\":\"Error\",\"msg\":\"something\"}")); // Case insensitive
    try std.testing.expect(!d1.match(allocator, "{\"level\":\"info\",\"msg\":\"something\"}"));

    // Case 2: key missing
    try std.testing.expect(!d1.match(allocator, "{\"msg\":\"error\"}"));

    // Case 3: Invalid JSON
    try std.testing.expect(!d1.match(allocator, "not json"));

    // Case 4: Default key (message)
    const d2 = Detector{ .json = JsonDetector.init("failed") };
    try std.testing.expect(d2.match(allocator, "{\"message\":\"task failed\"}"));
    try std.testing.expect(!d2.match(allocator, "{\"other\":\"failed\"}"));
}
