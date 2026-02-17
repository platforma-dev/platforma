package main

import (
	"context"
	"net/http"
	"time"

	"github.com/platforma-dev/platforma/application"
	"github.com/platforma-dev/platforma/auth"
	"github.com/platforma-dev/platforma/database"
	"github.com/platforma-dev/platforma/httpserver"
	"github.com/platforma-dev/platforma/log"
	"github.com/platforma-dev/platforma/session"
)

func main() {
	ctx := context.Background()

	db, err := database.New("postgres://user:pass@localhost:5432/mydb?sslmode=disable")
	if err != nil {
		log.ErrorContext(ctx, "failed to connect to database", "error", err)
		return
	}

	sessionDomain := session.New(db.Connection())
	authDomain := auth.New(db.Connection(), sessionDomain.Service, "session_id", nil, nil, nil)

	app := application.New()
	app.RegisterDatabase("main", db)
	app.RegisterDomain("session", "main", sessionDomain)
	app.RegisterDomain("auth", "main", authDomain)

	api := httpserver.New("8080", 3*time.Second)
	api.Use(log.NewTraceIDMiddleware(nil, ""))
	api.Use(httpserver.NewRecoverMiddleware())

	api.HandleGroup("/auth", authDomain.HandleGroup)

	protected := httpserver.NewHandlerGroup()
	protected.Use(authDomain.Middleware)
	protected.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		w.Write([]byte("Welcome, " + user.Username))
	})
	api.HandleGroup("/api", protected)

	api.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	app.RegisterService("api", api)

	if err := app.Run(ctx); err != nil {
		log.ErrorContext(ctx, "app finished with error", "error", err)
	}
}
