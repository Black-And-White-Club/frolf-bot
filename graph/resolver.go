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
	LeaderboardService *services.LeaderboardService

	// Function fields for mocking
	CreateUserFunc     func(ctx context.Context, input model.UserInput) (*model.User, error)
	GetLeaderboardFunc func(ctx context.Context) (*model.Leaderboard, error)
	GetRoundsFunc      func(ctx context.Context, limit *int, offset *int) ([]*model.Round, error)
	GetUserScoreFunc   func(ctx context.Context, userID string) (int, error)
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
	if r.CreateUserFunc != nil {
		return r.CreateUserFunc(ctx, input) // Call the mock function if set
	}

	log.Printf("CreateUser  called with input: %+v", input)
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
		LeaderboardService: leaderboardService,
		DB:                 db,
	}
}

// Define a custom type for the context key
type contextKey string

// Export the key by capitalizing the first letter
const UserIDKey contextKey = "userID"

// GetUser IDFromContext retrieves the user ID from the context
func GetUserIDFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(UserIDKey).(string)
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
	if r.GetLeaderboardFunc != nil {
		return r.GetLeaderboardFunc(ctx) // Call the mock function if set
	}

	return r.LeaderboardService.GetLeaderboard(ctx) // Call the actual method
}

// GetRounds resolver
func (r *Resolver) GetRounds(ctx context.Context, limit *int, offset *int) ([]*model.Round, error) {
	if r.GetRoundsFunc != nil {
		return r.GetRoundsFunc(ctx, limit, offset) // Call the mock function if set
	}

	return r.ScoringServices.RoundService.GetRounds(ctx, limit, offset) // Call the actual method
}

// GetUser Score resolver
func (r *Resolver) GetUserScore(ctx context.Context, userID string) (int, error) {
	if r.GetUserScoreFunc != nil {
		return r.GetUserScoreFunc(ctx, userID) // Call the mock function if set
	}

	return r.ScoringServices.ScoreService.GetUserScore(ctx, userID) // Call the actual method
}
