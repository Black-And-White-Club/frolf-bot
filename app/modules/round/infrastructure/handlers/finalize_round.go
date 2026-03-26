package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/metricattrs"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/google/uuid"
)

// HandleAllScoresSubmitted handles the transition from all scores being in to finalizing the round.
func (h *RoundHandlers) HandleAllScoresSubmitted(
	ctx context.Context,
	payload *roundevents.AllScoresSubmittedPayloadV1,
) ([]handlerwrapper.Result, error) {
	return h.finalizeRound(ctx, payload.GuildID, payload.RoundID, "all_scores_submitted")
}

// HandleRoundFinalizeRequested handles explicit finalize commands (for example,
// when a native Discord scheduled event is manually/completely ended).
func (h *RoundHandlers) HandleRoundFinalizeRequested(
	ctx context.Context,
	payload *roundevents.RoundFinalizeRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	return h.finalizeRound(ctx, payload.GuildID, payload.RoundID, "finalize_requested")
}

func (h *RoundHandlers) finalizeRound(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	roundID sharedtypes.RoundID,
	source string,
) ([]handlerwrapper.Result, error) {
	req := &roundtypes.FinalizeRoundInput{
		GuildID: guildID,
		RoundID: roundID,
	}

	finalizeResult, err := h.service.FinalizeRound(ctx, req)
	if err != nil {
		return nil, err
	}

	if finalizeResult.Failure != nil {
		h.logger.WarnContext(ctx, "backend round finalization failed",
			attr.String("source", source),
			attr.String("round_id", roundID.String()),
			attr.Any("failure", *finalizeResult.Failure),
		)
		return []handlerwrapper.Result{
			{
				Topic: roundevents.RoundFinalizationErrorV1,
				Payload: &roundevents.RoundFinalizationErrorPayloadV1{
					GuildID: guildID,
					RoundID: roundID,
					Error:   (*finalizeResult.Failure).Error(),
				},
			},
		}, nil
	}

	if finalizeResult.Success == nil {
		return nil, sharedtypes.ValidationError{Message: "unexpected result from service: both success and failure are nil"}
	}

	resultData := *finalizeResult.Success
	if resultData.AlreadyFinalized {
		h.logger.InfoContext(ctx, "Round already finalized; republishing finalization events for downstream idempotency",
			attr.String("source", source),
			attr.String("round_id", roundID.String()),
			attr.String("guild_id", string(guildID)),
		)
	}

	if resultData.Round == nil {
		return nil, sharedtypes.ValidationError{Message: "unexpected finalize result: round is nil"}
	}

	fetchedRound := resultData.Round
	discordFinalizationPayload := &roundevents.RoundFinalizedDiscordPayloadV1{
		GuildID:        guildID,
		RoundID:        roundID,
		Title:          fetchedRound.Title,
		StartTime:      fetchedRound.StartTime,
		Location:       fetchedRound.Location,
		Participants:   resultData.Participants,
		Teams:          resultData.Teams,
		EventMessageID: fetchedRound.EventMessageID,
		DiscordEventID: fetchedRound.DiscordEventID,
	}

	backendFinalizationPayload := &roundevents.RoundFinalizedPayloadV1{
		GuildID:   guildID,
		RoundID:   roundID,
		RoundData: *fetchedRound,
	}

	// 0. Resolve ClubID securely from GuildID (Backend Edge Enrichment)
	if h.clubResolver != nil && string(guildID) != "" {
		clubUUID, err := h.clubResolver.GetClubIDForGuild(ctx, string(guildID))
		if err == nil && clubUUID != uuid.Nil {
			backendFinalizationPayload.ClubID = &clubUUID
			ctx = metricattrs.WithClubID(ctx, clubUUID)
		} else {
			h.logger.WarnContext(ctx, "failed to resolve club id for round finalization",
				attr.String("guild_id", string(guildID)),
				attr.Error(err),
			)
		}
	}

	results := []handlerwrapper.Result{
		{
			Topic:   roundevents.RoundFinalizedDiscordV1,
			Payload: discordFinalizationPayload,
			Metadata: map[string]string{
				"discord_message_id": fetchedRound.EventMessageID,
			},
		},
		{
			Topic:   roundevents.RoundFinalizedV2,
			Payload: backendFinalizationPayload,
		},
	}

	// Add both legacy GuildID and internal ClubUUID scoped versions for PWA/NATS transition.
	return h.addParallelIdentityResults(ctx, results, roundevents.RoundFinalizedV2, guildID), nil
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

	// Map result to ensure correct event payload structure
	mappedResult := result.Map(
		func(r *roundtypes.Round) any {
			scores := make([]sharedtypes.ScoreInfo, len(r.Participants))
			for i, p := range r.Participants {
				score := sharedtypes.Score(0)
				if p.Score != nil {
					score = *p.Score
				}
				scores[i] = sharedtypes.ScoreInfo{
					UserID:    p.UserID,
					Score:     score,
					TagNumber: p.TagNumber,
					TeamID:    p.TeamID,
					IsDNF:     p.IsDNF,
				}
			}

			// Determine round mode based on participants or teams if available
			mode := sharedtypes.RoundModeSingles
			if len(r.Teams) > 0 {
				mode = sharedtypes.RoundModeDoubles
			}

			return &sharedevents.ProcessRoundScoresRequestedPayloadV1{
				GuildID:      r.GuildID,
				RoundID:      r.ID,
				Scores:       scores,
				Overwrite:    true,
				RoundMode:    mode,
				Participants: r.Participants,
			}
		},
		func(err error) any {
			return &roundevents.RoundFinalizationErrorPayloadV1{
				GuildID: payload.GuildID,
				RoundID: payload.RoundID,
				Error:   err.Error(),
			}
		},
	)

	return mapOperationResult(mappedResult,
		sharedevents.ProcessRoundScoresRequestedV1,
		roundevents.RoundFinalizationErrorV1,
	), nil
}
