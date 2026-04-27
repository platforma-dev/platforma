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

// Mount mounts handler at both prefix (group root) and prefix+"/" (subtree).
// The handler receives requests with the path prefix stripped; an empty stripped
// path is normalized to "/" so that nested groups can register "GET /" etc.
func (hg *HandlerGroup) Mount(prefix string, handler http.Handler) {
	if prefix != "" && !strings.HasPrefix(prefix, "/") {
		panic("httpserver: mount prefix must be a path starting with /")
	}

	prefix = strings.TrimRight(prefix, "/")
	if prefix == "" {
		prefix = "/"
	}
	mounted := stripPrefix(prefix, handler)

	if prefix == "/" {
		hg.mux.Handle(prefix, mounted)
		return
	}

	hg.mux.Handle(prefix, mounted)
	hg.mux.Handle(prefix+"/", mounted)
}

// stripPrefix returns a handler that strips prefix from r.URL.Path, writing a 404
// if the request path does not start with prefix. If stripping leaves an empty
// path, it  is normalized to "/".
func stripPrefix(prefix string, handler http.Handler) http.Handler {
	if prefix == "/" {
		return handler
	}

	normalized := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"

			if r.URL.RawPath == "" {
				handler.ServeHTTP(w, r2)
				return
			}

			r2.URL.RawPath = "/"
			handler.ServeHTTP(w, r2)
			return
		}

		handler.ServeHTTP(w, r)
	})

	return http.StripPrefix(prefix, normalized)
}

// ServeHTTP implements the http.Handler interface, allowing HandlerGroup to
// be used as an HTTP handler itself.
func (hg *HandlerGroup) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wrappedMux := wrapHandlerInMiddleware(hg.mux, hg.middlewares)
	wrappedMux.ServeHTTP(w, r)
}
