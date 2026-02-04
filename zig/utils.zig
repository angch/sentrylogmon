const std = @import("std");

pub fn containsPattern(haystack: []const u8, needle: []const u8) bool {
    return std.ascii.indexOfIgnoreCase(haystack, needle) != null;
}

pub fn extractTimestamp(line: []const u8) []const u8 {
    // 1. Dmesg style: [123.456]
    if (std.mem.indexOf(u8, line, "[")) |start| {
        if (std.mem.indexOf(u8, line[start..], "]")) |end| {
            // Ensure it looks like a timestamp (digits and dot)
            const content = line[start + 1 .. start + end];
            if (content.len > 0) {
                 return content;
            }
        }
    }

    // 2. Syslog RFC 3164: Mmm dd hh:mm:ss (e.g., "Oct 11 22:14:15")
    // Length is 15 chars.
    // We assume the timestamp is at the beginning of the line or close to it.
    if (line.len >= 15) {
        // Check for "Mmm " pattern
        // Simple heuristic: 3rd char is space, 6th is space, 9th is :, 12th is :
        if (line[3] == ' ' and line[6] == ' ' and line[9] == ':' and line[12] == ':') {
            return line[0..15];
        }
    }

    // 3. Syslog RFC 5424 / ISO8601: YYYY-MM-DDThh:mm:ss...
    if (line.len >= 19) {
        // Check for YYYY-MM-DD
        if (line[4] == '-' and line[7] == '-') {
             // Find end of timestamp (space or Z)
             if (std.mem.indexOfAny(u8, line, " Z")) |end| {
                 return line[0..end];
             }
             // Just take first 19 chars if nothing else
             return line[0..19];
        }
    }

    return "unknown";
}

test "containsPattern" {
    const haystack = "Hello World";
    try std.testing.expect(containsPattern(haystack, "world"));
    try std.testing.expect(containsPattern(haystack, "HELLO"));
    try std.testing.expect(!containsPattern(haystack, "foo"));
}

test "extractTimestamp" {
    // Dmesg
    try std.testing.expectEqualStrings("123.456", extractTimestamp("[123.456] Some message"));

    // Syslog RFC 3164
    try std.testing.expectEqualStrings("Oct 11 22:14:15", extractTimestamp("Oct 11 22:14:15 myhost myapp: message"));

    // ISO8601
    try std.testing.expectEqualStrings("2023-10-11T22:14:15", extractTimestamp("2023-10-11T22:14:15Z myhost message"));

    // Unknown
    try std.testing.expectEqualStrings("unknown", extractTimestamp("Just a message"));
}
