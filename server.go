package main

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"
	"github.com/romero-jace/leaderboard/graph"
	"github.com/romero-jace/leaderboard/graph/services"
)

func main() {
	ctx := context.Background()

	// Initialize Firestore
	if err := services.InitializeFirestore(ctx); err != nil {
		log.Fatalf("Failed to initialize Firestore: %v", err)
	}

	// Create Firestore client
	client, err := firestore.NewClient(ctx, "your-project-id")
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	// Initialize services
	userService := services.NewUserService()
	roundService := services.NewRoundService(client)

	// Create a new resolver with the services
	resolver := &graph.Resolver{
		UserService:  userService,
		RoundService: roundService,
	}

	// Set up your server and routes...
}
