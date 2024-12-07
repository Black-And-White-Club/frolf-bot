// round/service.go

package round

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/db"
	"github.com/Black-And-White-Club/tcr-bot/events"
	"github.com/Black-And-White-Club/tcr-bot/nats"
	subscribers "github.com/Black-And-White-Club/tcr-bot/round/subs"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// RoundService handles round-related logic and database interactions.
type RoundService struct {
	roundDB            db.RoundDB
	converter          RoundConverter
	publisher          message.Publisher
	natsConnectionPool *nats.NatsConnectionPool // Add this field
}

// NewRoundService creates a new RoundService.
func NewRoundService(roundDB db.RoundDB, natsConnectionPool *nats.NatsConnectionPool, publisher message.Publisher) *RoundService {
	return &RoundService{
		roundDB:            roundDB,
		converter:          &DefaultRoundConverter{},
		publisher:          publisher,
		natsConnectionPool: natsConnectionPool, // Initialize the field
	}
}

// GetRounds retrieves all rounds.
func (s *RoundService) GetRounds(ctx context.Context) ([]*Round, error) {
	modelRounds, err := s.roundDB.GetRounds(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get rounds: %w", err)
	}

	var apiRounds []*Round
	for _, modelRound := range modelRounds {
		apiRounds = append(apiRounds, s.converter.ConvertModelRoundToStructRound(modelRound))
	}

	return apiRounds, nil
}

// GetRound retrieves a specific round by ID.
func (s *RoundService) GetRound(ctx context.Context, roundID int64) (*Round, error) {
	modelRound, err := s.roundDB.GetRound(ctx, roundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get round: %w", err)
	}

	return s.converter.ConvertModelRoundToStructRound(modelRound), nil
}

func (s *RoundService) HasActiveRounds(ctx context.Context) (bool, error) {
	// 1. Check for upcoming rounds within the next hour
	now := time.Now()
	oneHourFromNow := now.Add(time.Hour)
	upcomingRounds, err := s.roundDB.GetUpcomingRounds(ctx, now, oneHourFromNow)
	if err != nil {
		return false, fmt.Errorf("failed to get upcoming rounds: %w", err)
	}
	if len(upcomingRounds) > 0 {
		return true, nil // There are upcoming rounds
	}

	// 2. If no upcoming rounds, check for rounds in progress
	rounds, err := s.GetRounds(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get rounds: %w", err)
	}
	for _, round := range rounds {
		if round.State == RoundStateInProgress {
			return true, nil // There's a round in progress
		}
	}

	// 3. No active rounds found
	return false, nil
}

// ScheduleRound schedules a new round.
func (s *RoundService) ScheduleRound(ctx context.Context, input ScheduleRoundInput) (*Round, error) {
	if input.Title == "" {
		return nil, errors.New("title is required")
	}

	// Convert round.ScheduleRoundInput to models.ScheduleRoundInput using the converter
	modelInput := s.converter.ConvertScheduleRoundInputToModel(input)

	round, err := s.roundDB.CreateRound(ctx, modelInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create round: %w", err)
	}

	return s.converter.ConvertModelRoundToStructRound(round), nil
}

// UpdateParticipant updates a participant's response in a round.
func (s *RoundService) UpdateParticipant(ctx context.Context, input UpdateParticipantResponseInput) (*Round, error) {
	// Use the converter to create the models.Participant
	participant := s.converter.ConvertUpdateParticipantInputToParticipant(input)

	err := s.roundDB.UpdateParticipant(ctx, input.RoundID, participant)
	if err != nil {
		return nil, fmt.Errorf("failed to update participant response: %w", err)
	}

	return s.GetRound(ctx, input.RoundID)
}

// JoinRound adds a participant to a round.
func (s *RoundService) JoinRound(ctx context.Context, input JoinRoundInput) (*Round, error) {
	switch input.Response {
	case ResponseAccept, ResponseTentative:
		// Valid response, proceed
	default:
		return nil, errors.New("invalid response value")
	}

	round, err := s.GetRound(ctx, input.RoundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get round: %w", err)
	}
	if round == nil {
		return nil, errors.New("round not found")
	}

	if round.Finalized {
		return nil, errors.New("cannot join a finalized round")
	}

	// Check if the user is already a participant
	for _, participant := range round.Participants {
		if participant.DiscordID == input.DiscordID {
			return nil, errors.New("user is already a participant")
		}
	}

	// --- Publish TagNumberRequestedEvent ---
	if err := s.publisher.Publish(TagNumberRequestedEvent{}.Topic(), message.NewMessage(watermill.NewUUID(), []byte(input.DiscordID))); err != nil {
		log.Printf("Error publishing TagNumberRequestedEvent: %v", err)
		return nil, fmt.Errorf("failed to publish TagNumberRequestedEvent: %w", err)
	}

	// --- Add participant to the round ---
	modelParticipant := s.converter.ConvertJoinRoundInputToModelParticipant(input) // Direct conversion
	// Set a default tag number (e.g., 0 or nil) if no tag is returned
	// The event handler will update this if a tag is found
	var tagNumber int // Or *int if you prefer a pointer
	modelParticipant.TagNumber = &tagNumber

	err = s.roundDB.UpdateParticipant(ctx, input.RoundID, modelParticipant)
	if err != nil {
		return nil, fmt.Errorf("failed to add participant: %w", err)
	}

	return s.GetRound(ctx, input.RoundID)
}

// SubmitScore submits a score for a participant in a round.
func (s *RoundService) SubmitScore(ctx context.Context, input SubmitScoreInput) error {
	round, err := s.GetRound(ctx, input.RoundID)
	if err != nil {
		return err // Corrected return statement
	}
	if round == nil {
		return errors.New("round not found") // Corrected return statement
	}

	if round.State == RoundStateFinalized {
		return errors.New("cannot submit score for a finalized round") // Corrected return statement
	}

	// --- Publish ScoreSubmittedEvent ---
	event := ScoreSubmittedEvent{
		RoundID: input.RoundID,
		UserID:  input.DiscordID,
		Score:   input.Score,
	}

	payload, err := json.Marshal(event) // Marshal the event into a JSON payload
	if err != nil {
		return fmt.Errorf("failed to marshal ScoreSubmittedEvent: %w", err)
	}

	if err := s.publisher.Publish(event.Topic(), message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish ScoreSubmittedEvent: %w", err)
	}

	return nil
}

// ProcessScoreSubmission handles the logic for processing a submitted score.
func (s *RoundService) ProcessScoreSubmission(ctx context.Context, event ScoreSubmittedEvent) error {
	// Fetch the round
	modelRound, err := s.roundDB.GetRound(ctx, event.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get round: %w", err)
	}
	if modelRound == nil {
		return errors.New("round not found")
	}

	// Update the score in the Round struct
	modelRound.Scores[event.UserID] = event.Score

	// Update the score in the database
	if err := s.roundDB.SubmitScore(ctx, event.RoundID, event.UserID, event.Score); err != nil {
		log.Printf("Error updating scores in ProcessScoreSubmission: %v", err)
		return fmt.Errorf("failed to update scores: %w", err)
	}

	// Check if all scores are submitted
	if len(modelRound.Scores) == len(modelRound.Participants) {
		// Trigger FinalizeAndProcessScores asynchronously
		go func() {
			if _, err := s.FinalizeAndProcessScores(context.Background(), event.RoundID); err != nil {
				log.Printf("Error automatically finalizing round: %v", err)
				// Consider more robust error handling here, e.g.,
				// - Publish an error event
				// - Retry finalization later
				// - Send a notification to an admin
			}
		}()
	}

	return nil
}

// FinalizeAndProcessScores finalizes a round and processes the scores.
func (s *RoundService) FinalizeAndProcessScores(ctx context.Context, roundID int64) (*Round, error) {
	round, err := s.GetRound(ctx, roundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get round: %w", err)
	}
	if round == nil {
		return nil, errors.New("round not found")
	}

	if round.Finalized {
		return round, nil // Return early if already finalized
	}

	// --- Construct ParticipantScores ---
	var participantsWithScores []ParticipantScore
	for _, participant := range round.Participants {
		score, ok := round.Scores[participant.DiscordID]
		if !ok {
			// Handle missing score for a participant (e.g., set a default score)
			score = 0 // Or any other default value
		}
		participantsWithScores = append(participantsWithScores, ParticipantScore{
			DiscordID: participant.DiscordID,
			TagNumber: *participant.TagNumber, // Assuming TagNumber is always present at this point
			Score:     score,
		})
	}

	// --- Publish RoundFinalizedEvent ---
	event := RoundFinalizedEvent{
		RoundID:      roundID,
		Participants: participantsWithScores,
	}

	if err := s.publisher.Publish(event.Topic(), message.NewMessage(watermill.NewUUID(), event.Marshal())); err != nil {
		// Handle the publishing error, e.g., log and return an error
		return nil, fmt.Errorf("failed to publish RoundFinalizedEvent: %w", err)
	}

	// --- Update round state ---
	if err := s.roundDB.UpdateRoundState(ctx, roundID, s.converter.ConvertRoundStateToModelRoundState(RoundStateFinalized)); err != nil {
		return nil, fmt.Errorf("failed to update round state: %w", err)
	}

	return s.GetRound(ctx, roundID)
}

// EditRound updates an existing round.
func (s *RoundService) EditRound(ctx context.Context, roundID int64, discordID string, input EditRoundInput) (*Round, error) {
	round, err := s.GetRound(ctx, roundID)
	if err != nil {
		return nil, err
	}
	if round == nil {
		return nil, errors.New("round not found")
	}

	// Convert the input to models.EditRoundInput using your RoundConverter
	modelInput := s.converter.ConvertEditRoundInputToModel(input)

	err = s.roundDB.UpdateRound(ctx, roundID, modelInput)
	if err != nil {
		return nil, fmt.Errorf("failed to update round: %w", err)
	}

	return s.GetRound(ctx, roundID)
}

// DeleteRound deletes a round by ID.
func (s *RoundService) DeleteRound(ctx context.Context, roundID int64) error { // No userID parameter
	round, err := s.GetRound(ctx, roundID)
	if err != nil {
		return err
	}
	if round == nil {
		return errors.New("round not found")
	}

	err = s.roundDB.DeleteRound(ctx, roundID) // Now matches the db layer signature
	if err != nil {
		return fmt.Errorf("failed to delete round: %w", err)
	}

	return nil
}

// CheckForUpcomingRounds checks for upcoming rounds and sends notifications.
func (s *RoundService) CheckForUpcomingRounds(ctx context.Context) error {
	now := time.Now()
	oneHourFromNow := now.Add(time.Hour)

	modelRounds, err := s.roundDB.GetUpcomingRounds(ctx, now, oneHourFromNow)
	if err != nil {
		return fmt.Errorf("failed to fetch upcoming rounds: %w", err)
	}

	var rounds []*Round
	for _, modelRound := range modelRounds {
		rounds = append(rounds, s.converter.ConvertModelRoundToStructRound(modelRound))
	}

	for _, round := range rounds {
		roundTime, err := time.Parse("15:04", round.Time)
		if err != nil {
			return fmt.Errorf("failed to parse round time: %w", err)
		}

		startTime := time.Date(round.Date.Year(), round.Date.Month(), round.Date.Day(), roundTime.Hour(), roundTime.Minute(), 0, 0, time.UTC)
		oneHourBefore := startTime.Add(-time.Hour)
		thirtyMinutesBefore := startTime.Add(-30 * time.Minute)

		if now.After(oneHourBefore) && now.Before(thirtyMinutesBefore) {
			msg := message.NewMessage(watermill.NewUUID(), nil)
			err := s.publisher.Publish("round_starting_one_hour", msg)
			if err != nil {
				return fmt.Errorf("failed to publish RoundStartingOneHourEvent: %w", err)
			}
		}

		if now.After(thirtyMinutesBefore) && now.Before(startTime) {
			msg := message.NewMessage(watermill.NewUUID(), nil)
			err := s.publisher.Publish("round_starting_thirty_minutes", msg)
			if err != nil {
				return fmt.Errorf("failed to publish RoundStartingThirtyMinutesEvent: %w", err)
			}
		}

		if now.After(startTime) {
			msg := message.NewMessage(watermill.NewUUID(), nil)
			err := s.publisher.Publish("round_started", msg)
			if err != nil {
				return fmt.Errorf("failed to publish RoundStartedEvent: %w", err)
			}
		}
	}

	return nil
}

// UpdateRoundState updates the state of a round.
func (s *RoundService) UpdateRoundState(ctx context.Context, roundID int64, state RoundState) error {
	// Convert the RoundState to models.RoundState
	modelState := s.converter.ConvertRoundStateToModelRoundState(state)

	err := s.roundDB.UpdateRoundState(ctx, roundID, modelState)
	if err != nil {
		return fmt.Errorf("failed to update round state in DB: %w", err)
	}
	return nil
}

func (s *RoundService) StartNATSSubscribers(ctx context.Context, handler *RoundEventHandler) error {
	subscriber, err := events.NewSubscriber(s.natsConnectionPool.GetURL(), watermill.NewStdLogger(false, false)) // Use s.natsConnectionPool.url
	if err != nil {
		return fmt.Errorf("failed to create subscriber: %w", err)
	}

	// Subscribe to round events
	if err := subscribers.SubscribeToRoundEvents(ctx, subscriber, handler); err != nil {
		return fmt.Errorf("failed to subscribe to round events: %w", err)
	}

	// Subscribe to participant events
	if err := subscribers.SubscribeToParticipantEvents(ctx, subscriber, handler); err != nil {
		return fmt.Errorf("failed to subscribe to participant events: %w", err)
	}

	// Subscribe to score events
	if err := subscribers.SubscribeToScoreEvents(ctx, subscriber, handler); err != nil {
		return fmt.Errorf("failed to subscribe to score events: %w", err)
	}

	return nil
}
