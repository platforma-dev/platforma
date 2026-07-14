package auth

import (
	"net/http"
	"time"
)

type LogoutHandler struct {
	service *Service
}

func NewLogoutHandler(service *Service) *LogoutHandler {
	return &LogoutHandler{
		service: service,
	}
}

func (h *LogoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract session cookie
	cookie, err := r.Cookie(h.service.CookieName())
	if err != nil {
		// No session cookie found, still return success
		w.WriteHeader(http.StatusOK)
		return
	}

	// Delete session from database
	if err := h.service.DeleteSession(r.Context(), cookie.Value); err != nil {
		http.Error(w, "failed to logout", http.StatusInternalServerError)
		return
	}

	// Clear session cookie by setting it to expire immediately
	http.SetCookie(w, &http.Cookie{
		Name:     h.service.CookieName(),
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Return success response
	w.WriteHeader(http.StatusOK)
}
