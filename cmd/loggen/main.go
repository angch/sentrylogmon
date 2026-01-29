package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

var (
	sizeFlag   = flag.String("size", "100MB", "Total size to generate (e.g., 100MB, 1GB)")
	formatFlag = flag.String("format", "nginx", "Log format: nginx, dmesg")
	errorRate  = flag.Float64("error-rate", 1.0, "Percentage of error logs (0-100)")
)

func main() {
	flag.Parse()

	targetSize := parseSize(*sizeFlag)
	if targetSize <= 0 {
		fmt.Fprintf(os.Stderr, "Invalid size: %s\n", *sizeFlag)
		os.Exit(1)
	}

	var generator func() string
	switch *formatFlag {
	case "nginx":
		generator = generateNginxLog
	case "nginx-error":
		generator = generateNginxErrorLog
	case "dmesg":
		generator = generateDmesgLog
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s\n", *formatFlag)
		os.Exit(1)
	}

	var generated int64
	for generated < targetSize {
		line := generator()
		n, err := fmt.Println(line)
		if err != nil {
			break
		}
		generated += int64(n)
	}
}

func parseSize(s string) int64 {
	var val int64
	// Try to handle units. Simplistic approach.
	// If it ends with MB, KB, GB
	s = strings.ToUpper(s)
	unit := ""
	if strings.HasSuffix(s, "MB") {
		unit = "MB"
		s = strings.TrimSuffix(s, "MB")
	} else if strings.HasSuffix(s, "KB") {
		unit = "KB"
		s = strings.TrimSuffix(s, "KB")
	} else if strings.HasSuffix(s, "GB") {
		unit = "GB"
		s = strings.TrimSuffix(s, "GB")
	}

	fmt.Sscanf(s, "%d", &val)

	switch unit {
	case "KB":
		return val * 1024
	case "MB":
		return val * 1024 * 1024
	case "GB":
		return val * 1024 * 1024 * 1024
	default:
		return val
	}
}

var (
	nginxLevels = []string{"info", "warn", "error", "crit", "alert", "emerg"}
	dmesgLevels = []string{"info", "warn", "error", "fail", "panic", "exception"}
	httpMethods = []string{"GET", "POST", "PUT", "DELETE", "HEAD"}
	paths       = []string{"/api/v1/users", "/index.html", "/login", "/static/style.css", "/images/logo.png"}
	agents      = []string{"Mozilla/5.0", "curl/7.64.1", "Googlebot/2.1"}
	messages    = []string{
		"Connection timed out",
		"File not found",
		"Permission denied",
		"Invalid argument",
		"Segmentation fault",
		"Disk quota exceeded",
		"Broken pipe",
	}
)

func shouldError() bool {
	return rand.Float64()*100 < *errorRate
}

func generateNginxLog() string {
	// Format: YYYY/MM/DD HH:MM:SS [level] 12345#0: *123 message, client: 1.2.3.4, server: example.com, request: "GET / HTTP/1.1", host: "example.com"

	ts := time.Now().Format("2006/01/02 15:04:05")
	level := "info"
	if shouldError() {
		// Pick an error level
		idx := 2 + rand.Intn(len(nginxLevels)-2) // start from error
		level = nginxLevels[idx]
	} else {
		// Pick info or warn
		level = nginxLevels[rand.Intn(2)]
	}

	msg := messages[rand.Intn(len(messages))]
	client := fmt.Sprintf("%d.%d.%d.%d", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256))
	method := httpMethods[rand.Intn(len(httpMethods))]
	path := paths[rand.Intn(len(paths))]

	return fmt.Sprintf("%s [%s] %d#0: *%d %s, client: %s, server: example.com, request: \"%s %s HTTP/1.1\"",
		ts, level, rand.Intn(10000), rand.Intn(100000), msg, client, method, path)
}

func generateDmesgLog() string {
	// Format: [TIMESTAMP] source: message
	// Or context lines

	ts := fmt.Sprintf("[%.6f]", float64(time.Now().Unix())+rand.Float64())

	if rand.Float64() < 0.1 {
		// Continuation line (stack trace or hex dump)
		return fmt.Sprintf(" %08x: %08x %08x %08x %08x", rand.Intn(0xFFFFFFFF), rand.Intn(0xFFFFFFFF), rand.Intn(0xFFFFFFFF), rand.Intn(0xFFFFFFFF), rand.Intn(0xFFFFFFFF))
	}

	source := fmt.Sprintf("dev%d", rand.Intn(10))
	msg := messages[rand.Intn(len(messages))]

	if shouldError() {
		// Add an error keyword
		kw := dmesgLevels[2+rand.Intn(len(dmesgLevels)-2)]
		msg = fmt.Sprintf("%s: %s", kw, msg)
	}

	return fmt.Sprintf("%s %s: %s", ts, source, msg)
}

func generateNginxErrorLog() string {
	// Format: YYYY/MM/DD HH:MM:SS [error] PID#PID: *ID connect() failed (ERRNO: MSG) while connecting to upstream, client: IP, server: HOST, request: "METHOD PATH PROTO", upstream: "URL", host: "HOST"

	ts := time.Now().Format("2006/01/02 15:04:05")
	pid := rand.Intn(30000)
	id := rand.Intn(100000000)

	// Always error for this format
	level := "error"

	msg := "connect() failed (113: No route to host)"

	client := fmt.Sprintf("%d.%d.%d.%d", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256))
	path := paths[rand.Intn(len(paths))]
	method := httpMethods[rand.Intn(len(httpMethods))]

	// upstream: "http://10.3.0.209:80..."
	upstreamIP := fmt.Sprintf("10.%d.%d.%d", rand.Intn(256), rand.Intn(256), rand.Intn(256))
	upstream := fmt.Sprintf("http://%s:80%s", upstreamIP, path)

	return fmt.Sprintf("%s [%s] %d#%d: *%d %s while connecting to upstream, client: %s, server: example.com, request: \"%s %s HTTP/1.1\", upstream: \"%s\", host: \"example.com\"",
		ts, level, pid, pid, id, msg, client, method, path, upstream)
}
