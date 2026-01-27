package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"time"
)

func newUnixClient(socketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
		Timeout: 5 * time.Second,
	}
}

func ListInstances(socketDir string) ([]StatusResponse, error) {
	pattern := filepath.Join(socketDir, "sentrylogmon.*.sock")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var instances []StatusResponse

	for _, socketPath := range matches {
		client := newUnixClient(socketPath)
		// URL host is ignored by unix dialer, but scheme must be http
		resp, err := client.Get("http://unix/status")
		if err != nil {
			// Skip dead sockets or permission denied
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var status StatusResponse
			if err := json.NewDecoder(resp.Body).Decode(&status); err == nil {
				instances = append(instances, status)
			}
		}
	}

	return instances, nil
}

func RequestUpdate(socketPath string) error {
	client := newUnixClient(socketPath)
	resp, err := client.Post("http://unix/update", "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %s", resp.Status)
	}
	return nil
}
