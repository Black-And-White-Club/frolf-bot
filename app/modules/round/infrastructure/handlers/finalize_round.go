package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleAllScoresSubmitted handles the transition from all scores being in to finalizing the round.
func (h *RoundHandlers) HandleAllScoresSubmitted(
	ctx context.Context,
	payload *roundevents.AllScoresSubmittedPayloadV1,
) ([]handlerwrapper.Result, error) {
	finalizeResult, err := h.service.FinalizeRound(ctx, *payload)
	if err != nil {
		return nil, err
	}

	if finalizeResult.Failure != nil {
		h.logger.WarnContext(ctx, "backend round finalization failed",
			attr.Any("failure", finalizeResult.Failure),
		)
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundFinalizationErrorV1, Payload: finalizeResult.Failure},
		}, nil
	}

	if finalizeResult.Success == nil {
		return nil, sharedtypes.ValidationError{Message: "unexpected result from service: both success and failure are nil"}
	}

	// Prepare data for multiple outgoing events
	fetchedRound := &payload.RoundData

	discordFinalizationPayload := &roundevents.RoundFinalizedDiscordPayloadV1{
		GuildID:        payload.GuildID,
		RoundID:        payload.RoundID,
		Title:          fetchedRound.Title,
		StartTime:      fetchedRound.StartTime,
		Location:       fetchedRound.Location,
		Participants:   payload.Participants,
		Teams:          payload.Teams,
		EventMessageID: fetchedRound.EventMessageID,
	}

	backendFinalizationPayload := &roundevents.RoundFinalizedPayloadV1{
		GuildID: payload.GuildID,
		RoundID: payload.RoundID,
		RoundData: roundtypes.Round{
			ID:             fetchedRound.ID,
			Title:          fetchedRound.Title,
			Description:    fetchedRound.Description,
			Location:       fetchedRound.Location,
			StartTime:      fetchedRound.StartTime,
			EventMessageID: fetchedRound.EventMessageID,
			CreatedBy:      fetchedRound.CreatedBy,
			State:          fetchedRound.State,
			Participants:   payload.Participants,
		},
	}

	// We return two separate results. The Discord-bound event needs the message ID
	// metadata to allow the Discord module to update/finalize the correct message.
	return []handlerwrapper.Result{
		{
			Topic:   roundevents.RoundFinalizedDiscordV1,
			Payload: discordFinalizationPayload,
			Metadata: map[string]string{
				"discord_message_id": fetchedRound.EventMessageID,
			},
		},
		{
			Topic:   roundevents.RoundFinalizedV1,
			Payload: backendFinalizationPayload,
		},
	}, nil
}

// HandleRoundFinalized handles the domain event after a round is finalized, notifying the score module.
func (h *RoundHandlers) HandleRoundFinalized(
	ctx context.Context,
	payload *roundevents.RoundFinalizedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.NotifyScoreModule(ctx, *payload)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		sharedevents.ProcessRoundScoresRequestedV1,
		roundevents.RoundFinalizationErrorV1,
	), nil
}
