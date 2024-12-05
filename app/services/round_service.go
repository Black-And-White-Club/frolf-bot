package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/app/models"
	"github.com/Black-And-White-Club/tcr-bot/internal/db"
	"github.com/Black-And-White-Club/tcr-bot/internal/nats"
)

// RoundService handles round-related logic and database interactions.
type RoundService struct {
	db                 db.RoundDB
	natsConnectionPool *nats.NatsConnectionPool
}

// NewRoundService creates a new RoundService.
func NewRoundService(db db.RoundDB, natsConnectionPool *nats.NatsConnectionPool) *RoundService {
	return &RoundService{
		db:                 db,
		natsConnectionPool: natsConnectionPool,
	}
}

// GetRounds retrieves all rounds.
func (s *RoundService) GetRounds(ctx context.Context) ([]*models.Round, error) {
	return s.db.GetRounds(ctx)
}

// GetRound retrieves a specific round by ID.
func (s *RoundService) GetRound(ctx context.Context, roundID int64) (*models.Round, error) {
	return s.db.GetRound(ctx, roundID)
}

// ScheduleRound schedules a new round.
func (s *RoundService) ScheduleRound(ctx context.Context, input models.ScheduleRoundInput) (*models.Round, error) {
	// Perform any necessary validations here
	if input.Title == "" {
		return nil, errors.New("title is required")
	}

	// Create the Round object
	round := &models.Round{
		Title:        input.Title,
		Location:     input.Location,
		EventType:    input.EventType,
		Date:         input.Date,
		Time:         input.Time,
		Finalized:    false,
		CreatorID:    input.DiscordID, // Use DiscordID directly
		State:        models.RoundStateUpcoming,
		Participants: []models.Participant{}, // Initialize the Participants slice
		Scores:       make(map[string]int),   // Initialize the Scores map
	}

	// Call the database layer to create the round
	return s.db.CreateRound(ctx, round)
}

// JoinRound adds a participant to a round.
func (s *RoundService) JoinRound(ctx context.Context, input models.JoinRoundInput) (*models.Round, error) {
	switch input.Response {
	case models.ResponseAccept, models.ResponseTentative:
		// Valid response, proceed
	default:
		return nil, errors.New("invalid response value")
	}

	round, err := s.GetRound(ctx, input.RoundID)
	if err != nil {
		return nil, err
	}
	if round == nil {
		return nil, errors.New("round not found")
	}

	if round.Finalized {
		// Check if the user is an admin
		conn, err := s.natsConnectionPool.GetConnection()
		if err != nil {
			return nil, fmt.Errorf("failed to get NATS connection from pool: %w", err)
		}
		defer s.natsConnectionPool.ReleaseConnection(conn)

		replyTo := conn.NewInbox()
		err = s.natsConnectionPool.Publish("user.get_role", &nats.UserGetRoleEvent{
			DiscordID: input.DiscordID,
			ReplyTo:   replyTo,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to publish user.get_role event: %w", err)
		}

		sub, err := conn.SubscribeSync(replyTo)
		if err != nil {
			return nil, fmt.Errorf("failed to subscribe to reply inbox: %w", err)
		}
		defer sub.Unsubscribe()

		msg, err := sub.NextMsgWithContext(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to receive user role response: %w", err)
		}

		var response nats.UserGetRoleResponse
		err = json.Unmarshal(msg.Data, &response)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal user role response: %w", err)
		}

		if response.Role != models.UserRoleAdmin {
			return nil, errors.New("only admins can add participants to a finalized round")
		}
	}

	// Check if the user is already a participant
	for _, participant := range round.Participants {
		if participant.DiscordID == input.DiscordID {
			return nil, errors.New("user is already a participant")
		}
	}

	// Get the tag number from the leaderboard service using NATS
	conn, err := s.natsConnectionPool.GetConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to get NATS connection from pool: %w", err)
	}
	defer s.natsConnectionPool.ReleaseConnection(conn)

	replyTo := conn.NewInbox()
	err = s.natsConnectionPool.Publish("leaderboard.get_tag_number", &nats.LeaderboardGetTagNumberEvent{
		DiscordID: input.DiscordID,
		ReplyTo:   replyTo,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to publish leaderboard.get_tag_number event: %w", err)
	}

	sub, err := conn.SubscribeSync(replyTo)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to reply inbox: %w", err)
	}
	defer sub.Unsubscribe()

	msg, err := sub.NextMsgWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to receive tag number response: %w", err)
	}

	var response nats.LeaderboardGetTagNumberResponse
	err = json.Unmarshal(msg.Data, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal tag number response: %w", err)
	}

	tagNumber := response.TagNumber // This can be nil if the user doesn't have a tag

	// Create a new participant
	newParticipant := models.Participant{
		DiscordID: input.DiscordID,
		Response:  input.Response,
		TagNumber: tagNumber, // This can be nil if the user doesn't have a tag
	}

	// Add the new participant to the round's Participants slice
	round.Participants = append(round.Participants, newParticipant)

	// Update the round in the database
	if err := s.db.UpdateRound(ctx, round); err != nil {
		return nil, fmt.Errorf("failed to update round: %w", err)
	}

	return round, nil
}

// SubmitScore submits a score for a participant in a round.
func (s *RoundService) SubmitScore(ctx context.Context, input models.SubmitScoreInput) (*models.Round, error) {
	round, err := s.GetRound(ctx, input.RoundID)
	if err != nil {
		return nil, err
	}
	if round == nil {
		return nil, errors.New("round not found")
	}

	if round.State == models.RoundStateFinalized {
		return nil, errors.New("cannot submit score for a finalized round")
	}

	// Check if the participant exists in the round
	participant, err := s.db.FindParticipant(ctx, round.ID, input.DiscordID)
	if err != nil {
		return nil, fmt.Errorf("failed to find participant: %w", err)
	}
	if participant == nil {
		return nil, errors.New("participant not found")
	}

	// Update the score in the Scores map
	if round.Scores == nil {
		round.Scores = make(map[string]int)
	}
	round.Scores[input.DiscordID] = input.Score

	// Update the round in the database
	if err := s.db.UpdateRound(ctx, round); err != nil {
		return nil, fmt.Errorf("failed to update round: %w", err)
	}

	return round, nil
}

// FinalizeAndProcessScores finalizes a round and processes the scores.
func (s *RoundService) FinalizeAndProcessScores(ctx context.Context, roundID int64) (*models.Round, error) {
	round, err := s.GetRound(ctx, roundID)
	if err != nil {
		return nil, err
	}
	if round == nil {
		return nil, errors.New("round not found")
	}

	if round.Finalized {
		return round, nil // Return early if already finalized
	}

	round.State = models.RoundStateFinalized
	round.Finalized = true

	if err := s.db.UpdateRound(ctx, round); err != nil {
		return nil, fmt.Errorf("failed to update round: %w", err)
	}

	// Publish RoundFinalizedEvent to the score module
	err = s.natsConnectionPool.Publish("round.finalized", &nats.RoundFinalizedEvent{
		RoundID: roundID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to publish round.finalized event: %w", err)
	}

	return round, nil
}

// EditRound updates an existing round.
func (s *RoundService) EditRound(ctx context.Context, roundID int64, userID string, input models.EditRoundInput) (*models.Round, error) {
	round, err := s.GetRound(ctx, roundID)
	if err != nil {
		return nil, err
	}
	if round == nil {
		return nil, errors.New("round not found")
	}

	// Check if the user is authorized to edit
	if round.CreatorID != userID {
		// Check if the user is an admin
		conn, err := s.natsConnectionPool.GetConnection()
		if err != nil {
			return nil, fmt.Errorf("failed to get NATS connection from pool: %w", err)
		}
		defer s.natsConnectionPool.ReleaseConnection(conn)

		replyTo := conn.NewInbox()
		err = s.natsConnectionPool.Publish("user.get_role", &nats.UserGetRoleEvent{
			DiscordID: userID,
			ReplyTo:   replyTo,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to publish user.get_role event: %w", err)
		}

		sub, err := conn.SubscribeSync(replyTo)
		if err != nil {
			return nil, fmt.Errorf("failed to subscribe to reply inbox: %w", err)
		}
		defer sub.Unsubscribe()

		msg, err := sub.NextMsgWithContext(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to receive user role response: %w", err)
		}

		var response nats.UserGetRoleResponse
		err = json.Unmarshal(msg.Data, &response)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal user role response: %w", err)
		}

		if response.Role != models.UserRoleAdmin {
			return nil, errors.New("only the round creator or an admin can edit the round")
		}
	}

	round.Title = input.Title
	round.Location = input.Location
	round.EventType = input.EventType
	round.Date = input.Date
	round.Time = input.Time

	if err := s.db.UpdateRound(ctx, round); err != nil {
		return nil, fmt.Errorf("failed to update round: %w", err)
	}

	return round, nil
}

// DeleteRound deletes a round by ID.
func (s *RoundService) DeleteRound(ctx context.Context, roundID int64, userID string) error {
	round, err := s.GetRound(ctx, roundID)
	if err != nil {
		return err
	}
	if round == nil {
		return errors.New("round not found")
	}

	// Check if the user is authorized to delete
	if round.CreatorID != userID {
		// Check if the user is an admin
		conn, err := s.natsConnectionPool.GetConnection()
		if err != nil {
			return fmt.Errorf("failed to get NATS connection from pool: %w", err)
		}
		defer s.natsConnectionPool.ReleaseConnection(conn)

		replyTo := conn.NewInbox()
		err = s.natsConnectionPool.Publish("user.get_role", &nats.UserGetRoleEvent{
			DiscordID: userID,
			ReplyTo:   replyTo,
		})
		if err != nil {
			return fmt.Errorf("failed to publish user.get_role event: %w", err)
		}

		sub, err := conn.SubscribeSync(replyTo)
		if err != nil {
			return fmt.Errorf("failed to subscribe to reply inbox: %w", err)
		}
		defer sub.Unsubscribe()

		msg, err := sub.NextMsgWithContext(ctx)
		if err != nil {
			return fmt.Errorf("failed to receive user role response: %w", err)
		}

		var response nats.UserGetRoleResponse
		err = json.Unmarshal(msg.Data, &response)
		if err != nil {
			return fmt.Errorf("failed to unmarshal user role response: %w", err)
		}

		if response.Role != models.UserRoleAdmin {
			return errors.New("only the round creator or an admin can delete the round")
		}
	}

	return s.db.DeleteRound(ctx, roundID, userID)
}

// UpdateParticipantResponse updates a participant's response in a round.
func (s *RoundService) UpdateParticipantResponse(ctx context.Context, input models.UpdateParticipantResponseInput) (*models.Round, error) {
	round, err := s.db.UpdateParticipantResponse(ctx, input.RoundID, input.DiscordID, input.Response)
	if err != nil {
		return nil, fmt.Errorf("failed to update participant response: %w", err)
	}
	return round, nil
}

// UpdateRoundState updates the state of a round.
func (s *RoundService) UpdateRoundState(ctx context.Context, roundID int64, state models.RoundState) error {
	return s.db.UpdateRoundState(ctx, roundID, state)
}

// CheckForUpcomingRounds checks for upcoming rounds and sends notifications.
func (s *RoundService) CheckForUpcomingRounds(ctx context.Context) error {
	now := time.Now()
	oneHourFromNow := now.Add(time.Hour)

	rounds, err := s.db.GetUpcomingRounds(ctx, now, oneHourFromNow)
	if err != nil {
		return fmt.Errorf("failed to fetch upcoming rounds: %w", err)
	}

	for _, round := range rounds {
		roundTime, err := time.Parse("15:04", round.Time)
		if err != nil {
			return fmt.Errorf("failed to parse round time: %w", err)
		}

		startTime := time.Date(round.Date.Year(), round.Date.Month(), round.Date.Day(), roundTime.Hour(), roundTime.Minute(), 0, 0, time.UTC)
		oneHourBefore := startTime.Add(-time.Hour)

		if now.After(oneHourBefore) && now.Before(startTime) {
			// Send 1-hour notification
			fmt.Printf("Sending 1-hour notification for round %d\n", round.ID)
		}

		if now.After(startTime) {
			// Send round start notification
			fmt.Printf("Sending round start notification for round %d\n", round.ID)

			if err := s.UpdateRoundState(ctx, round.ID, models.RoundStateInProgress); err != nil {
				return fmt.Errorf("failed to update round state: %w", err)
			}
		}
	}

	return nil
}
