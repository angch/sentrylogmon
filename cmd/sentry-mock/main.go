package main

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
)

type EventStore struct {
	mu     sync.Mutex
	Events [][]byte `json:"events"`
}

var store = &EventStore{
	Events: make([][]byte, 0),
}

func (s *EventStore) Add(data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Events = append(s.Events, data)
	log.Printf("Received event, total: %d", len(s.Events))
}

func (s *EventStore) GetAll() [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Return a copy
	dst := make([][]byte, len(s.Events))
	copy(dst, s.Events)
	return dst
}

func (s *EventStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Events = make([][]byte, 0)
	log.Println("Cleared all events")
}

func handleEnvelope(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reader io.ReadCloser
	var err error

	// Handle GZIP
	if r.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "Failed to create gzip reader", http.StatusBadRequest)
			return
		}
		defer reader.Close()
	} else {
		reader = r.Body
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}

	// Simple validation
	if len(body) == 0 {
		http.Error(w, "Empty body", http.StatusBadRequest)
		return
	}

	store.Add(body)

	// Sentry expects a JSON response with id, usually.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"id":"c9938dbd8dd54b778e741a8d0869aacd"}`))
}

func handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		events := store.GetAll()

		// Convert bytes to strings for JSON output
		stringEvents := make([]string, len(events))
		for i, e := range events {
			stringEvents[i] = string(e)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stringEvents)
	} else if r.Method == http.MethodDelete {
		store.Clear()
		w.WriteHeader(http.StatusOK)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	http.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/envelope/") || strings.HasSuffix(r.URL.Path, "/store/") {
			handleEnvelope(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	http.HandleFunc("/events", handleEvents)

	log.Println("Sentry Mock Server listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
