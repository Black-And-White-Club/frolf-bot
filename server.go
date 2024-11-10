package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
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
	roundService := services.NewRoundService(client)
	leaderboardService := services.NewLeaderboardService(client)

	// Create a new resolver
	resolver := graph.NewResolver(userService, scoreService, roundService, leaderboardService, client)

	// Set up GraphQL server
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))

	// Set up HTTP server
	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
