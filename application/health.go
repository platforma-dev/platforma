package application

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
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

	mu sync.RWMutex
}

// NewHealth creates an ApplicationHealth with initialized storage.
func NewHealth() *Health {
	return &Health{Services: make(map[string]*ServiceHealth)}
}

// RegisterService adds a service with its initial health status.
func (h *Health) RegisterService(serviceName string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.Services[serviceName] = &ServiceHealth{Status: ServiceStatusNotStarted}
}

// StartService marks the given service as started and stores start time.
func (h *Health) StartService(serviceName string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if service, ok := h.Services[serviceName]; ok {
		service.Status = ServiceStatusStarted

		st := time.Now()
		service.StartedAt = &st
	}
}

// FailService marks the given service as failed and stores the error.
func (h *Health) FailService(serviceName string, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if service, ok := h.Services[serviceName]; ok {
		service.Status = ServiceStatusError

		st := time.Now()
		service.StoppedAt = &st

		service.Error = err.Error()
	}
}

// SetServiceData stores an owned, JSON-serialized copy of a service health payload.
// The caller must not mutate data until SetServiceData returns, but retains ownership
// and may safely mutate it afterward.
func (h *Health) SetServiceData(serviceName string, data any) error {
	var dataSnapshot json.RawMessage
	var serializeErr error
	if data != nil {
		dataSnapshot, serializeErr = serializeServiceData(data)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	service, ok := h.Services[serviceName]
	if !ok {
		return nil
	}
	if data == nil {
		service.Data = nil

		return nil
	}
	if serializeErr != nil {
		return fmt.Errorf("serialize health data for service %q: %w", serviceName, serializeErr)
	}

	service.Data = dataSnapshot

	return nil
}

// Snapshot returns a detached copy of the current health state.
func (h *Health) Snapshot() *Health {
	h.mu.RLock()
	defer h.mu.RUnlock()

	services := make(map[string]*ServiceHealth, len(h.Services))
	for serviceName, service := range h.Services {
		services[serviceName] = snapshotServiceHealth(service)
	}

	return &Health{
		StartedAt: h.StartedAt,
		Services:  services,
	}
}

func snapshotServiceHealth(service *ServiceHealth) *ServiceHealth {
	if service == nil {
		return nil
	}

	snapshot := *service
	snapshot.StartedAt = snapshotTime(service.StartedAt)
	snapshot.StoppedAt = snapshotTime(service.StoppedAt)
	snapshot.Data = snapshotServiceData(service.Data)

	return &snapshot
}

func serializeServiceData(data any) (json.RawMessage, error) {
	serialized, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal health data: %w", err)
	}

	return json.RawMessage(serialized), nil
}

func snapshotServiceData(data any) any {
	if data == nil {
		return nil
	}

	serialized, ok := data.(json.RawMessage)
	if !ok {
		return nil
	}

	return json.RawMessage(bytes.Clone(serialized))
}

func snapshotTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	snapshot := *value

	return &snapshot
}

func (h *Health) String() string {
	b, _ := json.Marshal(h.Snapshot())
	return string(b)
}

// StartApplication marks application start time.
func (h *Health) StartApplication() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.StartedAt = time.Now()
}
