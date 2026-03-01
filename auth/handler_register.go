package auth

import (
	"encoding/json"
	"errors"
	"net/http"
)

type RegisterHandler struct {
	service *Service
}

func NewRegisterHandler(service *Service) *RegisterHandler {
	return &RegisterHandler{
		service: service,
	}
}

func (h *RegisterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	// Register user
	if err := h.service.CreateWithLoginAndPassword(r.Context(), req.Login, req.Password); err != nil {
		if errors.Is(err, ErrInvalidUsername) {
			http.Error(w, "invalid username", http.StatusBadRequest)
			return
		}

		if errors.Is(err, ErrInvalidPassword) {
			http.Error(w, "invalid password", http.StatusBadRequest)
			return
		}

		http.Error(w, "failed to register user", http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusCreated)
}
