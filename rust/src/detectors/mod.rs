use anyhow::Result;
use regex::Regex;
use std::sync::Mutex;

pub trait Detector: Send + Sync {
    /// Detect returns true if the line contains an issue
    fn detect(&self, line: &[u8]) -> bool;
}

pub struct GenericDetector {
    pattern: Regex,
}

impl GenericDetector {
    pub fn new(pattern: &str) -> Result<Self> {
        let regex = Regex::new(pattern)?;
        Ok(Self { pattern: regex })
    }
}

impl Detector for GenericDetector {
    fn detect(&self, line: &[u8]) -> bool {
        if let Ok(s) = std::str::from_utf8(line) {
            self.pattern.is_match(s)
        } else {
            false
        }
    }
}

pub struct NginxDetector {
    detector: GenericDetector,
}

impl NginxDetector {
    pub fn new() -> Result<Self> {
        let detector = GenericDetector::new("(?i)(error|critical|crit|alert|emerg)")?;
        Ok(Self { detector })
    }
}

impl Detector for NginxDetector {
    fn detect(&self, line: &[u8]) -> bool {
        self.detector.detect(line)
    }
}

struct DmesgState {
    last_match_time: f64,
    last_match_header: String,
}

pub struct DmesgDetector {
    detector: GenericDetector,
    state: Mutex<DmesgState>,
    dmesg_line_regex: Regex,
    dmesg_start_regex: Regex,
}

impl DmesgDetector {
    pub fn new() -> Result<Self> {
        // Added "exception" to the pattern
        let detector = GenericDetector::new("(?i)(error|fail|panic|oops|exception)")?;
        // Example: [787739.009553] ata1.00: exception Emask...
        let dmesg_line_regex = Regex::new(r"^\[\s*(\d+\.\d+)\]\s*([^:]+):")?;
        // Example: [ 123.456] ...
        let dmesg_start_regex = Regex::new(r"^\[\s*\d+\.\d+\]")?;

        Ok(Self {
            detector,
            state: Mutex::new(DmesgState {
                last_match_time: 0.0,
                last_match_header: String::new(),
            }),
            dmesg_line_regex,
            dmesg_start_regex,
        })
    }
}

impl Detector for DmesgDetector {
    fn detect(&self, line: &[u8]) -> bool {
         let line_str = match std::str::from_utf8(line) {
            Ok(s) => s,
            Err(_) => return false,
        };

        // 1. Check if it matches the error pattern first
        let is_error = self.detector.detect(line);

        // 2. Check if it looks like a new dmesg line (starts with timestamp)
        let is_dmesg_line = self.dmesg_start_regex.is_match(line_str);

        // 3. Parse the line for detailed info
        let mut timestamp = 0.0;
        let mut header = String::new();

        if let Some(caps) = self.dmesg_line_regex.captures(line_str) {
             if let Some(t_match) = caps.get(1) {
                 if let Ok(t) = t_match.as_str().parse::<f64>() {
                     timestamp = t;
                 }
             }
             if let Some(h_match) = caps.get(2) {
                 header = h_match.as_str().to_string();
             }
        }

        let mut state = self.state.lock().unwrap();

        if is_error {
            // Update state
            if timestamp > 0.0 {
                state.last_match_time = timestamp;
            }
            if !header.is_empty() {
                state.last_match_header = header;
            }
            return true;
        }

        // 4. If not an explicit error, check if it's related context
        if !state.last_match_header.is_empty() {
             if is_dmesg_line {
                 // It's a new log line. Check if it's related.
                 if !header.is_empty() && timestamp > 0.0 {
                     if (timestamp - state.last_match_time) <= 5.0 {
                         if are_headers_related(&state.last_match_header, &header) {
                             return true;
                         }
                     }
                 }
                 // Not related
                 return false;
             } else {
                 // Continuation line
                 return true;
             }
        }

        false
    }
}

fn are_headers_related(h1: &str, h2: &str) -> bool {
    let h1 = h1.trim();
    let h2 = h2.trim();
    if h1.is_empty() || h2.is_empty() {
        return false;
    }
    if h1 == h2 {
        return true;
    }
    if h1.starts_with(h2) || h2.starts_with(h1) {
        return true;
    }
    false
}

pub fn get_detector(format: &str, pattern: &str) -> Result<Box<dyn Detector>> {
    if !pattern.is_empty() {
        return Ok(Box::new(GenericDetector::new(pattern)?));
    }

    match format {
        "nginx" => Ok(Box::new(NginxDetector::new()?)),
        "dmesg" => Ok(Box::new(DmesgDetector::new()?)),
        _ => Ok(Box::new(GenericDetector::new("(?i)error")?)),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use std::path::Path;
    use std::io::BufRead;

    #[test]
    fn test_detectors_with_test_data() -> Result<()> {
        // Try to find testdata relative to CARGO_MANIFEST_DIR
        let manifest_dir = std::env::var("CARGO_MANIFEST_DIR")?;
        let test_data_dir = Path::new(&manifest_dir).parent().unwrap().join("testdata");

        if !test_data_dir.exists() {
            panic!("Could not find testdata directory at {:?}", test_data_dir);
        }

        run_tests(&test_data_dir)
    }

    fn run_tests(test_data_dir: &Path) -> Result<()> {
        for entry in fs::read_dir(test_data_dir)? {
            let entry = entry?;
            let path = entry.path();
            if !path.is_dir() {
                continue;
            }

            let detector_name = path.file_name().unwrap().to_string_lossy().to_string();

            // Skip directories that are not detectors we know about (though we should support all in testdata)
            // But for now we only have nginx and dmesg
            if detector_name != "nginx" && detector_name != "dmesg" {
                 continue;
            }

            let detector_dir = path;

            for file in fs::read_dir(detector_dir)? {
                let file = file?;
                let file_path = file.path();

                if !file_path.is_file() {
                    continue;
                }

                let file_name = file_path.file_name().unwrap().to_string_lossy();
                if !file_name.ends_with(".txt") || file_name.ends_with(".expect.txt") {
                    continue;
                }

                let base_name = file_name.trim_end_matches(".txt");
                let expect_filename = format!("{}.expect.txt", base_name);
                let expect_path = file_path.parent().unwrap().join(expect_filename);

                if !expect_path.exists() {
                    continue;
                }

                println!("Testing {}/{}", detector_name, file_name);

                // Create detector
                let detector = get_detector(&detector_name, "")?;

                // Read expected lines
                let expected_lines: Vec<String> = fs::read_to_string(&expect_path)?
                    .lines()
                    .map(|s| s.to_string())
                    .collect();

                // Process input
                let input_file = fs::File::open(&file_path)?;
                let reader = std::io::BufReader::new(input_file);
                let mut detected_lines = Vec::new();

                for line in reader.lines() {
                    let line = line?;
                    if detector.detect(line.as_bytes()) {
                        detected_lines.push(line);
                    }
                }

                // Verify
                assert_eq!(
                    detected_lines.len(),
                    expected_lines.len(),
                    "Mismatch in number of lines for {}/{}",
                    detector_name, file_name
                );

                for (i, (got, want)) in detected_lines.iter().zip(expected_lines.iter()).enumerate() {
                     assert_eq!(got, want, "Mismatch at line {} in {}/{}", i + 1, detector_name, file_name);
                }
            }
        }
        Ok(())
    }
}
