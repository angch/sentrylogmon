package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/angch/sentrylogmon/config"
	"github.com/angch/sentrylogmon/detectors"
	"github.com/angch/sentrylogmon/monitor"
	"github.com/angch/sentrylogmon/sources"
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

	if len(cfg.Monitors) == 0 {
		log.Fatal("No monitors configured. Use --file, --dmesg, --journalctl, --command, or config file.")
	}

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

		// Use the monitor type as the detector format by default, unless pattern is provided
		// Wait, config doesn't have a separate "Format" field yet, just "Pattern".
		// But "Type" (file/journalctl/dmesg) often implies the format.
		// However, "file" could be nginx, syslog, or java.
		// So we should probably check if monCfg.Name (or a new field) matches a detector.
		// But let's stick to: use monCfg.Pattern if present (Custom).
		// If monCfg.Pattern is empty, maybe try to infer from monCfg.Type == "dmesg"?

		// Actually, I should probably add a 'Format' field to MonitorConfig, but strictly following the prompt I should enable using these modules.
		// Let's rely on monCfg.Pattern being passed to GetDetector as "custom" if it's set.
		// If monCfg.Pattern is empty, we can try to guess or use a default.

		// BETTER APPROACH:
		// If monCfg.Type is "dmesg", use dmesg detector?
		// But what if I want to monitor nginx log via "file" type? I need to tell it to use "nginx" detector.

		// I will modify Config to include "Format". But first let's see what I can do with existing config.
		// Existing config has Type and Pattern.

		// If I don't modify config struct, I can't express "use nginx detector on this file".
		// So I should modify Config struct.

		detectorFormat := monCfg.Format
		if detectorFormat == "" {
			// Fallback logic
			if monCfg.Type == "dmesg" {
				detectorFormat = "dmesg"
			} else {
				detectorFormat = "custom"
			}
		}

		det, err := detectors.GetDetector(detectorFormat, monCfg.Pattern)
		if err != nil {
			log.Printf("Failed to create detector for monitor '%s': %v", monCfg.Name, err)
			continue
		}

		m, err := monitor.New(src, det, cfg.Verbose)
		if err != nil {
			log.Printf("Failed to create monitor '%s': %v", monCfg.Name, err)
			continue
		}
		monitors = append(monitors, m)
	}

	if len(monitors) == 0 {
		log.Fatal("No valid monitors to start.")
	}

	for _, m := range monitors {
		go m.Start()
	}

	// Wait for signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	sig := <-c
	if cfg.Verbose {
		log.Printf("Received signal %v, shutting down...", sig)
	}

	// Clean up
	for _, m := range monitors {
		if err := m.Source.Close(); err != nil {
			log.Printf("Error closing source %s: %v", m.Source.Name(), err)
		}
	}
}
