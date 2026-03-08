package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleRoundStartRequested handles the minimal backend request to start a round.
// The handler uses the DB as the source of truth (service will fetch the round).
func (h *RoundHandlers) HandleRoundStartRequested(
	ctx context.Context,
	payload *roundevents.RoundStartRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	req := &roundtypes.StartRoundRequest{
		GuildID: sharedtypes.GuildID(payload.GuildID),
		RoundID: sharedtypes.RoundID(payload.RoundID),
	}

	result, err := h.service.StartRound(ctx, req)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{
			{
				Topic: roundevents.RoundStartFailedV1,
				Payload: &roundevents.RoundStartFailedPayloadV1{
					GuildID: req.GuildID,
					RoundID: req.RoundID,
					Error:   (*result.Failure).Error(),
				},
			},
		}, nil
	}

	if result.Success == nil {
		return []handlerwrapper.Result{}, nil
	}

	if result.AlreadyStarted {
		return []handlerwrapper.Result{}, nil
	}

	round := *result.Success
	startedPayload := &roundevents.RoundStartedPayloadV1{
		GuildID:   round.GuildID,
		RoundID:   round.ID,
		Title:     round.Title,
		Location:  round.Location,
		StartTime: round.StartTime,
	}

	results := []handlerwrapper.Result{
		{
			Topic:   roundevents.RoundStartedV2,
			Payload: startedPayload,
		},
	}

	// Publish both guild and club scoped start events so PWA-only consumers can react.
	results = h.addParallelIdentityResults(ctx, results, roundevents.RoundStartedV2, req.GuildID)

	// Discord-specific start updates require a target message id.
	if round.EventMessageID != "" {
		eventParticipants := make([]roundevents.RoundParticipantV1, len(round.Participants))
		for i, p := range round.Participants {
			eventParticipants[i] = roundevents.RoundParticipantV1{
				UserID: p.UserID,
				Score:  p.Score,
			}
		}

		results = append(results, handlerwrapper.Result{
			Topic: roundevents.RoundStartedDiscordV2,
			Payload: &roundevents.DiscordRoundStartPayloadV1{
				GuildID:        round.GuildID,
				RoundID:        round.ID,
				Title:          round.Title,
				Location:       round.Location,
				StartTime:      round.StartTime,
				Participants:   eventParticipants,
				EventMessageID: round.EventMessageID,
				DiscordEventID: round.DiscordEventID,
			},
		})
	}

	return results, nil
}
