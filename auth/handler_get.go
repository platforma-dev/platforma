package auth

import (
	"net/http"

	"github.com/platforma-dev/platforma/httpserver"
	"github.com/platforma-dev/platforma/log"
)

type GetHandler struct {
	service *Service
}

func NewGetHandler(service *Service) *GetHandler {
	return &GetHandler{
		service: service,
	}
}

func (h *GetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	cookie, err := r.Cookie(h.service.CookieName())
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user, err := h.service.GetFromSession(ctx, cookie.Value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	resp := struct {
		Username string `json:"username"`
	}{
		Username: user.Username,
	}

	if err := httpserver.WriteJSON(w, http.StatusOK, resp); err != nil {
		log.ErrorContext(ctx, "failed to write user response", "error", err)
	}
}
