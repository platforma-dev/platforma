package httpserver

import (
	"net/http"
	"strings"
)

// HandlerGroup represents a group of HTTP handlers that share common middlewares.
type HandlerGroup struct {
	mux         *http.ServeMux
	middlewares []Middleware
}

// NewHandlerGroup creates a new HandlerGroup with an initialized http.ServeMux.
func NewHandlerGroup() *HandlerGroup {
	return &HandlerGroup{mux: http.NewServeMux()}
}

// Use adds a middleware to the HandlerGroup's middleware chain.
func (hg *HandlerGroup) Use(middlewares ...Middleware) {
	hg.middlewares = append(hg.middlewares, middlewares...)
}

// UseFunc adds a function as a middleware to the HandlerGroup's middleware chain.
func (hg *HandlerGroup) UseFunc(middlewareFuncs ...func(http.Handler) http.Handler) {
	for _, middlewareFunc := range middlewareFuncs {
		hg.middlewares = append(hg.middlewares, MiddlewareFunc(middlewareFunc))
	}
}

// Handle registers an http.Handler for the given pattern
func (hg *HandlerGroup) Handle(pattern string, handler http.Handler) {
	hg.mux.Handle(pattern, handler)
}

// HandleFunc registers an http.HandlerFunc for the given pattern
func (hg *HandlerGroup) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	hg.mux.Handle(pattern, http.HandlerFunc(handler))
}

// HandleGroup mounts handler at both pattern (group root) and pattern+"/" (subtree).
// The handler receives requests with the pattern prefix stripped; an empty stripped
// path is normalized to "/" so that nested groups can register "GET /" etc.
func (hg *HandlerGroup) HandleGroup(pattern string, handler http.Handler) {
	pattern = strings.TrimRight(pattern, "/")
	mounted := stripPrefix(pattern, handler)

	hg.mux.Handle(pattern, mounted)
	hg.mux.Handle(pattern+"/", mounted)
}

// stripPrefix returns a handler that strips prefix from r.URL.Path, writing a 404
// if the request path does not start with prefix. If stripping leaves an empty
// path, it is normalized to "/".
func stripPrefix(prefix string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, prefix)
		if p == r.URL.Path {
			http.NotFound(w, r)
			return
		}
		if p == "" {
			p = "/"
		}
		r2 := r.Clone(r.Context())
		r2.URL.Path = p
		if r.URL.RawPath != "" {
			r2.URL.RawPath = strings.TrimPrefix(r.URL.RawPath, prefix)
		}
		handler.ServeHTTP(w, r2)
	})
}

// ServeHTTP implements the http.Handler interface, allowing HandlerGroup to
// be used as an HTTP handler itself.
func (hg *HandlerGroup) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wrappedMux := wrapHandlerInMiddleware(hg.mux, hg.middlewares)
	wrappedMux.ServeHTTP(w, r)
}
