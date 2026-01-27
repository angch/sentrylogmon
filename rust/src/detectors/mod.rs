use anyhow::Result;
use regex::Regex;

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

pub fn get_detector(format: &str, pattern: &str) -> Result<Box<dyn Detector>> {
    // For simplicity, we only implement generic detector
    // Could add specialized detectors for nginx, dmesg, etc. like the Go version
    let pattern_to_use = if !pattern.is_empty() {
        pattern
    } else if format == "dmesg" {
        "(?i)(error|fail|panic|oops)"
    } else {
        "(?i)error"
    };

    Ok(Box::new(GenericDetector::new(pattern_to_use)?))
}
