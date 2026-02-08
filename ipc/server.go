package ipc

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/angch/sentrylogmon/config"
)

func StartServer(socketPath string, cfg *config.Config, restartFunc func()) error {
	// Ensure socket file is removed before listening, in case of crash/restart
	os.Remove(socketPath)

	listener, err := listenSecure("unix", socketPath)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()

	startTime := time.Now()

	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		status := StatusResponse{
			PID:         os.Getpid(),
			StartTime:   startTime,
			Version:     cfg.Sentry.Release, // Assuming Release is version
			MemoryAlloc: m.Alloc,
			Config:      cfg.Redacted(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	})

	mux.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Acknowledge request before restarting
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Restarting..."))

		// execute restart in a separate goroutine to allow response to return
		go func() {
			time.Sleep(100 * time.Millisecond) // Give time for response to flush
			if restartFunc != nil {
				restartFunc()
			}
		}()
	})

	server := &http.Server{
		Handler: mux,
	}

	if cfg.Verbose {
		log.Printf("IPC Server listening on %s", socketPath)
	}

	return server.Serve(listener)
}
