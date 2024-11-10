//go:build !test
// +build !test

package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/romero-jace/tcr-bot/graph"
	"github.com/romero-jace/tcr-bot/graph/services"
)

func main() {
	// Set the Firestore emulator host
	log.Println("Firestore Emulator Host:", os.Getenv("FIRESTORE_EMULATOR_HOST"))

	// Initialize Firestore client
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, "your-project-id") // Replace with your Firestore project ID
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()
	log.Println("Firestore client created successfully")

	// Initialize your services
	userService := services.NewUserService(client)
	scoreService := services.NewScoreService(client)
	roundService := services.NewRoundService(client, scoreService) // Pass scoreService here
	leaderboardService := services.NewLeaderboardService(client)

	// Create a new resolver
	resolver := graph.NewResolver(userService, scoreService, roundService, leaderboardService, client)

	// Set up GraphQL server
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))

	// Set up Chi router
	router := chi.NewRouter()

	// Use Chi's built-in middleware
	router.Use(middleware.RequestID) // Generates a unique request ID
	router.Use(middleware.Logger)    // Logs each request
	router.Use(middleware.Recoverer) // Recovers from panics

	// Apply your custom middleware
	router.Use(services.Middleware(client))

	// Set up routes
	router.Handle("/", playground.Handler("GraphQL playground", "/query"))
	router.Handle("/query", srv)

	// Start the server
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
