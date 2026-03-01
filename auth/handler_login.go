package auth

import (
	"encoding/json"
	"errors"
	"net/http"
)

type LoginHandler struct {
	service *Service
}

func NewLoginHandler(service *Service) *LoginHandler {
	return &LoginHandler{
		service: service,
	}
}

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Login    string `json:"login"`
		Password string `json:"password"` //nolint:gosec // Password in request
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	sessionId, err := h.service.CreateSessionFromUsernameAndPassword(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, ErrWrongUserOrPassword) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if sessionId == "" {
		http.Error(w, "invalid login or password", http.StatusUnauthorized)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     h.service.CookieName(),
		Value:    sessionId,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.WriteHeader(http.StatusOK)
}
