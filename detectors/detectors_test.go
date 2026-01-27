package detectors

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectorsWithTestData(t *testing.T) {
	testDataDir := "../testdata"

	// Read directories in testdata
	entries, err := os.ReadDir(testDataDir)
	if err != nil {
		t.Fatalf("Failed to read testdata directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		detectorName := entry.Name()
		t.Run(detectorName, func(t *testing.T) {
			dirPath := filepath.Join(testDataDir, detectorName)
			files, err := os.ReadDir(dirPath)
			if err != nil {
				t.Fatalf("Failed to read directory %s: %v", dirPath, err)
			}

			for _, file := range files {
				if file.IsDir() || strings.HasSuffix(file.Name(), ".expect.txt") {
					continue
				}

				if !strings.HasSuffix(file.Name(), ".txt") {
					continue
				}

				inputFilename := file.Name()
				// Construct expect filename: foo.txt -> foo.expect.txt
				baseName := strings.TrimSuffix(inputFilename, filepath.Ext(inputFilename))
				expectFilename := baseName + ".expect.txt"

				t.Run(inputFilename, func(t *testing.T) {
					// Create detector for each file to ensure fresh state
					detector, err := GetDetector(detectorName, "")
					if err != nil {
						t.Fatalf("Failed to get detector for %s: %v", detectorName, err)
					}

					inputPath := filepath.Join(dirPath, inputFilename)
					expectPath := filepath.Join(dirPath, expectFilename)

					// Read expected lines
					expectedLines := readLines(t, expectPath)

					// Process input
					inputFile, err := os.Open(inputPath)
					if err != nil {
						t.Fatalf("Failed to open input file %s: %v", inputPath, err)
					}
					defer inputFile.Close()

					var detectedLines []string
					scanner := bufio.NewScanner(inputFile)
					for scanner.Scan() {
						lineBytes := scanner.Bytes()
						if detector.Detect(lineBytes) {
							detectedLines = append(detectedLines, string(lineBytes))
						}
					}

					// Verify
					if len(detectedLines) != len(expectedLines) {
						t.Errorf("Expected %d detected lines, got %d", len(expectedLines), len(detectedLines))
						t.Logf("Expected:\n%s", strings.Join(expectedLines, "\n"))
						t.Logf("Got:\n%s", strings.Join(detectedLines, "\n"))
					} else {
						for i := 0; i < len(detectedLines); i++ {
							if detectedLines[i] != expectedLines[i] {
								t.Errorf("Mismatch at line %d:\nExpected: %s\nGot:      %s", i+1, expectedLines[i], detectedLines[i])
							}
						}
					}
				})
			}
		})
	}
}

func readLines(t *testing.T, path string) []string {
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Failed to open file %s: %v", path, err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func TestIsKnownDetector(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"KnownDmesg", "dmesg", true},
		{"KnownNginx", "nginx", true},
		{"KnownNginxError", "nginx-error", true},
		{"UnknownFoo", "foo", false},
		{"UnknownEmpty", "", false},
		{"UnknownCase", "Nginx", false}, // Currently case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsKnownDetector(tt.input); got != tt.expected {
				t.Errorf("IsKnownDetector(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
