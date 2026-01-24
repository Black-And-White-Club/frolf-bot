package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/google/uuid"
)

// HandleScoreUpdateRequest handles requests to update a participant's score.
func (h *RoundHandlers) HandleScoreUpdateRequest(
	ctx context.Context,
	payload *roundevents.ScoreUpdateRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.ValidateScoreUpdateRequest(ctx, *payload)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		roundevents.RoundScoreUpdateValidatedV1,
		roundevents.RoundScoreUpdateErrorV1,
	), nil
}

// HandleScoreUpdateValidated processes a validated score update and applies it to the round.
func (h *RoundHandlers) HandleScoreUpdateValidated(
	ctx context.Context,
	payload *roundevents.ScoreUpdateValidatedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.UpdateParticipantScore(ctx, *payload)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		roundevents.RoundParticipantScoreUpdatedV1,
		roundevents.RoundScoreUpdateErrorV1,
	), nil
}

// HandleScoreBulkUpdateRequest handles bulk score overrides for a round.
func (h *RoundHandlers) HandleScoreBulkUpdateRequest(
	ctx context.Context,
	payload *roundevents.ScoreBulkUpdateRequestPayloadV1,
) ([]handlerwrapper.Result, error) {

	if payload == nil {
		return nil, sharedtypes.ValidationError{
			Message: "bulk score update payload is nil",
		}
	}

	// 1. Call service (unchanged contract)
	opResult, err := h.service.UpdateParticipantScoresBulk(ctx, *payload)
	if err != nil {
		return nil, err // infra error → retry
	}

	// 2. Always emit the authoritative domain event
	resultsOut := mapOperationResult(
		opResult,
		roundevents.RoundScoresBulkUpdatedV1,
		roundevents.RoundScoreUpdateErrorV1,
	)

	// 3. If domain failed, stop here
	if !opResult.IsSuccess() {
		return resultsOut, nil
	}

	// 4. SUCCESS CASE ONLY — emit projection / shared event

	appliedCount := len(payload.Updates)

	userIDs := make([]sharedtypes.DiscordID, 0, appliedCount)
	for _, u := range payload.Updates {
		userIDs = append(userIDs, u.UserID)
	}

	sharedPayload := &sharedevents.ScoreBulkUpdatedPayloadV1{
		GuildID:        payload.GuildID,
		RoundID:        payload.RoundID,
		AppliedCount:   appliedCount,
		FailedCount:    0,
		TotalRequested: appliedCount,
		UserIDsApplied: userIDs,
	}

	resultsOut = append(resultsOut, handlerwrapper.Result{
		Topic:   sharedevents.ScoreBulkUpdatedV1,
		Payload: sharedPayload,
		Metadata: map[string]string{
			"event_role":   "projection",
			"derived_from": roundevents.RoundScoresBulkUpdatedV1,
			"consumer":     "ui",
		},
	})

	return resultsOut, nil
}

// HandleParticipantScoreUpdated checks if all scores have been submitted after an update.
func (h *RoundHandlers) HandleParticipantScoreUpdated(
	ctx context.Context,
	payload *roundevents.ParticipantScoreUpdatedPayloadV1,
) ([]handlerwrapper.Result, error) {

	h.logger.InfoContext(ctx, "HandleParticipantScoreUpdated called",
		attr.String("round_id", payload.RoundID.String()),
		attr.String("user_id", string(payload.UserID)),
	)

	// 1. Ask the domain service if the round is ready
	result, err := h.service.CheckAllScoresSubmitted(ctx, *payload)
	if err != nil {
		h.logger.ErrorContext(ctx, "CheckAllScoresSubmitted failed", attr.Error(err))
		return nil, err
	}

	// 2. Business failure
	if result.Failure != nil {
		return []handlerwrapper.Result{
			{
				Topic:   roundevents.RoundFinalizationFailedV1,
				Payload: result.Failure,
			},
		}, nil
	}

	// 3. All scores submitted
	if allScoresData, ok := result.Success.(*roundevents.AllScoresSubmittedPayloadV1); ok {
		scores := make([]sharedtypes.ScoreInfo, 0, len(allScoresData.Participants))

		for _, p := range allScoresData.Participants {
			if p.Score == nil {
				continue
			}

			teamID := uuid.Nil
			if p.TeamID != uuid.Nil {
				teamID = p.TeamID
			}

			scores = append(scores, sharedtypes.ScoreInfo{
				UserID:    p.UserID,
				Score:     *p.Score,
				TagNumber: p.TagNumber,
				TeamID:    teamID,
			})
		}

		return []handlerwrapper.Result{

			{
				Topic:   roundevents.RoundAllScoresSubmittedV1,
				Payload: allScoresData,
			},
		}, nil
	}

	// 4. Partial submission (unchanged behavior)
	if partialData, ok := result.Success.(*roundevents.ScoresPartiallySubmittedPayloadV1); ok {
		scoredTeams := make([]roundtypes.NormalizedTeam, 0)
		remainingParticipants := make([]roundtypes.Participant, 0)

		for _, p := range partialData.Participants {
			if p.Score != nil && p.TeamID != uuid.Nil {
				scoredTeams = append(scoredTeams, roundtypes.NormalizedTeam{
					TeamID:  p.TeamID,
					Members: []roundtypes.TeamMember{{UserID: &p.UserID, RawName: string(p.UserID)}},
					Total:   int(*p.Score),
				})
			} else {
				remainingParticipants = append(remainingParticipants, p)
			}
		}

		partialData.Teams = scoredTeams
		partialData.Participants = remainingParticipants

		return []handlerwrapper.Result{
			{
				Topic:   roundevents.RoundScoresPartiallySubmittedV1,
				Payload: partialData,
			},
		}, nil
	}

	return nil, fmt.Errorf("unexpected result from CheckAllScoresSubmitted")
}
