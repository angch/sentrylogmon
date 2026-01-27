package config

import (
	"flag"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type SentryConfig struct {
	DSN         string `yaml:"dsn"`
	Environment string `yaml:"environment"`
	Release     string `yaml:"release"`
}

type MonitorConfig struct {
	Name    string `yaml:"name"`
	Type    string `yaml:"type"`   // file, journalctl, dmesg, command
	Path    string `yaml:"path"`   // for file
	Args    string `yaml:"args"`   // for journalctl or command
	Pattern string `yaml:"pattern"` // regex pattern for custom format
	Format  string `yaml:"format"` // dmesg, nginx, custom (default: custom if pattern set)
}

type Config struct {
	Sentry   SentryConfig    `yaml:"sentry"`
	Monitors []MonitorConfig `yaml:"monitors"`
	Verbose  bool            `yaml:"-"`
}

var (
	configFile  = flag.String("config", "", "Path to configuration file")
	dsn         = flag.String("dsn", os.Getenv("SENTRY_DSN"), "Sentry DSN")
	useDmesg    = flag.Bool("dmesg", false, "Monitor dmesg output")
	inputFile   = flag.String("file", "", "Monitor a log file")
	journalctl  = flag.String("journalctl", "", "Monitor journalctl output (pass args)")
	command     = flag.String("command", "", "Monitor custom command output")
	pattern     = flag.String("pattern", "Error", "Pattern to match (case sensitive)")
	environment = flag.String("environment", "production", "Sentry environment")
	release     = flag.String("release", "", "Sentry release version")
	verbose     = flag.Bool("verbose", false, "Verbose logging")
)

// ParseFlags parses the command line flags.
// It must be called before Load.
func ParseFlags() {
	if !flag.Parsed() {
		flag.Parse()
	}
}

func Load() (*Config, error) {
	// Ensure flags are parsed
	ParseFlags()

	cfg := &Config{
		Verbose: *verbose,
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
		// If verbose flag is set, it overrides config?
		// Actually config file doesn't have verbose field in YAML usually,
		// but let's stick to flag for verbose.
		cfg.Verbose = *verbose
		return cfg, nil
	}

	// Legacy/CLI mode
	cfg.Sentry = SentryConfig{
		DSN:         *dsn,
		Environment: *environment,
		Release:     *release,
	}

	monitor := MonitorConfig{
		Pattern: *pattern,
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
	}

	if monitor.Type != "" {
		cfg.Monitors = append(cfg.Monitors, monitor)
	}

	return cfg, nil
}
