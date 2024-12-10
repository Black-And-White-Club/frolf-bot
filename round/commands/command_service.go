// round/commands/command_service.go

package roundcommands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	natsjetstream "github.com/Black-And-White-Club/tcr-bot/nats"
	"github.com/Black-And-White-Club/tcr-bot/round"
	common "github.com/Black-And-White-Club/tcr-bot/round/common"
	roundconverter "github.com/Black-And-White-Club/tcr-bot/round/converter"
	rounddb "github.com/Black-And-White-Club/tcr-bot/round/db"
	roundevents "github.com/Black-And-White-Club/tcr-bot/round/eventhandling"
	roundhelper "github.com/Black-And-White-Club/tcr-bot/round/helpers"
	apimodels "github.com/Black-And-White-Club/tcr-bot/round/models"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// RoundCommandService handles command-related logic for rounds.
type RoundCommandService struct {
	roundDB            rounddb.RoundDB
	converter          roundconverter.RoundConverter // Use the RoundConverter interface
	publisher          message.Publisher
	natsConnectionPool *natsjetstream.NatsConnectionPool
	eventHandler       round.RoundEventHandler
	helper             roundhelper.RoundHelper // Add the RoundHelper field
}

// NewRoundCommandService creates a new RoundCommandService.
func NewRoundCommandService(roundDB rounddb.RoundDB, converter roundconverter.RoundConverter, publisher message.Publisher, eventHandler round.RoundEventHandler) *RoundCommandService { // Inject converter
	return &RoundCommandService{
		roundDB:      roundDB,
		converter:    converter, // Assign the injected converter
		publisher:    publisher,
		eventHandler: eventHandler,
		helper:       &roundhelper.RoundHelperImpl{Converter: converter}, // Inject converter into the helper
	}
}

// ScheduleRound implements the CommandService interface.
func (s *RoundCommandService) ScheduleRound(ctx context.Context, input apimodels.ScheduleRoundInput) (*apimodels.Round, error) {
	if input.Title == "" {
		return nil, errors.New("title is required")
	}

	modelInput := s.converter.ConvertScheduleRoundInputToModel(input)

	round, err := s.roundDB.CreateRound(ctx, modelInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create round: %w", err)
	}

	return s.converter.ConvertModelRoundToStructRound(round), nil
}

// UpdateParticipant implements the CommandService interface.
func (s *RoundCommandService) UpdateParticipant(ctx context.Context, input apimodels.UpdateParticipantResponseInput) (*apimodels.Round, error) {
	participant := s.converter.ConvertUpdateParticipantInputToParticipant(input)

	err := s.roundDB.UpdateParticipant(ctx, input.RoundID, participant)
	if err != nil {
		return nil, fmt.Errorf("failed to update participant response: %w", err)
	}

	return s.helper.GetRound(ctx, s.roundDB, s.converter, input.RoundID) // Add s.converter
}

// JoinRound implements the CommandService interface.
func (s *RoundCommandService) JoinRound(ctx context.Context, input apimodels.JoinRoundInput) (*apimodels.Round, error) {
	switch input.Response {
	case apimodels.ResponseAccept, apimodels.ResponseTentative:
		// Valid response, proceed
	default:
		return nil, errors.New("invalid response value")
	}

	finalized, err := s.roundDB.IsRoundFinalized(ctx, input.RoundID)
	if err != nil {
		return nil, fmt.Errorf("failed to check round finalized status: %w", err)
	}
	if finalized {
		return nil, errors.New("cannot join a finalized round")
	}

	isParticipant, err := s.roundDB.IsUserParticipant(ctx, input.RoundID, input.DiscordID)
	if err != nil {
		return nil, fmt.Errorf("failed to check participant status: %w", err)
	}
	if isParticipant {
		return nil, errors.New("user is already a participant")
	}

	if err := s.publisher.Publish(roundevents.TagNumberRequestedEvent{}.Topic(), message.NewMessage(watermill.NewUUID(), []byte(input.DiscordID))); err != nil {
		log.Printf("Error publishing TagNumberRequestedEvent: %v", err)
		return nil, fmt.Errorf("failed to publish TagNumberRequestedEvent: %w", err)
	}

	modelParticipant := s.converter.ConvertJoinRoundInputToModelParticipant(input)
	var tagNumber int
	modelParticipant.TagNumber = &tagNumber

	err = s.roundDB.UpdateParticipant(ctx, input.RoundID, modelParticipant)
	if err != nil {
		return nil, fmt.Errorf("failed to add participant: %w", err)
	}

	return s.helper.GetRound(ctx, s.roundDB, s.converter, input.RoundID)
}

// SubmitScore implements the CommandService interface.
func (s *RoundCommandService) SubmitScore(ctx context.Context, input apimodels.SubmitScoreInput) error {
	round, err := s.helper.GetRound(ctx, s.roundDB, s.converter, input.RoundID)
	if err != nil {
		return err
	}
	if round == nil {
		return errors.New("round not found")
	}

	if round.State == apimodels.RoundStateFinalized {
		return errors.New("cannot submit score for a finalized round")
	}

	// Create a ScoreSubmittedEvent directly
	event := roundevents.ScoreSubmissionEvent{
		RoundID:   input.RoundID,
		DiscordID: input.DiscordID,
		Score:     input.Score,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal ScoreSubmittedEvent: %w", err)
	}

	if err := s.publisher.Publish(event.Topic(), message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish ScoreSubmittedEvent: %w", err)
	}

	return nil
}

// SetEventHandler sets the RoundEventHandler for the service.
func (s *RoundCommandService) SetEventHandler(handler round.RoundEventHandler) {
	s.eventHandler = handler
}

// ProcessScoreSubmission implements the CommandService interface.
func (s *RoundCommandService) ProcessScoreSubmission(ctx context.Context, event common.ScoreSubmissionEvent, input apimodels.SubmitScoreInput) error {
	modelRound, err := s.helper.GetRound(ctx, s.roundDB, s.converter, input.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get round: %w", err)
	}
	if modelRound == nil {
		return errors.New("round not found")
	}

	modelRound.Scores[event.GetDiscordID()] = event.GetScore()

	if err := s.roundDB.SubmitScore(ctx, event.GetRoundID(), event.GetDiscordID(), event.GetScore()); err != nil { // Use interface methods
		log.Printf("Error updating scores in ProcessScoreSubmission: %v", err)
		return fmt.Errorf("failed to update scores: %w", err)
	}

	if len(modelRound.Scores) == len(modelRound.Participants) {
		go func() {
			if _, err := s.FinalizeAndProcessScores(context.Background(), event.GetRoundID()); err != nil {
				log.Printf("Error automatically finalizing round: %v", err)
			}
		}()
	}

	return nil
}

// FinalizeAndProcessScores implements the CommandService interface.
func (s *RoundCommandService) FinalizeAndProcessScores(ctx context.Context, roundID int64) (*apimodels.Round, error) {
	round, err := s.helper.GetRound(ctx, s.roundDB, s.converter, roundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get round: %w", err)
	}
	if round == nil {
		return nil, errors.New("round not found")
	}

	if round.Finalized {
		return round, nil
	}

	var participantsWithScores []apimodels.ParticipantScore
	for _, participant := range round.Participants {
		score, ok := round.Scores[participant.DiscordID]
		if !ok {
			score = 0
		}
		participantsWithScores = append(participantsWithScores, apimodels.ParticipantScore{
			DiscordID: participant.DiscordID,
			TagNumber: *participant.TagNumber,
			Score:     score,
		})
	}

	event := roundevents.RoundFinalizedEvent{
		RoundID:      roundID,
		Participants: participantsWithScores,
	}

	if err := s.publisher.Publish(event.Topic(), message.NewMessage(watermill.NewUUID(), event.Marshal())); err != nil {
		return nil, fmt.Errorf("failed to publish RoundFinalizedEvent: %w", err)
	}

	if err := s.roundDB.UpdateRoundState(ctx, roundID, s.converter.ConvertRoundStateToModelRoundState(apimodels.RoundStateFinalized)); err != nil {
		return nil, fmt.Errorf("failed to update round state: %w", err)
	}

	return s.helper.GetRound(ctx, s.roundDB, s.converter, roundID)
}

// EditRound implements the CommandService interface.
func (s *RoundCommandService) EditRound(ctx context.Context, roundID int64, discordID string, input apimodels.EditRoundInput) (*apimodels.Round, error) {
	round, err := s.helper.GetRound(ctx, s.roundDB, s.converter, roundID)
	if err != nil {
		return nil, err
	}
	if round == nil {
		return nil, errors.New("round not found")
	}

	modelInput := s.converter.ConvertEditRoundInputToModel(input)

	err = s.roundDB.UpdateRound(ctx, roundID, modelInput)
	if err != nil {
		return nil, fmt.Errorf("failed to update round: %w", err)
	}

	return s.helper.GetRound(ctx, s.roundDB, s.converter, roundID)
}

// DeleteRound implements the CommandService interface.
func (s *RoundCommandService) DeleteRound(ctx context.Context, roundID int64) error {
	round, err := s.helper.GetRound(ctx, s.roundDB, s.converter, roundID)
	if err != nil {
		return err
	}
	if round == nil {
		return errors.New("round not found")
	}

	err = s.roundDB.DeleteRound(ctx, roundID)
	if err != nil {
		return fmt.Errorf("failed to delete round: %w", err)
	}

	return nil
}

// UpdateRoundState implements the CommandService interface.
func (s *RoundCommandService) UpdateRoundState(ctx context.Context, roundID int64, state common.RoundState) error {
	// Convert state to rounddb.RoundState
	modelState := s.converter.ConvertCommonRoundStateToDB(state) // Use the converter

	err := s.roundDB.UpdateRoundState(ctx, roundID, modelState)
	if err != nil {
		return fmt.Errorf("failed to update round state in DB: %w", err)
	}
	return nil
}
