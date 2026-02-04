package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // Register pprof handlers
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/angch/sentrylogmon/config"
	"github.com/angch/sentrylogmon/detectors"
	"github.com/angch/sentrylogmon/ipc"
	"github.com/angch/sentrylogmon/monitor"
	"github.com/angch/sentrylogmon/sources"
	"github.com/angch/sentrylogmon/sysstat"
	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	statusFlag = flag.Bool("status", false, "List running instances")
	updateFlag = flag.Bool("update", false, "Update/Restart all running instances")
)

func main() {
	// Ensure flags are parsed first to handle --status/--update without requiring full config
	config.ParseFlags()

	if *statusFlag {
		instances, err := ipc.ListInstances(ipc.GetSocketDir())
		if err != nil {
			log.Fatalf("Error listing instances: %v", err)
		}

		isTerminal := false
		if fi, err := os.Stdout.Stat(); err == nil {
			isTerminal = (fi.Mode() & os.ModeCharDevice) != 0
		}

		if isTerminal {
			printInstanceTable(instances)
		} else {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(instances)
		}
		return
	}

	if *updateFlag {
		instances, err := ipc.ListInstances(ipc.GetSocketDir())
		if err != nil {
			log.Fatalf("Error listing instances: %v", err)
		}
		for _, inst := range instances {
			socketPath := filepath.Join(ipc.GetSocketDir(), fmt.Sprintf("sentrylogmon.%d.sock", inst.PID))
			fmt.Printf("Requesting update for PID %d...\n", inst.PID)
			if err := ipc.RequestUpdate(socketPath); err != nil {
				fmt.Printf("Failed to update PID %d: %v\n", inst.PID, err)
			} else {
				fmt.Printf("Update requested for PID %d\n", inst.PID)
			}
		}
		return
	}

	// Load configuration after checking for IPC flags
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if cfg.Sentry.DSN == "" {
		log.Fatal("Sentry DSN is required. Set via --dsn flag, SENTRY_DSN environment variable, or config file")
	}

	// Initialize Sentry
	err = sentry.Init(sentry.ClientOptions{
		Dsn:         cfg.Sentry.DSN,
		Environment: cfg.Sentry.Environment,
		Release:     cfg.Sentry.Release,
	})
	if err != nil {
		log.Fatalf("Failed to initialize Sentry: %v", err)
	}
	defer sentry.Flush(2 * time.Second)

	if cfg.Verbose {
		log.Printf("Initialized Sentry (env=%s, release=%s)", cfg.Sentry.Environment, cfg.Sentry.Release)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cfg.Verbose || cfg.OneShot {
		defer func() {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			log.Printf("Final Memory Usage: Alloc = %v MiB, TotalAlloc = %v MiB, Sys = %v MiB, NumGC = %v",
				m.Alloc/1024/1024,
				m.TotalAlloc/1024/1024,
				m.Sys/1024/1024,
				m.NumGC,
			)
		}()
	}

	if len(cfg.Monitors) == 0 {
		log.Fatal("No monitors configured. Use --file, --dmesg, --journalctl, --command, or config file.")
	}

	if cfg.MetricsPort > 0 {
		go func() {
			addr := fmt.Sprintf(":%d", cfg.MetricsPort)
			if cfg.Verbose {
				log.Printf("Starting Prometheus metrics server on %s/metrics", addr)
			}
			http.Handle("/metrics", promhttp.Handler())
			http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})
			if err := http.ListenAndServe(addr, nil); err != nil {
				log.Printf("Failed to start metrics server: %v", err)
			}
		}()
	}

	// Start System Stats Collector
	sysstatCollector := sysstat.New()
	go sysstatCollector.Run()

	// Start monitors
	var monitors []*monitor.Monitor

	addMonitor := func(src sources.LogSource, monCfg config.MonitorConfig) {
		detectorFormat := determineDetectorFormat(monCfg)

		det, err := detectors.GetDetector(detectorFormat, monCfg.Pattern)
		if err != nil {
			log.Printf("Failed to create detector for monitor '%s': %v", monCfg.Name, err)
			return
		}

		// Prepare Sentry Options
		sentryDSN := monCfg.Sentry.DSN
		sentryEnv := monCfg.Sentry.Environment
		sentryRelease := monCfg.Sentry.Release

		// Inherit global config if DSN is overridden but other fields are missing
		if sentryDSN != "" {
			if sentryEnv == "" {
				sentryEnv = cfg.Sentry.Environment
			}
			if sentryRelease == "" {
				sentryRelease = cfg.Sentry.Release
			}
		}

		m, err := monitor.New(ctx, src, det, sysstatCollector, monitor.Options{
			Verbose:           cfg.Verbose,
			ExcludePattern:    monCfg.ExcludePattern,
			RateLimitBurst:    monCfg.RateLimitBurst,
			RateLimitWindow:   monCfg.RateLimitWindow,
			SentryDSN:         sentryDSN,
			SentryEnvironment: sentryEnv,
			SentryRelease:     sentryRelease,
		})
		if err != nil {
			log.Printf("Failed to create monitor '%s': %v", monCfg.Name, err)
			return
		}
		m.StopOnEOF = cfg.OneShot
		monitors = append(monitors, m)
	}

	for _, monCfg := range cfg.Monitors {
		switch monCfg.Type {
		case "file":
			if monCfg.Path == "" {
				log.Printf("Skipping file monitor '%s': path is empty", monCfg.Name)
				continue
			}

			if strings.ContainsAny(monCfg.Path, "*?[]") {
				matches, err := filepath.Glob(monCfg.Path)
				if err != nil {
					log.Printf("Error matching glob pattern %s: %v", monCfg.Path, err)
					continue
				}
				if len(matches) == 0 {
					log.Printf("No files matched glob pattern %s", monCfg.Path)
					continue
				}
				for _, match := range matches {
					// Use a unique name for each file source
					name := monCfg.Name + ":" + match
					src := sources.NewFileSource(name, match)
					addMonitor(src, monCfg)
				}
			} else {
				src := sources.NewFileSource(monCfg.Name, monCfg.Path)
				addMonitor(src, monCfg)
			}
		case "journalctl":
			src := sources.NewJournalctlSource(monCfg.Name, monCfg.Args)
			addMonitor(src, monCfg)
		case "dmesg":
			src := sources.NewDmesgSource(monCfg.Name)
			addMonitor(src, monCfg)
		case "command":
			parts := strings.Fields(monCfg.Args)
			if len(parts) > 0 {
				src := sources.NewCommandSource(monCfg.Name, parts[0], parts[1:]...)
				addMonitor(src, monCfg)
			} else {
				log.Printf("Skipping command monitor '%s': command is empty", monCfg.Name)
				continue
			}
		case "syslog":
			src := sources.NewSyslogSource(monCfg.Name, monCfg.Path)
			addMonitor(src, monCfg)
		default:
			log.Printf("Unknown monitor type: %s", monCfg.Type)
			continue
		}
	}

	if len(monitors) == 0 {
		log.Fatal("No valid monitors to start.")
	}

	var wg sync.WaitGroup
	for _, m := range monitors {
		wg.Add(1)
		go func(mon *monitor.Monitor) {
			defer wg.Done()
			mon.Start()
		}(m)
	}

	shutdown := func() {
		cancel()
		for _, m := range monitors {
			if err := m.Source.Close(); err != nil {
				log.Printf("Error closing source %s: %v", m.Source.Name(), err)
			}
		}

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			log.Println("Timeout waiting for monitors to stop")
		}
	}

	// Start IPC Server
	socketDir := ipc.GetSocketDir()
	var socketPath string
	var restartFunc func()

	if err := ipc.EnsureSecureDirectory(socketDir); err != nil {
		log.Printf("Failed to ensure secure IPC directory: %v", err)
	} else {
		socketPath = filepath.Join(socketDir, fmt.Sprintf("sentrylogmon.%d.sock", os.Getpid()))
		defer os.Remove(socketPath)
	}

	restartFunc = func() {
		log.Println("Restart requested. Shutting down...")
		shutdown()

		if socketPath != "" {
			os.Remove(socketPath)
		}

		executable, err := os.Executable()
		if err != nil {
			log.Printf("Failed to get executable path: %v", err)
			return
		}

		log.Printf("Re-executing %s %v", executable, os.Args[1:])
		if err := syscall.Exec(executable, os.Args, os.Environ()); err != nil {
			log.Fatalf("Failed to re-exec: %v", err)
		}
	}

	if socketPath != "" {
		go func() {
			if err := ipc.StartServer(socketPath, cfg, restartFunc); err != nil {
				log.Printf("IPC Server error: %v", err)
			}
		}()
	}

	// Start config watcher
	if f := flag.Lookup("config"); f != nil {
		configPath := f.Value.String()
		if configPath != "" {
			go watchConfig(ctx, configPath, restartFunc)
		}
	}

	// Wait for signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	if cfg.OneShot {
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			if cfg.Verbose {
				log.Println("All monitors finished.")
			}
		case sig := <-c:
			if cfg.Verbose {
				log.Printf("Received signal %v, shutting down...", sig)
			}
			shutdown()
		}
	} else {
		sig := <-c
		if cfg.Verbose {
			log.Printf("Received signal %v, shutting down...", sig)
		}
		shutdown()
	}
}

func determineDetectorFormat(monCfg config.MonitorConfig) string {
	if monCfg.Format != "" {
		return monCfg.Format
	}
	// If pattern is present, assume custom (GenericDetector).
	// This allows overriding the default dmesg detector for dmesg source if a custom pattern is provided.
	if monCfg.Pattern != "" {
		return "custom"
	}

	// Infer detector format from monitor type.
	// Specifically, 'dmesg' source type defaults to 'dmesg' detector format.
	if monCfg.Type == "dmesg" {
		return "dmesg"
	}

	// Infer detector format from monitor name if it matches a known detector (e.g. "nginx").
	if detectors.IsKnownDetector(monCfg.Name) {
		return monCfg.Name
	}
	return "custom"
}

func printInstanceTable(instances []ipc.StatusResponse) {
	if len(instances) == 0 {
		fmt.Println("No running instances found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "PID\tSTARTED\tUPTIME\tVERSION\tDETAILS")
	for _, inst := range instances {
		uptime := time.Since(inst.StartTime).Round(time.Second)
		uptimeStr := formatDuration(uptime)

		var details string
		if inst.Config != nil && len(inst.Config.Monitors) > 0 {
			const limit = 60
			var buffer strings.Builder
			monitors := inst.Config.Monitors

			for i, m := range monitors {
				part := fmt.Sprintf("%s(%s)", m.Name, m.Type)
				sep := ""
				if i > 0 {
					sep = ", "
				}

				// Handle first item special case to ensure it's always visible
				if i == 0 {
					remaining := len(monitors) - 1
					suffixLen := 0
					if remaining > 0 {
						suffixLen = 12 // Space for " (+NN more)"
					}

					if len(part)+suffixLen > limit {
						// Truncate first item if it's too long
						avail := limit - suffixLen - 3 // -3 for "..."
						if avail < 10 {
							avail = 10
						}
						if len(part) > avail {
							part = part[:avail] + "..."
						}
					}
					buffer.WriteString(part)
					continue
				}

				// Check subsequent items
				reserved := 12 // Space for " (+NN more)"
				if i == len(monitors)-1 {
					reserved = 0
				}

				if buffer.Len()+len(sep)+len(part)+reserved <= limit {
					buffer.WriteString(sep)
					buffer.WriteString(part)
				} else {
					remaining := len(monitors) - i
					fmt.Fprintf(&buffer, " (+%d more)", remaining)
					break
				}
			}
			details = buffer.String()
		}
		if details == "" {
			details = "-"
		}

		version := inst.Version
		if version == "" {
			version = "-"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", inst.PID, inst.StartTime.Format("2006-01-02 15:04:05"), uptimeStr, version, details)
	}
	w.Flush()
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	day := d / (24 * time.Hour)
	d -= day * 24 * time.Hour
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	if day > 0 {
		return fmt.Sprintf("%dd %dh %dm", day, h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
