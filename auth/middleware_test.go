package auth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/platforma-dev/platforma/auth"
	platformalog "github.com/platforma-dev/platforma/log"
)

func TestAuthenticationMiddleware_ValidSession(t *testing.T) {
	t.Parallel()

	userSvc := &mockUserService{
		users: map[string]*auth.User{
			"valid-session-id": {ID: "user-id", Username: "testuser"},
		},
		cookieName: "session",
	}
	middleware := auth.NewAuthenticationMiddleware(userSvc)

	handler := middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "valid-session-id"})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestAuthenticationMiddleware_ValidSessionAddsUserInfoToWideEvent(t *testing.T) {
	t.Parallel()

	userSvc := &mockUserService{
		users: map[string]*auth.User{
			"valid-session-id": {ID: "user-id", Username: "testuser"},
		},
		cookieName: "session",
	}
	middleware := auth.NewAuthenticationMiddleware(userSvc)

	handler := middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "valid-session-id"})

	event := platformalog.NewEvent("http.request")
	req = req.WithContext(context.WithValue(req.Context(), platformalog.WideEventKey, event))

	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	userID, ok := event.Attr("user.id")
	if !ok {
		t.Fatal("expected user.id attribute in event")
	}
	if userID != "user-id" {
		t.Fatalf("expected user.id to be user-id, got %v", userID)
	}

	username, ok := event.Attr("user.username")
	if !ok {
		t.Fatal("expected user.username attribute in event")
	}
	if username != "testuser" {
		t.Fatalf("expected user.username to be testuser, got %v", username)
	}
}

func TestAuthenticationMiddleware_NoSessionCookie(t *testing.T) {
	t.Parallel()

	userSvc := &mockUserService{
		cookieName: "session",
	}
	middleware := auth.NewAuthenticationMiddleware(userSvc)

	handler := middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called when authentication fails")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestAuthenticationMiddleware_InvalidSession(t *testing.T) {
	t.Parallel()

	userSvc := &mockUserService{
		users:      map[string]*auth.User{},
		cookieName: "session",
	}
	middleware := auth.NewAuthenticationMiddleware(userSvc)

	handler := middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called when authentication fails")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "invalid-session-id"})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestAuthenticationMiddleware_UserServiceError(t *testing.T) {
	t.Parallel()

	userSvc := &mockUserService{
		error:      errors.New("database error"),
		cookieName: "session",
	}
	middleware := auth.NewAuthenticationMiddleware(userSvc)

	handler := middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called when authentication fails")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "session-id"})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}

func TestAuthenticationMiddleware_UserNotFound(t *testing.T) {
	t.Parallel()

	userSvc := &mockUserService{
		users:      map[string]*auth.User{},
		cookieName: "session",
	}
	middleware := auth.NewAuthenticationMiddleware(userSvc)

	handler := middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called when authentication fails")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "session-id"})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

type mockUserService struct {
	users      map[string]*auth.User
	error      error
	cookieName string
}

func (m *mockUserService) GetFromSession(ctx context.Context, sessionId string) (*auth.User, error) {
	if m.error != nil {
		return nil, m.error
	}

	if user, ok := m.users[sessionId]; ok {
		return user, nil
	}
	return nil, auth.ErrUserNotFound
}

func (m *mockUserService) CookieName() string {
	return m.cookieName
}
