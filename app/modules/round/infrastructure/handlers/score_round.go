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
	result, err := h.service.ValidateScoreUpdateRequest(ctx, &roundtypes.ScoreUpdateRequest{
		GuildID: payload.GuildID,
		RoundID: payload.RoundID,
		UserID:  payload.UserID,
		Score:   payload.Score,
	})
	if err != nil {
		return nil, err
	}

	// Map result to event payloads ensuring correct structure
	mappedResult := result.Map(
		func(req *roundtypes.ScoreUpdateRequest) any {
			return &roundevents.ScoreUpdateValidatedPayloadV1{
				GuildID: req.GuildID, // Set top-level GuildID
				ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayloadV1{
					GuildID: req.GuildID,
					RoundID: req.RoundID,
					UserID:  req.UserID,
					Score:   req.Score,
				},
			}
		},
		func(err error) any {
			return &roundevents.RoundScoreUpdateErrorPayloadV1{
				GuildID:            payload.GuildID,
				ScoreUpdateRequest: payload,
				Error:              err.Error(),
			}
		},
	)

	return mapOperationResult(mappedResult,
		roundevents.RoundScoreUpdateValidatedV1,
		roundevents.RoundScoreUpdateErrorV1,
	), nil
}

// HandleScoreUpdateValidated processes a validated score update and applies it to the round.
func (h *RoundHandlers) HandleScoreUpdateValidated(
	ctx context.Context,
	payload *roundevents.ScoreUpdateValidatedPayloadV1,
) ([]handlerwrapper.Result, error) {
	var score sharedtypes.Score
	if payload.ScoreUpdateRequestPayload.Score != nil {
		score = *payload.ScoreUpdateRequestPayload.Score
	}

	result, err := h.service.UpdateParticipantScore(ctx, &roundtypes.ScoreUpdateRequest{
		GuildID: payload.GuildID,
		RoundID: payload.ScoreUpdateRequestPayload.RoundID,
		UserID:  payload.ScoreUpdateRequestPayload.UserID,
		Score:   payload.ScoreUpdateRequestPayload.Score,
	})
	if err != nil {
		return nil, err
	}

	results := mapOperationResult(result.Map(
		func(s *roundtypes.ScoreUpdateResult) any {
			return &roundevents.ParticipantScoreUpdatedPayloadV1{
				GuildID:        s.GuildID,
				RoundID:        s.RoundID,
				UserID:         payload.ScoreUpdateRequestPayload.UserID,
				Score:          score,
				EventMessageID: s.EventMessageID,
				Participants:   s.UpdatedParticipants,
			}
		},
		func(f error) any {
			return &roundevents.RoundScoreUpdateErrorPayloadV1{
				GuildID: payload.GuildID,
				ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayloadV1{
					GuildID: payload.GuildID,
					RoundID: payload.ScoreUpdateRequestPayload.RoundID,
					UserID:  payload.ScoreUpdateRequestPayload.UserID,
					Score:   payload.ScoreUpdateRequestPayload.Score,
				},
				Error: f.Error(),
			}
		},
	),
		roundevents.RoundParticipantScoreUpdatedV1,
		roundevents.RoundScoreUpdateErrorV1,
	)

	// Add both legacy GuildID and internal ClubUUID scoped versions for PWA/NATS transition
	results = h.addParallelIdentityResults(ctx, results, roundevents.RoundParticipantScoreUpdatedV1, payload.GuildID)

	return results, nil
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

	updates := make([]roundtypes.ScoreUpdateRequest, 0, len(payload.Updates))
	for _, u := range payload.Updates {
		var score sharedtypes.Score
		if u.Score != nil {
			score = *u.Score
		}
		updates = append(updates, roundtypes.ScoreUpdateRequest{
			GuildID: u.GuildID,
			RoundID: u.RoundID,
			UserID:  u.UserID,
			Score:   &score,
		})
	}

	opResult, err := h.service.UpdateParticipantScoresBulk(ctx, &roundtypes.BulkScoreUpdateRequest{
		GuildID: payload.GuildID,
		RoundID: payload.RoundID,
		Updates: updates,
	})
	if err != nil {
		return nil, err
	}

	resultsOut := mapOperationResult(
		opResult,
		roundevents.RoundScoresBulkUpdatedV1,
		roundevents.RoundScoreUpdateErrorV1,
	)

	if !opResult.IsSuccess() {
		return resultsOut, nil
	}

	// 🔑 Extract authoritative success payload
	successPayload := opResult.Success

	userIDs := make([]sharedtypes.DiscordID, 0, len(payload.Updates))
	for _, u := range payload.Updates {
		userIDs = append(userIDs, u.UserID)
	}

	sharedPayload := &sharedevents.ScoreBulkUpdatedPayloadV1{
		GuildID:        (*successPayload).GuildID,
		RoundID:        (*successPayload).RoundID,
		AppliedCount:   len(payload.Updates),
		FailedCount:    0,
		TotalRequested: len(payload.Updates),
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
		attr.String("guild_id", string(payload.GuildID)),
	)

	// 1. Ask the domain service if the round is ready
	result, err := h.service.CheckAllScoresSubmitted(ctx, &roundtypes.CheckAllScoresSubmittedRequest{
		GuildID: payload.GuildID,
		RoundID: payload.RoundID,
		UserID:  payload.UserID,
	})
	if err != nil {
		h.logger.ErrorContext(ctx, "CheckAllScoresSubmitted failed with error", attr.Error(err))
		return nil, err
	}

	if result.Success != nil {
		h.logger.InfoContext(ctx, "CheckAllScoresSubmitted success",
			attr.Bool("is_complete", (*result.Success).IsComplete),
			attr.Int("participant_count", len((*result.Success).Participants)),
		)
	} else {
		h.logger.WarnContext(ctx, "CheckAllScoresSubmitted returned no success result")
	}

	if result.Failure != nil {
		h.logger.WarnContext(ctx, "CheckAllScoresSubmitted returned failure", attr.Any("failure", result.Failure))
	}

	// 2. Business failure
	if result.Failure != nil {
		// Map to event payload ensuring context fields
		failurePayload := &roundevents.RoundFinalizationFailedPayloadV1{
			GuildID: payload.GuildID,
			RoundID: payload.RoundID,
			Error:   fmt.Sprintf("%v", result.Failure),
		}
		return []handlerwrapper.Result{
			{
				Topic:   roundevents.RoundFinalizationFailedV1,
				Payload: failurePayload,
			},
		}, nil
	}

	// 3. All scores submitted
	if result.Success != nil && (*result.Success).IsComplete {
		allScoresData := *result.Success
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

		// Convert domain result to event payload
		eventPayload := &roundevents.AllScoresSubmittedPayloadV1{
			GuildID:        payload.GuildID,
			RoundID:        payload.RoundID,
			EventMessageID: payload.EventMessageID,
			RoundData:      *allScoresData.Round,
			Participants:   allScoresData.Participants,
			Teams:          allScoresData.Teams,
		}

		return []handlerwrapper.Result{
			{
				Topic:   roundevents.RoundAllScoresSubmittedV1,
				Payload: eventPayload,
			},
		}, nil
	}

	// 4. Partial submission (unchanged behavior)
	if result.Success != nil && !(*result.Success).IsComplete {
		partialData := *result.Success
		scoredTeams := make([]roundtypes.NormalizedTeam, 0)
		remainingParticipants := make([]roundtypes.Participant, 0)

		for _, p := range partialData.Participants {
			if p.Score != nil && p.TeamID != uuid.Nil {
				// Handle guest users (empty UserID)
				var userIDPtr *sharedtypes.DiscordID
				if p.UserID != "" {
					userIDPtr = &p.UserID
				}
				rawName := p.RawName
				if rawName == "" && p.UserID != "" {
					rawName = string(p.UserID)
				}
				scoredTeams = append(scoredTeams, roundtypes.NormalizedTeam{
					TeamID:  p.TeamID,
					Members: []roundtypes.TeamMember{{UserID: userIDPtr, RawName: rawName}},
					Total:   int(*p.Score),
				})
			} else {
				remainingParticipants = append(remainingParticipants, p)
			}
		}

		// Convert domain result to event payload
		// Note: ScoresPartiallySubmittedPayloadV1 expects Scores and Participants
		scores := make([]roundevents.ParticipantScoreV1, 0)
		// ... logic to populate scores if needed, but previously we just returned partialData
		// The previous code casted result.Success to *roundevents.ScoresPartiallySubmittedPayloadV1
		// Now result.Success is AllScoresSubmittedResult (shared type).

		eventPayload := &roundevents.ScoresPartiallySubmittedPayloadV1{
			GuildID:        payload.GuildID,
			RoundID:        payload.RoundID,
			UserID:         payload.UserID,
			Score:          payload.Score,
			EventMessageID: payload.EventMessageID,
			Scores:         scores, // We need to populate this if consumers use it
			Participants:   remainingParticipants,
			Teams:          scoredTeams,
		}

		return []handlerwrapper.Result{
			{
				Topic:   roundevents.RoundScoresPartiallySubmittedV1,
				Payload: eventPayload,
			},
		}, nil
	}

	return nil, fmt.Errorf("unexpected result from CheckAllScoresSubmitted")
}
