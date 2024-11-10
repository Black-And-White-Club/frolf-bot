package graph

import (
	"context"
	"log"
	"strconv"

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

// Mutation resolvers

// ScheduleRound resolver
func (r *Resolver) ScheduleRound(ctx context.Context, input model.RoundInput) (*model.Round, error) {
	return r.ScoringServices.RoundService.ScheduleRound(ctx, input) // Implement this method in your RoundService
}

// JoinRound resolver
func (r *Resolver) JoinRound(ctx context.Context, roundID string, userID string) (*model.Round, error) {
	return r.ScoringServices.RoundService.JoinRound(ctx, roundID, userID) // Implement this method in your RoundService
}

// SubmitScore resolver
func (r *Resolver) SubmitScore(ctx context.Context, roundID string, userID string, score int) (*model.Round, error) {
	scores := map[string]string{userID: strconv.Itoa(score)} // Convert score to string
	return r.ScoringServices.RoundService.SubmitScore(ctx, roundID, scores)
}

// FinalizeRound resolver
func (r *Resolver) FinalizeRound(ctx context.Context, roundID string, editorID string) (*model.Round, error) {
	return r.ScoringServices.RoundService.FinalizeRound(ctx, roundID, editorID) // Implement this method in your RoundService
}
