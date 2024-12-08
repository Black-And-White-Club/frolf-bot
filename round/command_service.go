// round/command_service.go
package round

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/Black-And-White-Club/tcr-bot/nats"
	rounddb "github.com/Black-And-White-Club/tcr-bot/round/db"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// RoundCommandService handles command-related logic for rounds.
type RoundCommandService struct {
	roundDB            rounddb.RoundDB
	converter          RoundConverter
	publisher          message.Publisher
	natsConnectionPool *nats.NatsConnectionPool
	PlayerAddedToRoundEventHandler
	TagNumberRetrievedEventHandler
	RoundStartedEventHandler
	RoundStartingOneHourEventHandler
	RoundStartingThirtyMinutesEventHandler
	RoundUpdatedEventHandler
	RoundDeletedEventHandler
	RoundFinalizedEventHandler
	ScoreSubmittedEventHandler
	RoundCreateEventHandler
}

// NewRoundCommandService creates a new RoundCommandService.
func NewRoundCommandService(roundDB rounddb.RoundDB, publisher message.Publisher) *RoundCommandService {
	return &RoundCommandService{
		roundDB:   roundDB,
		converter: &DefaultRoundConverter{},
		publisher: publisher,
	}
}

// ScheduleRound schedules a new round.
func (s *RoundCommandService) ScheduleRound(ctx context.Context, input ScheduleRoundInput) (*Round, error) {
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
func (s *RoundCommandService) UpdateParticipant(ctx context.Context, input UpdateParticipantResponseInput) (*Round, error) {
	// Use the converter to create the models.Participant
	participant := s.converter.ConvertUpdateParticipantInputToParticipant(input)

	err := s.roundDB.UpdateParticipant(ctx, input.RoundID, participant)
	if err != nil {
		return nil, fmt.Errorf("failed to update participant response: %w", err)
	}

	return s.GetRound(ctx, input.RoundID)
}

// JoinRound adds a participant to a round.
func (s *RoundCommandService) JoinRound(ctx context.Context, input JoinRoundInput) (*Round, error) {
	switch input.Response {
	case ResponseAccept, ResponseTentative:
		// Valid response, proceed
	default:
		return nil, errors.New("invalid response value")
	}

	// Check if the round is finalized
	finalized, err := s.roundDB.IsRoundFinalized(ctx, input.RoundID)
	if err != nil {
		return nil, fmt.Errorf("failed to check round finalized status: %w", err)
	}
	if finalized {
		return nil, errors.New("cannot join a finalized round")
	}

	// Check if the user is already a participant
	isParticipant, err := s.roundDB.IsUserParticipant(ctx, input.RoundID, input.DiscordID)
	if err != nil {
		return nil, fmt.Errorf("failed to check participant status: %w", err)
	}
	if isParticipant {
		return nil, errors.New("user is already a participant")
	}

	// --- Publish TagNumberRequestedEvent ---
	if err := s.publisher.Publish(TagNumberRequestedEvent{}.Topic(), message.NewMessage(watermill.NewUUID(), []byte(input.DiscordID))); err != nil {
		log.Printf("Error publishing TagNumberRequestedEvent: %v", err)
		return nil, fmt.Errorf("failed to publish TagNumberRequestedEvent: %w", err)
	}

	// --- Add participant to the round ---
	modelParticipant := s.converter.ConvertJoinRoundInputToModelParticipant(input)
	var tagNumber int
	modelParticipant.TagNumber = &tagNumber

	err = s.roundDB.UpdateParticipant(ctx, input.RoundID, modelParticipant)
	if err != nil {
		return nil, fmt.Errorf("failed to add participant: %w", err)
	}

	// Fetch and return the updated round (we still need this for the response)
	return getRound(ctx, s.roundDB, s.converter, input.RoundID)
}

// SubmitScore submits a score for a participant in a round.
func (s *RoundCommandService) SubmitScore(ctx context.Context, input SubmitScoreInput) error {
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
func (s *RoundCommandService) ProcessScoreSubmission(ctx context.Context, event ScoreSubmittedEvent) error {
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
func (s *RoundCommandService) FinalizeAndProcessScores(ctx context.Context, roundID int64) (*Round, error) {
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
func (s *RoundCommandService) EditRound(ctx context.Context, roundID int64, discordID string, input EditRoundInput) (*Round, error) {
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
func (s *RoundCommandService) DeleteRound(ctx context.Context, roundID int64) error { // No userID parameter
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

// UpdateRoundState updates the state of a round.
func (s *RoundCommandService) UpdateRoundState(ctx context.Context, roundID int64, state RoundState) error {
	// Convert the RoundState to models.RoundState
	modelState := s.converter.ConvertRoundStateToModelRoundState(state)

	err := s.roundDB.UpdateRoundState(ctx, roundID, modelState)
	if err != nil {
		return fmt.Errorf("failed to update round state in DB: %w", err)
	}
	return nil
}

// GetRound retrieves a specific round by ID. (This is a duplicate from query_service, consider refactoring)
func (s *RoundCommandService) GetRound(ctx context.Context, roundID int64) (*Round, error) {
	modelRound, err := s.roundDB.GetRound(ctx, roundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get round: %w", err)
	}

	return s.converter.ConvertModelRoundToStructRound(modelRound), nil
}
