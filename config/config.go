package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/angch/sentrylogmon/sysstat"
	"gopkg.in/yaml.v3"
)

type SentryConfig struct {
	DSN         string `yaml:"dsn"`
	Environment string `yaml:"environment"`
	Release     string `yaml:"release"`
}

type MonitorConfig struct {
	Name            string       `yaml:"name"`
	Type            string       `yaml:"type"`            // file, journalctl, dmesg, command
	Path            string       `yaml:"path"`            // for file
	Args            string       `yaml:"args"`            // for journalctl or command
	Pattern         string       `yaml:"pattern"`         // regex pattern for custom format
	Format          string       `yaml:"format"`          // dmesg, nginx, custom (default: custom if pattern set)
	ExcludePattern  string       `yaml:"exclude_pattern"` // regex pattern to exclude from reporting
	RateLimitBurst  int          `yaml:"rate_limit_burst"`
	RateLimitWindow string       `yaml:"rate_limit_window"`
	Sentry          SentryConfig `yaml:"sentry"` // Override global Sentry config
}

type Config struct {
	Sentry      SentryConfig    `yaml:"sentry"`
	Monitors    []MonitorConfig `yaml:"monitors"`
	Verbose     bool            `yaml:"-"`
	OneShot     bool            `yaml:"-"`
	MetricsPort int             `yaml:"metrics_port"`
}

var (
	configFile     = flag.String("config", "", "Path to configuration file")
	dsn            = flag.String("dsn", os.Getenv("SENTRY_DSN"), "Sentry DSN")
	useDmesg       = flag.Bool("dmesg", false, "Monitor dmesg output")
	inputFile      = flag.String("file", "", "Monitor a log file")
	journalctl     = flag.String("journalctl", "", "Monitor journalctl output (pass args)")
	command        = flag.String("command", "", "Monitor custom command output")
	syslogAddr     = flag.String("syslog", "", "Syslog address (e.g. udp:127.0.0.1:5514 or :5514)")
	format         = flag.String("format", "", "Detector format (dmesg, nginx, custom)")
	pattern        = flag.String("pattern", "Error", "Pattern to match (case sensitive)")
	excludePattern = flag.String("exclude", "", "Pattern to exclude from reporting (case sensitive)")
	environment    = flag.String("environment", "production", "Sentry environment")
	release        = flag.String("release", "", "Sentry release version")
	verbose        = flag.Bool("verbose", false, "Verbose logging")
	oneshot        = flag.Bool("oneshot", false, "Run once and exit when input stream ends")
	metricsPort    = flag.Int("metrics-port", 0, "Port to expose Prometheus metrics (0 to disable)")
)

// ParseFlags parses the command line flags.
// It must be called before Load.
func ParseFlags() {
	if !flag.Parsed() {
		flag.Usage = func() {
			out := flag.CommandLine.Output()
			fmt.Fprintf(out, "Sentry Log Monitor\n")
			fmt.Fprintf(out, "A lightweight tool to monitor logs and report errors to Sentry.\n\n")
			fmt.Fprintf(out, "Usage:\n  sentrylogmon [flags]\n\n")
			fmt.Fprintf(out, "Examples:\n")
			fmt.Fprintf(out, "  # Monitor a file for errors\n")
			fmt.Fprintf(out, "  sentrylogmon --dsn=https://... --file=/var/log/syslog\n\n")
			fmt.Fprintf(out, "  # Monitor with config file\n")
			fmt.Fprintf(out, "  sentrylogmon --config=sentrylogmon.yaml\n\n")
			fmt.Fprintf(out, "  # Monitor journalctl\n")
			fmt.Fprintf(out, "  sentrylogmon --dsn=... --journalctl=\"--unit=nginx -f\"\n\n")
			fmt.Fprintf(out, "  # Monitor syslog\n")
			fmt.Fprintf(out, "  sentrylogmon --dsn=... --syslog=:5514\n\n")
			fmt.Fprintf(out, "Flags:\n")
			flag.PrintDefaults()
		}
		flag.Parse()
	}
}

func Load() (*Config, error) {
	// Ensure flags are parsed
	ParseFlags()

	cfg := &Config{
		Verbose: *verbose,
		OneShot: *oneshot,
	}

	if *configFile != "" {
		if *verbose {
			log.Printf("Loading configuration from %s", *configFile)
		}
		data, err := os.ReadFile(*configFile)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}

		// Fallback to flags/env if missing in config
		if cfg.Sentry.DSN == "" {
			cfg.Sentry.DSN = *dsn
		}
		if cfg.Sentry.Environment == "" {
			cfg.Sentry.Environment = *environment
		}
		if cfg.Sentry.Release == "" {
			cfg.Sentry.Release = *release
		}

		// Flags override config file
		if *metricsPort != 0 {
			cfg.MetricsPort = *metricsPort
		}

		// Verbose flag always overrides
		cfg.Verbose = *verbose
		cfg.OneShot = *oneshot
		return cfg, nil
	}

	// Legacy/CLI mode
	cfg.Sentry = SentryConfig{
		DSN:         *dsn,
		Environment: *environment,
		Release:     *release,
	}

	cfg.MetricsPort = *metricsPort

	monitor := MonitorConfig{
		Pattern:        *pattern,
		ExcludePattern: *excludePattern,
		Format:         *format,
	}

	if *useDmesg {
		monitor.Name = "dmesg"
		monitor.Type = "dmesg"
	} else if *inputFile != "" {
		monitor.Name = "file"
		monitor.Type = "file"
		monitor.Path = *inputFile
	} else if *journalctl != "" {
		monitor.Name = "journalctl"
		monitor.Type = "journalctl"
		monitor.Args = *journalctl
	} else if *command != "" {
		monitor.Name = "command"
		monitor.Type = "command"
		monitor.Args = *command
	} else if *syslogAddr != "" {
		monitor.Name = "syslog"
		monitor.Type = "syslog"
		monitor.Path = *syslogAddr
	}

	if monitor.Type != "" {
		cfg.Monitors = append(cfg.Monitors, monitor)
	}

	return cfg, nil
}

// Redacted returns a deep copy of the configuration with sensitive fields redacted.
func (c *Config) Redacted() *Config {
	newC := *c

	// Deep copy monitors slice
	if c.Monitors != nil {
		newC.Monitors = make([]MonitorConfig, len(c.Monitors))
		copy(newC.Monitors, c.Monitors)
	}

	// Redact Global DSN
	if newC.Sentry.DSN != "" {
		newC.Sentry.DSN = "***"
	}

	// Redact Monitor DSNs
	for i := range newC.Monitors {
		if newC.Monitors[i].Sentry.DSN != "" {
			newC.Monitors[i].Sentry.DSN = "***"
		}
		if newC.Monitors[i].Args != "" {
			parts := strings.Fields(newC.Monitors[i].Args)
			newC.Monitors[i].Args = sysstat.SanitizeCommand(parts)
		}
	}

	return &newC
}
