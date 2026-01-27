package main

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/angch/sentrylogmon/config"
	"github.com/angch/sentrylogmon/detectors"
	"github.com/angch/sentrylogmon/monitor"
	"github.com/angch/sentrylogmon/sources"
	"github.com/angch/sentrylogmon/sysstat"
	"github.com/getsentry/sentry-go"
)

func main() {
	// Parse flags and load config
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

	// Start System Stats Collector
	sysstatCollector := sysstat.New()
	go sysstatCollector.Run()

	// Start monitors
	var monitors []*monitor.Monitor
	for _, monCfg := range cfg.Monitors {
		var src sources.LogSource

		switch monCfg.Type {
		case "file":
			if monCfg.Path == "" {
				log.Printf("Skipping file monitor '%s': path is empty", monCfg.Name)
				continue
			}
			src = sources.NewFileSource(monCfg.Name, monCfg.Path)
		case "journalctl":
			src = sources.NewJournalctlSource(monCfg.Name, monCfg.Args)
		case "dmesg":
			src = sources.NewDmesgSource(monCfg.Name)
		case "command":
			parts := strings.Fields(monCfg.Args)
			if len(parts) > 0 {
				src = sources.NewCommandSource(monCfg.Name, parts[0], parts[1:]...)
			} else {
				log.Printf("Skipping command monitor '%s': command is empty", monCfg.Name)
				continue
			}
		default:
			log.Printf("Unknown monitor type: %s", monCfg.Type)
			continue
		}

		detectorFormat := determineDetectorFormat(monCfg)

		det, err := detectors.GetDetector(detectorFormat, monCfg.Pattern)
		if err != nil {
			log.Printf("Failed to create detector for monitor '%s': %v", monCfg.Name, err)
			continue
		}

		m, err := monitor.New(src, det, sysstatCollector, cfg.Verbose)
		if err != nil {
			log.Printf("Failed to create monitor '%s': %v", monCfg.Name, err)
			continue
		}
		m.StopOnEOF = cfg.OneShot
		monitors = append(monitors, m)
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
		}
	} else {
		sig := <-c
		if cfg.Verbose {
			log.Printf("Received signal %v, shutting down...", sig)
		}
	}

	// Clean up
	for _, m := range monitors {
		if err := m.Source.Close(); err != nil {
			log.Printf("Error closing source %s: %v", m.Source.Name(), err)
		}
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
	if monCfg.Type == "dmesg" {
		return "dmesg"
	}
	if detectors.IsKnownDetector(monCfg.Name) {
		return monCfg.Name
	}
	return "custom"
}
