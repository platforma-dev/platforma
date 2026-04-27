package httpserver_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/platforma-dev/platforma/httpserver"
)

func TestHTTPServer(t *testing.T) {
	t.Parallel()

	t.Run("single http.HandlerFunc endpoint", func(t *testing.T) {
		t.Parallel()

		server := httpserver.New("", 0)

		server.HandleFunc("/ping", func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte("pong"))
		})

		r := httptest.NewRequest(http.MethodGet, "/ping", nil)
		w := httptest.NewRecorder()

		server.ServeHTTP(w, r)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)

		if string(body) != "pong" {
			t.Fatalf("expected body to be 'pong', got %s", string(body))
		}
	})

	t.Run("single http.Handler endpoint", func(t *testing.T) {
		t.Parallel()

		pingHandler := &handler{
			serveHTTP: func(w http.ResponseWriter, _ *http.Request) {
				w.Write([]byte("pong"))
			},
		}

		server := httpserver.New("", 0)

		server.Handle("/ping", pingHandler)

		r := httptest.NewRequest(http.MethodGet, "/ping", nil)
		w := httptest.NewRecorder()

		server.ServeHTTP(w, r)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)

		if string(body) != "pong" {
			t.Fatalf("expected body to be 'pong', got %s", string(body))
		}
	})

	t.Run("mount group", func(t *testing.T) {
		t.Parallel()

		hg := httpserver.NewHandlerGroup()
		hg.Handle("/test", &handler{})

		server := httpserver.New("", 0)
		server.Mount("/hg", hg)

		r := httptest.NewRequest(http.MethodGet, "/hg/test", nil)
		w := httptest.NewRecorder()

		server.ServeHTTP(w, r)

		resp := w.Result()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status code to be 200, got %d", resp.StatusCode)
		}
	})

	t.Run("mount group exact root", func(t *testing.T) {
		t.Parallel()

		hg := httpserver.NewHandlerGroup()
		hg.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte("root"))
		})

		server := httpserver.New("", 0)
		server.Mount("/hg", hg)

		r := httptest.NewRequest(http.MethodGet, "/hg", nil)
		w := httptest.NewRecorder()

		server.ServeHTTP(w, r)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status code to be 200, got %d", resp.StatusCode)
		}

		if string(body) != "root" {
			t.Fatalf("expected body to be 'root', got %s", string(body))
		}
	})

	t.Run("mount group subtree", func(t *testing.T) {
		t.Parallel()

		hg := httpserver.NewHandlerGroup()
		hg.HandleFunc("GET /verify", func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte("verified"))
		})

		server := httpserver.New("", 0)
		server.Mount("/domains", hg)

		r := httptest.NewRequest(http.MethodGet, "/domains/verify", nil)
		w := httptest.NewRecorder()

		server.ServeHTTP(w, r)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status code to be 200, got %d", resp.StatusCode)
		}

		if string(body) != "verified" {
			t.Fatalf("expected body to be 'verified', got %s", string(body))
		}
	})

	t.Run("mount group trailing slash pattern", func(t *testing.T) {
		t.Parallel()

		hg := httpserver.NewHandlerGroup()
		hg.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte("root"))
		})

		server := httpserver.New("", 0)
		server.Mount("/hg/", hg)

		r := httptest.NewRequest(http.MethodGet, "/hg", nil)
		w := httptest.NewRecorder()

		server.ServeHTTP(w, r)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status code to be 200, got %d", resp.StatusCode)
		}

		if string(body) != "root" {
			t.Fatalf("expected body to be 'root', got %s", string(body))
		}
	})

	t.Run("mount group root prefix", func(t *testing.T) {
		t.Parallel()

		hg := httpserver.NewHandlerGroup()
		hg.HandleFunc("GET /verify", func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte("verified"))
		})

		server := httpserver.New("", 0)
		server.Mount("/", hg)

		r := httptest.NewRequest(http.MethodGet, "/verify", nil)
		w := httptest.NewRecorder()

		server.ServeHTTP(w, r)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status code to be 200, got %d", resp.StatusCode)
		}

		if string(body) != "verified" {
			t.Fatalf("expected body to be 'verified', got %s", string(body))
		}
	})

	t.Run("mount group empty root prefix", func(t *testing.T) {
		t.Parallel()

		hg := httpserver.NewHandlerGroup()
		hg.HandleFunc("GET /verify", func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte("verified"))
		})

		server := httpserver.New("", 0)
		server.Mount("", hg)

		r := httptest.NewRequest(http.MethodGet, "/verify", nil)
		w := httptest.NewRecorder()

		server.ServeHTTP(w, r)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status code to be 200, got %d", resp.StatusCode)
		}

		if string(body) != "verified" {
			t.Fatalf("expected body to be 'verified', got %s", string(body))
		}
	})

	t.Run("mount rejects method pattern", func(t *testing.T) {
		t.Parallel()

		hg := httpserver.NewHandlerGroup()
		server := httpserver.New("", 0)

		defer func() {
			if recover() == nil {
				t.Fatal("expected Mount with method pattern to panic")
			}
		}()

		server.Mount("GET /hg", hg)
	})

	t.Run("mount rejects escaped prefix mismatch", func(t *testing.T) {
		t.Parallel()

		hg := httpserver.NewHandlerGroup()
		hg.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte("root"))
		})

		server := httpserver.New("", 0)
		server.Mount("/hg", hg)

		r := httptest.NewRequest(http.MethodGet, "/hg", nil)
		r.URL.RawPath = "/%68g"
		w := httptest.NewRecorder()

		server.ServeHTTP(w, r)

		resp := w.Result()

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected status code to be 404, got %d", resp.StatusCode)
		}
	})

	t.Run("mount group not found", func(t *testing.T) {
		t.Parallel()

		hg := httpserver.NewHandlerGroup()
		hg.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte("root"))
		})

		server := httpserver.New("", 0)
		server.Mount("/hg", hg)

		r := httptest.NewRequest(http.MethodGet, "/other", nil)
		w := httptest.NewRecorder()

		server.ServeHTTP(w, r)

		resp := w.Result()

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected status code to be 404, got %d", resp.StatusCode)
		}
	})

	t.Run("healthcheck", func(t *testing.T) {
		t.Parallel()

		server := httpserver.New("8080", 0)
		hcData, ok := server.Healthcheck(context.TODO()).(map[string]any)
		if !ok {
			t.Fatal("failed type assert health data")
		}

		port := hcData["port"]
		if port != "8080" {
			t.Fatalf("expected port to be 8080, got %s", port)
		}
	})

	t.Run("use", func(t *testing.T) {
		t.Parallel()

		server := httpserver.New("", 0)

		customMiddleware := &testMiddleware{
			wrapFunc: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Test-Middleware", "applied")
					next.ServeHTTP(w, r)
				})
			},
		}
		server.Use(customMiddleware)
		server.Handle("/test", &handler{})

		r := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		server.ServeHTTP(w, r)
		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status code to be 200, got %d", resp.StatusCode)
		}

		middlewareHeader := resp.Header.Get("X-Test-Middleware")
		if middlewareHeader != "applied" {
			t.Fatalf("expected X-Test-Middleware header to be 'applied', got %s", middlewareHeader)
		}
	})

	t.Run("use func", func(t *testing.T) {
		t.Parallel()

		server := httpserver.New("", 0)

		customMiddlewareFunc := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Test-Func-Middleware", "applied")
				next.ServeHTTP(w, r)
			})
		}

		server.UseFunc(customMiddlewareFunc)
		server.Handle("/test", &handler{})

		r := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		server.ServeHTTP(w, r)
		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status code to be 200, got %d", resp.StatusCode)
		}

		middlewareHeader := resp.Header.Get("X-Test-Func-Middleware")
		if middlewareHeader != "applied" {
			t.Fatalf("expected X-Test-Middleware header to be 'applied', got %s", middlewareHeader)
		}
	})

	t.Run("multiple middlewares", func(t *testing.T) {
		t.Parallel()

		server := httpserver.New("", 0)

		middlewareCallLog := []string{}

		firstMiddleware := &testMiddleware{
			wrapFunc: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					middlewareCallLog = append(middlewareCallLog, "first")
					w.Header().Set("X-First-Middleware", "applied")
					next.ServeHTTP(w, r)
				})
			},
		}

		secondMiddlewareFunc := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				middlewareCallLog = append(middlewareCallLog, "second")
				w.Header().Set("X-Second-Middleware", "applied")
				next.ServeHTTP(w, r)
			})
		}

		server.Use(firstMiddleware)
		server.UseFunc(secondMiddlewareFunc)

		r := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		server.ServeHTTP(w, r)
		resp := w.Result()

		firstHeader := resp.Header.Get("X-First-Middleware")
		if firstHeader != "applied" {
			t.Fatalf("expected X-First-Middleware header to be 'applied', got %s", firstHeader)
		}

		secondHeader := resp.Header.Get("X-Second-Middleware")
		if secondHeader != "applied" {
			t.Fatalf("expected X-Second-Middleware header to be 'applied', got %s", secondHeader)
		}

		if middlewareCallLog[0] != "first" {
			t.Fatalf("expected first middleware to be called first, got %s", middlewareCallLog[0])
		}

		if middlewareCallLog[1] != "second" {
			t.Fatalf("expected second middleware to be called second, got %s", middlewareCallLog[1])
		}
	})
}

type handler struct {
	serveHTTP func(http.ResponseWriter, *http.Request)
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.serveHTTP != nil {
		h.serveHTTP(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type testMiddleware struct {
	wrapFunc func(http.Handler) http.Handler
}

func (m *testMiddleware) Wrap(next http.Handler) http.Handler {
	if m.wrapFunc != nil {
		return m.wrapFunc(next)
	}
	return next
}
