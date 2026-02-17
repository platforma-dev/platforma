package main

import (
	"context"
	"net/http"
	"time"

	"github.com/platforma-dev/platforma/application"
	"github.com/platforma-dev/platforma/httpserver"
	"github.com/platforma-dev/platforma/log"
)

func main() {
	ctx := context.Background()

	// Initialize new application
	app := application.New()

	// Create HTTP server
	api := httpserver.New("8080", 3*time.Second)

	// Add /ping endpoint to `api`
	api.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	// Add /long endpoint to HTTP server to test graceful shutdown
	api.HandleFunc("/long", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second)
		w.Write([]byte("pong"))
	})

	// Add middleware to HTTP server. It will add trace ID to logs and responce headers
	api.Use(log.NewTraceIDMiddleware(nil, ""))

	// Create handler group
	subApiGroup := httpserver.NewHandlerGroup()

	// Add /clock endpoint to handler group
	subApiGroup.HandleFunc("/clock", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(time.Now().String()))
	})

	// Add middleware to HTTP server. It will log all incoming requests to this handle group
	subApiGroup.UseFunc(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.InfoContext(r.Context(), "incoming request", "addr", r.RemoteAddr)
			h.ServeHTTP(w, r)
		})
	})

	// Add handle group to HTTP server with /subApi path
	api.HandleGroup("/subApi", subApiGroup)

	// Register HTTP server as application server
	app.RegisterService("api", api)

	// Run application
	if err := app.Run(ctx); err != nil {
		log.ErrorContext(ctx, "app finished with error", "error", err)
	}

	// Now you can access http://localhost:8080/ping, http://localhost:8080/long
	// and http://localhost:8080/subApi/clock URLs with GET method
}
