package roundservice

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/app/shared/logging"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// GetRound retrieves the round from the database and publishes a round.fetched event.
func (s *RoundService) GetRound(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundUpdateValidatedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundUpdateValidatedPayload: %w", err)
	}

	dbRound, err := s.RoundDB.GetRound(ctx, eventPayload.RoundUpdateRequestPayload.RoundID)
	if err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to get round", map[string]interface{}{
			"round_id": eventPayload.RoundUpdateRequestPayload.RoundID,
		})
		// Consider using a more specific error event like round.not_found
		return s.publishRoundUpdateError(msg, eventPayload.RoundUpdateRequestPayload, err)
	}

	// Convert rounddb.Round to roundtypes.Round
	rtRound := roundtypes.Round{
		ID:           dbRound.ID,
		Title:        dbRound.Title,
		Description:  &dbRound.Description,
		Location:     &dbRound.Location,
		EventType:    dbRound.EventType,
		StartTime:    &dbRound.StartTime,
		Finalized:    dbRound.Finalized,
		CreatedBy:    dbRound.CreatorID,
		State:        roundtypes.RoundState(dbRound.State),
		Participants: convertParticipants(dbRound.Participants),
	}

	// If the round is found, publish a "round.fetched" event
	if err := s.publishEvent(msg, roundevents.RoundFetched, roundevents.RoundFetchedPayload{
		Round:                     rtRound,
		RoundUpdateRequestPayload: eventPayload.RoundUpdateRequestPayload,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.fetched event", map[string]interface{}{
			"error": err,
		})
		return fmt.Errorf("failed to publish round.fetched event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Round fetched from database", map[string]interface{}{
		"round_id": eventPayload.RoundUpdateRequestPayload.RoundID,
	})
	return nil
}

// convertParticipants converts []rounddb.Participant to []roundtypes.RoundParticipant.
func convertParticipants(dbParticipants []rounddb.Participant) []roundtypes.RoundParticipant {
	rtParticipants := make([]roundtypes.RoundParticipant, len(dbParticipants))
	for i, dbP := range dbParticipants {
		rtParticipants[i] = roundtypes.RoundParticipant{
			DiscordID: dbP.DiscordID,
			Response:  roundtypes.Response(dbP.Response), // Convert Response
			TagNumber: *dbP.TagNumber,
			Score:     dbP.Score,
		}
	}
	return rtParticipants
}

// CheckRoundExists checks if the round exists in the database and publishes a round.to.delete.fetched event.
func (s *RoundService) CheckRoundExists(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundDeleteValidatedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundDeleteValidatedPayload: %w", err)
	}

	dbRound, err := s.RoundDB.GetRound(ctx, eventPayload.RoundDeleteRequestPayload.RoundID)
	if err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to get round", map[string]interface{}{
			"round_id": eventPayload.RoundDeleteRequestPayload.RoundID,
			"error":    err,
		})
		// Consider using a more specific error event like round.not_found
		return s.publishRoundDeleteError(msg, eventPayload.RoundDeleteRequestPayload, err)
	}

	// Convert rounddb.Round to roundtypes.Round
	rtRound := roundtypes.Round{
		ID:           dbRound.ID,
		Title:        dbRound.Title,
		Description:  &dbRound.Description,
		Location:     &dbRound.Location,
		EventType:    dbRound.EventType,
		StartTime:    &dbRound.StartTime,
		Finalized:    dbRound.Finalized,
		CreatedBy:    dbRound.CreatorID,
		State:        roundtypes.RoundState(dbRound.State),
		Participants: convertParticipants(dbRound.Participants),
	}

	// If the round is found, publish a "round.to.delete.fetched" event
	if err := s.publishEvent(msg, roundevents.RoundToDeleteFetched, roundevents.RoundToDeleteFetchedPayload{
		Round:                     rtRound,
		RoundDeleteRequestPayload: eventPayload.RoundDeleteRequestPayload,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.to.delete.fetched event", map[string]interface{}{
			"error": err,
		})
		return fmt.Errorf("failed to publish round.to.delete.fetched event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Round found in database", map[string]interface{}{
		"round_id": eventPayload.RoundDeleteRequestPayload.RoundID,
	})
	return nil
}
