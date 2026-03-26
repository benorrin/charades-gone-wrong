package main

import (
	"fmt"
	"net/http"

	"game/api"
	"game/db"
	"game/logger"
)

func main() {
	// Initialize database
	database, err := db.New("game.db")
	if err != nil {
		logger.Panic("Main", "Failed to initialize database: %v", err)
	}
	defer database.Close()

	logger.Info("Main", "Database initialized successfully")

	// Create API server
	server := api.NewServer(database)

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok"}`)
	})

	// Register all API routes
	server.RegisterRoutes(mux)

	// Start server
	port := ":8080"
	logger.Info("Main", "Starting server on %s", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		logger.Panic("Main", "Server error: %v", err)
	}
}
