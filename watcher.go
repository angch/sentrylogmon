package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/angch/sentrylogmon/config"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

func watchConfig(ctx context.Context, configPath string, onReload func()) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Failed to create file watcher: %v", err)
		return
	}
	defer watcher.Close()

	if err := watcher.Add(configPath); err != nil {
		log.Printf("Failed to watch config file %s: %v", configPath, err)
		return
	}

	log.Printf("Watching config file %s for changes...", configPath)

	var debounceTimer *time.Timer
	const debounceDuration = 500 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) || event.Has(fsnotify.Chmod) {
				// Rename can happen if some editors save by atomic rename.
				// However, if it's renamed, the watcher might lose track if it's not a directory watcher.
				// But let's assume standard write or atomic replace (which might require re-adding).
				// If atomic rename happens, the inode changes. fsnotify might handle it or not depending on OS.
				// For robustness, if Rename/Remove happens, we might need to re-add the file.

				if event.Has(fsnotify.Rename) || event.Has(fsnotify.Remove) {
					// Wait a bit for the new file to appear
					time.Sleep(100 * time.Millisecond)
					if err := watcher.Add(configPath); err != nil {
						// If we can't re-add, maybe it's gone for good or permission issue.
						// Log and continue (maybe retry later? but we keep loop)
						log.Printf("Config file %s renamed/removed and could not be re-watched: %v", configPath, err)
						continue
					}
				}

				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(debounceDuration, func() {
					// Validate config
					data, err := os.ReadFile(configPath)
					if err != nil {
						log.Printf("Failed to read config file during reload check: %v", err)
						return
					}

					var cfg config.Config
					if err := yaml.Unmarshal(data, &cfg); err != nil {
						log.Printf("Config file changed but is invalid (YAML error), ignoring reload: %v", err)
						return
					}

					if err := cfg.Validate(); err != nil {
						log.Printf("Config file changed but is invalid (Validation error), ignoring reload: %v", err)
						return
					}

					log.Println("Config file changed and valid, reloading...")
					onReload()
				})
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}
