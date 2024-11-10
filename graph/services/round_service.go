// graph/services/round_service.go
package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/romero-jace/tcr-bot/graph/model"

	"cloud.google.com/go/firestore"
)

type RoundService struct {
	client       *firestore.Client
	scoreService *ScoreService
}

// NewRoundService creates a new instance of RoundService
func NewRoundService(client *firestore.Client) *RoundService {
	return &RoundService{
		client:       client,
		scoreService: NewScoreService(client),
	}
}

// generateID generates a unique ID for a round
func generateID() string {
	return uuid.New().String()
}

// ScheduleRound creates a new round
func (s *RoundService) ScheduleRound(ctx context.Context, input model.RoundInput) (*model.Round, error) {
	round := model.Round{
		ID:           generateID(),
		Title:        input.Title,
		Location:     input.Location,
		EventType:    input.EventType,
		Date:         input.Date,
		Time:         input.Time,
		Participants: []*model.Participant{}, // Use pointers
		Scores:       []*model.Score{},       // Use pointers
		Finalized:    false,
		EditHistory:  []*model.EditLog{}, // Use pointers
	}

	// Save the round to Firestore
	_, err := s.client.Collection("rounds").Doc(round.ID).Set(ctx, round)
	if err != nil {
		return nil, err
	}

	return &round, nil
}

// JoinRound allows a user to join an existing round
func (s *RoundService) JoinRound(ctx context.Context, roundID string, userID string) (*model.Round, error) {
	docRef := s.client.Collection("rounds").Doc(roundID)
	doc, err := docRef.Get(ctx)
	if err != nil {
		return nil, errors.New("round not found")
	}

	var round model.Round
	if err := doc.DataTo(&round); err != nil {
		return nil, err
	}

	// Check if the user is already a participant
	for _, participant := range round.Participants {
		if participant.User.ID == userID {
			return nil, errors.New("user already joined the round")
		}
	}

	// Add the user as a participant
	round.Participants = append(round.Participants, &model.Participant{ // Use pointer
		User:     &model.User{ID: userID},  // Use pointer
		Response: model.Response("ACCEPT"), // Assign directly as a string
		Rank:     0,                        // Default rank
	})

	// Update the round document in Firestore
	_, err = docRef.Set(ctx, round)
	if err != nil {
		return nil, err
	}

	return &round, nil
}

// GetRounds retrieves a list of rounds with optional pagination
func (s *RoundService) GetRounds(ctx context.Context, limit *int, offset *int) ([]*model.Round, error) {
	query := s.client.Collection("rounds").OrderBy("date", firestore.Desc).Limit(*limit)
	if offset != nil {
		query = query.Offset(*offset)
	}

	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var rounds []*model.Round
	for _, doc := range docs {
		var round model.Round
		if err := doc.DataTo(&round); err != nil {
			return nil, err
		}
		rounds = append(rounds, &round)
	}

	return rounds, nil
}

// SubmitScore allows users to submit scores for themselves or multiple people
func (s *RoundService) SubmitScore(ctx context.Context, roundID string, scores map[string]string) (*model.Round, error) {
	docRef := s.client.Collection("rounds").Doc(roundID)
	doc, err := docRef.Get(ctx)
	if err != nil {
		return nil, errors.New("round not found")
	}

	var round model.Round
	if err := doc.DataTo(&round); err != nil {
		return nil, err
	}

	// Delegate score submission to ScoreService
	if err := s.scoreService.SubmitScore(ctx, &round, scores); err != nil {
		return nil, err
	}

	// Update the round document in Firestore
	_, err = docRef.Set(ctx, round)
	if err != nil {
		return nil, err
	}
	return &round, nil
}

// FinalizeRound finalizes a round
func (s *RoundService) FinalizeRound(ctx context.Context, roundID string, editorID string) (*model.Round, error) {
	docRef := s.client.Collection("rounds").Doc(roundID)
	doc, err := docRef.Get(ctx)
	if err != nil {
		return nil, errors.New("round not found")
	}

	var round model.Round
	if err := doc.DataTo(&round); err != nil {
		return nil, err
	}

	// Check if the round is already finalized
	if round.Finalized {
		return nil, errors.New("round has already been finalized")
	}

	// Log the edit action
	editLog := &model.EditLog{ // Use pointer
		EditorID:  editorID,
		Timestamp: time.Now().Format(time.RFC3339),
		Changes:   "Finalized round and updated rankings",
	}

	// Add the edit log to the round's edit history
	round.EditHistory = append(round.EditHistory, editLog) // Use pointer

	// Lock the round to prevent further changes
	round.Finalized = true

	// Update the round document in Firestore
	_, err = docRef.Set(ctx, round)
	if err != nil {
		return nil, err
	}
	return &round, nil
}
