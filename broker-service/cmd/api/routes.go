package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (app *Config) routes() http.Handler {
	mux := chi.NewRouter()

	// specify who is allowed to connect
	mux.Use(cors.Handler(cors.Options{
		// dev mode, allow all connection
		AllowedOrigins:   []string{"https://*", "http://*"}, // allows all connections from all domains
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"}, // cross site request forgery token
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true, // deal with credential requests
		MaxAge:           300,
	}))

	// Make sure service is alive by checking it's heartbeat route
	mux.Use(middleware.Heartbeat("/ping"))

	// Write a POST request handler to our / endpoint
	mux.Post("/", app.Broker)

	// add routes
	return mux
}
