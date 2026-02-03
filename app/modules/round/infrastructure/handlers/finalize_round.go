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
	req := &roundtypes.FinalizeRoundInput{
		GuildID: payload.GuildID,
		RoundID: payload.RoundID,
	}

	finalizeResult, err := h.service.FinalizeRound(ctx, req)
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
	// Use the result from service as the source of truth
	resultData := *finalizeResult.Success
	fetchedRound := resultData.Round

	discordFinalizationPayload := &roundevents.RoundFinalizedDiscordPayloadV1{
		GuildID:        payload.GuildID,
		RoundID:        payload.RoundID,
		Title:          fetchedRound.Title,
		StartTime:      fetchedRound.StartTime,
		Location:       fetchedRound.Location,
		Participants:   resultData.Participants,
		Teams:          resultData.Teams,
		EventMessageID: fetchedRound.EventMessageID,
	}

	backendFinalizationPayload := &roundevents.RoundFinalizedPayloadV1{
		GuildID:   payload.GuildID,
		RoundID:   payload.RoundID,
		RoundData: *fetchedRound,
	}

	// We return two separate results. The Discord-bound event needs the message ID
	// metadata to allow the Discord module to update/finalize the correct message.
	results := []handlerwrapper.Result{
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
	}

	// Add both legacy GuildID and internal ClubUUID scoped versions for PWA/NATS transition
	results = h.addParallelIdentityResults(ctx, results, roundevents.RoundFinalizedV1, payload.GuildID)

	return results, nil
}

// HandleRoundFinalized handles the domain event after a round is finalized, notifying the score module.
func (h *RoundHandlers) HandleRoundFinalized(
	ctx context.Context,
	payload *roundevents.RoundFinalizedPayloadV1,
) ([]handlerwrapper.Result, error) {
	req := &roundtypes.FinalizeRoundResult{
		Round:        &payload.RoundData,
		Participants: payload.RoundData.Participants,
		// Teams might be missing in payload.RoundData if not populated, but FinalizeRoundResult has it.
		// If payload doesn't have Teams separate, we might leave it nil or extract if possible.
		// RoundFinalizedPayloadV1 seems to rely on RoundData.
	}

	result, err := h.service.NotifyScoreModule(ctx, req)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		sharedevents.ProcessRoundScoresRequestedV1,
		roundevents.RoundFinalizationErrorV1,
	), nil
}
