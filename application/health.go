package application

import (
	"encoding/json"
	"time"
)

// ServiceStatus represents the lifecycle state of a service.
type ServiceStatus string

const (
	// ServiceStatusNotStarted indicates service has not started yet.
	ServiceStatusNotStarted ServiceStatus = "NOT_STARTED"
	// ServiceStatusStarted indicates service is currently running.
	ServiceStatusStarted ServiceStatus = "STARTED"
	// ServiceStatusError indicates service finished with an error.
	ServiceStatusError ServiceStatus = "ERROR"
)

// ServiceHealth contains health information for a single service.
type ServiceHealth struct {
	Status    ServiceStatus `json:"status"`
	StartedAt *time.Time    `json:"startedAt"`
	StoppedAt *time.Time    `json:"stoppedAt,omitempty"`
	Error     string        `json:"error,omitempty"`
	Data      any           `json:"data,omitempty"`
}

// Health contains overall application health and service states.
type Health struct {
	StartedAt time.Time                 `json:"startedAt"`
	Services  map[string]*ServiceHealth `json:"services"`
}

// NewHealth creates an ApplicationHealth with initialized storage.
func NewHealth() *Health {
	return &Health{Services: make(map[string]*ServiceHealth)}
}

// StartService marks the given service as started and stores start time.
func (h *Health) StartService(serviceName string) {
	if service, ok := h.Services[serviceName]; ok {
		service.Status = ServiceStatusStarted

		st := time.Now()
		service.StartedAt = &st

		h.Services[serviceName] = service
	}
}

// FailService marks the given service as failed and stores the error.
func (h *Health) FailService(serviceName string, err error) {
	if service, ok := h.Services[serviceName]; ok {
		service.Status = ServiceStatusError

		st := time.Now()
		service.StoppedAt = &st

		service.Error = err.Error()

		h.Services[serviceName] = service
	}
}

// SetServiceData stores additional health payload for the given service.
func (h *Health) SetServiceData(serviceName string, data any) {
	if service, ok := h.Services[serviceName]; ok {
		service.Data = data
		h.Services[serviceName] = service
	}
}

func (h *Health) String() string {
	b, _ := json.Marshal(h)
	return string(b)
}

// StartApplication marks application start time.
func (h *Health) StartApplication() {
	h.StartedAt = time.Now()
}
