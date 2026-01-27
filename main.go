package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
)

var (
	dsn         = flag.String("dsn", os.Getenv("SENTRY_DSN"), "Sentry DSN")
	useDmesg    = flag.Bool("dmesg", false, "Monitor dmesg output")
	inputFile   = flag.String("file", "", "Monitor a log file")
	pattern     = flag.String("pattern", "Error", "Pattern to match (case sensitive)")
	environment = flag.String("environment", "production", "Sentry environment")
	release     = flag.String("release", "", "Sentry release version")
	verbose     = flag.Bool("verbose", false, "Verbose logging")
)

func main() {
	flag.Parse()

	if *dsn == "" {
		log.Fatal("Sentry DSN is required. Set via --dsn flag or SENTRY_DSN environment variable")
	}

	// Initialize Sentry
	err := sentry.Init(sentry.ClientOptions{
		Dsn:         *dsn,
		Environment: *environment,
		Release:     *release,
	})
	if err != nil {
		log.Fatalf("Failed to initialize Sentry: %v", err)
	}
	defer sentry.Flush(2 * time.Second)

	if *verbose {
		log.Printf("Initialized Sentry with DSN (environment=%s, release=%s)", *environment, *release)
	}

	// Compile pattern
	patternRegex, err := regexp.Compile(*pattern)
	if err != nil {
		log.Fatalf("Failed to compile pattern: %v", err)
	}

	var reader io.Reader

	// Determine log source
	if *useDmesg {
		if *verbose {
			log.Println("Starting dmesg monitor...")
		}
		cmd := exec.Command("dmesg", "-w")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatalf("Failed to create dmesg pipe: %v", err)
		}
		if err := cmd.Start(); err != nil {
			log.Fatalf("Failed to start dmesg: %v", err)
		}
		defer cmd.Process.Kill()
		reader = stdout
	} else if *inputFile != "" {
		if *verbose {
			log.Printf("Monitoring file: %s", *inputFile)
		}
		file, err := os.Open(*inputFile)
		if err != nil {
			log.Fatalf("Failed to open file: %v", err)
		}
		defer file.Close()
		reader = file
	} else {
		log.Fatal("Please specify a log source: --dmesg or --file")
	}

	// Monitor logs
	monitor(reader, patternRegex)
}

// monitor reads log lines and groups by timestamp
func monitor(reader io.Reader, pattern *regexp.Regexp) {
	scanner := bufio.NewScanner(reader)
	
	// Map to group lines by timestamp
	timestampGroups := make(map[string][]string)
	timestampRegex := regexp.MustCompile(`^\[\s*([0-9.]+)\]`)

	for scanner.Scan() {
		line := scanner.Text()

		// Check if line matches pattern
		if !pattern.MatchString(line) {
			continue
		}

		if *verbose {
			log.Printf("Matched line: %s", line)
		}

		// Extract timestamp
		matches := timestampRegex.FindStringSubmatch(line)
		var timestamp string
		if len(matches) > 1 {
			timestamp = matches[1]
		} else {
			timestamp = "unknown"
		}

		// Group by timestamp
		timestampGroups[timestamp] = append(timestampGroups[timestamp], line)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading log: %v", err)
	}

	// Send grouped events to Sentry
	for timestamp, lines := range timestampGroups {
		sendToSentry(timestamp, lines)
	}
}

// sendToSentry sends grouped log lines to Sentry
func sendToSentry(timestamp string, lines []string) {
	message := fmt.Sprintf("Log errors at timestamp [%s]", timestamp)
	
	// Combine all lines for the event
	eventDetails := strings.Join(lines, "\n")

	if *verbose {
		log.Printf("Sending to Sentry: %d line(s) for timestamp %s", len(lines), timestamp)
	}

	// Send to Sentry using CaptureMessage
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetContext("log_lines", map[string]interface{}{
			"timestamp":   timestamp,
			"line_count":  len(lines),
			"lines":       eventDetails,
		})
		scope.SetTag("timestamp", timestamp)
		scope.SetTag("source", "dmesg")
		
		sentry.CaptureMessage(message)
	})

	if *verbose {
		log.Printf("Sent event to Sentry: %s", message)
	}
}
