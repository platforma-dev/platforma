package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/platforma-dev/platforma/httpserver"
	"github.com/platforma-dev/platforma/log"
)

type userDeleter interface {
	DeleteUser(ctx context.Context) error
}

type DeleteHandler struct {
	service userDeleter
}

func NewDeleteHandler(service userDeleter) *DeleteHandler {
	return &DeleteHandler{
		service: service,
	}
}

func (h *DeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := h.service.DeleteUser(ctx)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := httpserver.WriteJSON(w, http.StatusOK, "User deleted successfully"); err != nil {
		log.ErrorContext(ctx, "failed to write delete response", "error", err)
	}
}
