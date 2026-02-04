package ipc

import (
	"time"

	"github.com/angch/sentrylogmon/config"
)

type StatusResponse struct {
	PID         int            `json:"pid"`
	StartTime   time.Time      `json:"start_time"`
	Version     string         `json:"version"` // from config
	MemoryAlloc uint64         `json:"memory_alloc,omitempty"`
	Config      *config.Config `json:"config"`
}

type UpdateRequest struct {
	Action string `json:"action"` // "restart"
}
