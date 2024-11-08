// graph/services/round_service.go
package services

import (
	"context"
	"errors"
	"your-module/graph/model"

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
	if round.IsFinalized {
		return nil, errors.New("round has already been finalized")
	}

	// Delegate finalization to ScoreService
	if err := s.scoreService.FinalizeRound(&round, editorID); err != nil {
		return nil, err
	}

	// Update the round document in Firestore
	_, err = docRef.Set(ctx, round)
	if err != nil {
		return nil, err
	}
	return &round, nil
}
