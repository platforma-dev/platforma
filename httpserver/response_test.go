package httpserver_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/platforma-dev/platforma/httpserver"
)

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	t.Run("writes json response with status code", func(t *testing.T) {
		t.Parallel()

		data := map[string]string{"message": "hello"}

		w := httptest.NewRecorder()

		err := httpserver.WriteJSON(w, http.StatusCreated, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		resp := w.Result()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Fatalf("expected Content-Type 'application/json', got %s", contentType)
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]string
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("failed to unmarshal response body: %v", err)
		}

		if result["message"] != "hello" {
			t.Fatalf("expected message 'hello', got %s", result["message"])
		}
	})

	t.Run("writes struct as json", func(t *testing.T) {
		t.Parallel()

		type TestStruct struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}

		data := TestStruct{ID: 1, Name: "test"}

		w := httptest.NewRecorder()

		err := httpserver.WriteJSON(w, http.StatusOK, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)
		var result TestStruct
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("failed to unmarshal response body: %v", err)
		}

		if result.ID != 1 || result.Name != "test" {
			t.Fatalf("unexpected result: %+v", result)
		}
	})

	t.Run("returns error on unencodable data", func(t *testing.T) {
		t.Parallel()

		w := httptest.NewRecorder()

		unencodable := make(chan int)

		err := httpserver.WriteJSON(w, http.StatusOK, unencodable)
		if err == nil {
			t.Fatal("expected error for unencodable data")
		}

		if err.Error() == "" {
			t.Fatal("expected non-empty error message")
		}
	})
}
