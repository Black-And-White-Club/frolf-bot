// graph/services/round_service.go
package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/araddon/dateparse"
	"github.com/google/uuid"
	"github.com/romero-jace/tcr-bot/graph/model"
)

type RoundService struct {
	client       *firestore.Client
	scoreService *ScoreService
}

// NewRoundService creates a new instance of RoundService
func NewRoundService(client *firestore.Client, scoreService *ScoreService) *RoundService {
	return &RoundService{
		client:       client,
		scoreService: scoreService,
	}
}

// generateID generates a unique ID for a round
func generateID() string {
	return uuid.New().String()
}

func normalizeDateTime(input string) (string, string, error) {
	// Parse the input date/time string
	parsedTime, err := dateparse.ParseAny(input)
	if err != nil {
		return "", "", err
	}

	// Normalize to military time and specific date format
	militaryTime := parsedTime.Format("15:04") // 24-hour format
	date := parsedTime.Format("01/02/2006")    // MM/DD/YYYY

	return militaryTime, date, nil
}

// / ScheduleRound creates a new round
func (s *RoundService) ScheduleRound(ctx context.Context, input model.RoundInput, creatorID string) (*model.Round, error) {
	militaryTime, date, err := normalizeDateTime(input.Time)
	if err != nil {
		return nil, err
	}

	// Declare and initialize the round variable
	round := model.Round{
		ID:           generateID(),
		Title:        input.Title,
		Location:     input.Location,
		EventType:    input.EventType,
		Date:         date,
		Time:         militaryTime,
		Participants: []*model.Participant{},
		Scores:       []*model.Score{},
		Finalized:    false,
		EditHistory:  []*model.EditLog{},
		CreatorID:    creatorID,
	}

	// Save the round to Firestore
	_, err = s.client.Collection("rounds").Doc(round.ID).Set(ctx, round)
	if err != nil {
		return nil, err
	}

	// Return the round
	return &round, nil
}

// SubmitScore allows users to submit scores for a round
func (s *RoundService) SubmitScore(ctx context.Context, roundID string, scores map[string]string) (*model.Round, error) {
	round, err := s.GetRoundByID(ctx, roundID)
	if err != nil {
		return nil, err
	}

	// Delegate score submission to ScoreService
	if err := s.scoreService.SubmitScore(ctx, round, scores); err != nil {
		return nil, err
	}

	// Update the round document in Firestore
	docRef := s.client.Collection("rounds").Doc(roundID)
	_, err = docRef.Set(ctx, round)
	if err != nil {
		return nil, err
	}

	return round, nil
}

// JoinRound allows a user to join an existing round
func (s *RoundService) JoinRound(ctx context.Context, roundID string, userID string, response model.Response) (*model.Round, error) {
	docRef := s.client.Collection("rounds").Doc(roundID)

	var round model.Round // Declare round here

	// Use a transaction to handle concurrent joins
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(docRef)
		if err != nil {
			return errors.New("round not found")
		}

		if err := doc.DataTo(&round); err != nil {
			return err
		}

		// Check if the user is already a participant
		for _, participant := range round.Participants {
			if participant.User.ID == userID {
				return errors.New("user already joined the round")
			}
		}

		// Handle the response type
		if response == model.DECLINE {
			// Log the decline without adding to participants
			round.EditHistory = append(round.EditHistory, &model.EditLog{
				EditorID:  userID,
				Timestamp: time.Now().Format(time.RFC3339),
				Changes:   fmt.Sprintf("User  %s declined to join the round", userID),
			})
		} else {
			// Add the user as a participant for ACCEPT and TENTATIVE
			round.Participants = append(round.Participants, &model.Participant{
				User:     &model.User{ID: userID},
				Response: response,
				Rank:     0,
			})
		}

		// Update the round document in Firestore
		return tx.Set(docRef, round)
	})

	if err != nil {
		return nil, err
	}

	return &round, nil // Now this will work
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

// GetRoundByID retrieves a round by its ID
func (s *RoundService) GetRoundByID(ctx context.Context, roundID string) (*model.Round, error) {
	docRef := s.client.Collection("rounds").Doc(roundID)
	doc, err := docRef.Get(ctx)
	if err != nil {
		return nil, errors.New("round not found")
	}

	var round model.Round
	if err := doc.DataTo(&round); err != nil {
		return nil, err
	}

	return &round, nil
}

// EditRound allows the creator to edit the round
func (s *RoundService) EditRound(ctx context.Context, roundID string, userID string, input model.RoundInput) (*model.Round, error) {
	docRef := s.client.Collection("rounds").Doc(roundID)

	var round model.Round // Declare round here

	// Use a transaction to ensure safe editing
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(docRef)
		if err != nil {
			return errors.New("round not found")
		}

		if err := doc.DataTo(&round); err != nil {
			return err
		}

		// Check if the user is the creator of the round
		if round.CreatorID != userID {
			return errors.New("only the creator can edit the round")
		}

		// Update round details
		round.Title = input.Title
		round.Location = input.Location
		round.EventType = input.EventType

		militaryTime, date, err := normalizeDateTime(input.Time)
		if err != nil {
			return err
		}
		round.Date = date
		round.Time = militaryTime

		// Log the edit action
		round.EditHistory = append(round.EditHistory, &model.EditLog{
			EditorID:  userID,
			Timestamp: time.Now().Format(time.RFC3339),
			Changes:   "Updated round details",
		})

		// Update the round document in Firestore
		return tx.Set(docRef, round)
	})

	if err != nil {
		return nil, err
	}

	return &round, nil // Now this will work
}

// DeleteRound allows a user to delete a round
func (s *RoundService) DeleteRound(ctx context.Context, roundID string, userID string) error {
	docRef := s.client.Collection("rounds").Doc(roundID)

	// Use a transaction to ensure safe deletion
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(docRef)
		if err != nil {
			return errors.New("round not found")
		}

		var round model.Round
		if err := doc.DataTo(&round); err != nil {
			return err
		}

		// Check if the user is the creator of the round
		if round.CreatorID != userID {
			return errors.New("only the creator can delete the round")
		}

		// Delete the round document from Firestore
		return tx.Delete(docRef)
	})

	if err != nil {
		return err
	}

	return nil
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
	editLog := &model.EditLog{
		EditorID:  editorID,
		Timestamp: time.Now().Format(time.RFC3339),
		Changes:   "Finalized round and updated rankings",
	}

	// Add the edit log to the round's edit history
	round.EditHistory = append(round.EditHistory, editLog)

	// Lock the round to prevent further changes
	round.Finalized = true

	// Update the round document in Firestore
	_, err = docRef.Set(ctx, round)
	if err != nil {
		return nil, err
	}
	return &round, nil
}
