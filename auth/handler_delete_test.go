package auth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/platforma-dev/platforma/auth"
)

func TestDeleteHandler_Success(t *testing.T) {
	t.Parallel()

	mockService := &mockDeleteService{
		deleteUserErr: nil,
	}
	handler := auth.NewDeleteHandler(mockService)

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	ctx := context.WithValue(req.Context(), auth.UserContextKey, &auth.User{ID: "user-id"})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	expectedBody := `"User deleted successfully"`
	actualBody := w.Body.String()
	if actualBody != expectedBody+"\n" {
		t.Fatalf("expected body %q, got %q", expectedBody+"\n", actualBody)
	}
}

func TestDeleteHandler_WrongMethod(t *testing.T) {
	t.Parallel()

	mockService := &mockDeleteService{}
	handler := auth.NewDeleteHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", w.Code)
	}
}

func TestDeleteHandler_UserNotFound(t *testing.T) {
	t.Parallel()

	mockService := &mockDeleteService{
		deleteUserErr: auth.ErrUserNotFound,
	}
	handler := auth.NewDeleteHandler(mockService)

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestDeleteHandler_InternalError(t *testing.T) {
	t.Parallel()

	mockService := &mockDeleteService{
		deleteUserErr: errors.New("database error"),
	}
	handler := auth.NewDeleteHandler(mockService)

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	ctx := context.WithValue(req.Context(), auth.UserContextKey, &auth.User{ID: "user-id"})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}

func TestDeleteHandler_NoUserInContext(t *testing.T) {
	t.Parallel()

	mockService := &mockDeleteService{
		deleteUserErr: auth.ErrUserNotFound,
	}
	handler := auth.NewDeleteHandler(mockService)

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

type mockDeleteService struct {
	deleteUserErr error
}

func (m *mockDeleteService) DeleteUser(ctx context.Context) error {
	return m.deleteUserErr
}
