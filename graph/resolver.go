package graph

import (
	"context"
	"errors"
	"log"

	"cloud.google.com/go/firestore"
	"github.com/romero-jace/tcr-bot/graph/model"    // Corrected import path
	"github.com/romero-jace/tcr-bot/graph/services" // Adjust the import path as necessary
)

// Resolver serves as dependency injection for your app.
type Resolver struct {
	UserServices       UserServices
	ScoringServices    ScoringServices
	LeaderboardService *services.LeaderboardService // Add this line
	DB                 *firestore.Client
}

// UserServices groups all user-related services
type UserServices struct {
	UserService *services.UserService
}

// GetUser  is a wrapper method for UserService's GetUser
func (us *UserServices) GetUser(ctx context.Context, discordID string) (*model.User, error) {
	return us.UserService.GetUser(ctx, discordID)
}

// CreateUser  is a wrapper method for UserService's CreateUser
func (r *Resolver) CreateUser(ctx context.Context, input model.UserInput) (*model.User, error) {
	log.Printf("CreateUser  called with input: %+v", input)
	// Call the CreateUser  method on the UserService field
	return r.UserServices.UserService.CreateUser(ctx, input)
}

// ScoringServices groups all scoring-related services
type ScoringServices struct {
	ScoreService *services.ScoreService
	RoundService *services.RoundService // Assuming you have a RoundService
}

// NewResolver creates a new Resolver with dependencies injected.
func NewResolver(userService *services.UserService, scoreService *services.ScoreService, roundService *services.RoundService, leaderboardService *services.LeaderboardService, db *firestore.Client) *Resolver {
	return &Resolver{
		UserServices: UserServices{
			UserService: userService,
		},
		ScoringServices: ScoringServices{
			ScoreService: scoreService,
			RoundService: roundService,
		},
		LeaderboardService: leaderboardService, // Add this line
		DB:                 db,                 // Pass the Firestore client
	}
}

// In your resolver.go or a utility file
func getUserIDFromContext(ctx context.Context) (string, error) {
	// Assuming you have a way to get the user ID from the context
	userID, ok := ctx.Value("userID").(string)
	if !ok {
		return "", errors.New("user ID not found in context")
	}
	return userID, nil
}

// Query resolvers

// GetUser  resolver
func (r *Resolver) GetUser(ctx context.Context, discordID string) (*model.User, error) {
	return r.UserServices.GetUser(ctx, discordID)
}

// GetLeaderboard resolver
func (r *Resolver) GetLeaderboard(ctx context.Context) (*model.Leaderboard, error) {
	return r.LeaderboardService.GetLeaderboard(ctx) // Implement this method in your LeaderboardService
}

// GetRounds resolver
func (r *Resolver) GetRounds(ctx context.Context, limit *int, offset *int) ([]*model.Round, error) {
	return r.ScoringServices.RoundService.GetRounds(ctx, limit, offset) // Implement this method in your RoundService
}

// GetUser Score resolver
func (r *Resolver) GetUserScore(ctx context.Context, userID string) (int, error) {
	return r.ScoringServices.ScoreService.GetUserScore(ctx, userID) // Call the method from ScoreService
}
