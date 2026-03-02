package application

import (
	"context"
	"net/http"

	"github.com/platforma-dev/platforma/httpserver"
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

	if err := httpserver.WriteJSON(w, http.StatusOK, health); err != nil {
		log.ErrorContext(r.Context(), "failed to write health response", "error", err)
	}
}
