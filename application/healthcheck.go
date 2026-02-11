package application

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/platforma-dev/platforma/log"
)

type healther interface {
	Health(context.Context) *Health
}

// HealthCheckHandler serves application health information as JSON.
type HealthCheckHandler struct {
	app healther
}

// NewHealthCheckHandler creates a HealthCheckHandler for the given application.
func NewHealthCheckHandler(app healther) *HealthCheckHandler {
	return &HealthCheckHandler{app: app}
}

func (h *HealthCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	health := h.app.Health(r.Context())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(health)
	if err != nil {
		log.ErrorContext(r.Context(), "failed to decode response to json", "error", err)
	}
}
