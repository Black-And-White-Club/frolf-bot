package leaderboardhandlers

import (
	"context"
	"errors"
	"fmt"
	"sort"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/saga"
	"github.com/google/uuid"
)

func (h *LeaderboardHandlers) HandleBatchTagAssignmentRequested(
	ctx context.Context,
	payload *sharedevents.BatchTagAssignmentRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	requests := make([]sharedtypes.TagAssignmentRequest, len(payload.Assignments))
	for i, a := range payload.Assignments {
		requests[i] = sharedtypes.TagAssignmentRequest{
			UserID:    a.UserID,
			TagNumber: a.TagNumber,
		}
	}

	batchUUID, err := uuid.Parse(payload.BatchID)
	if err != nil {
		return nil, fmt.Errorf("invalid batch_id format: %w", err)
	}

	// Switch logic based on source/payload
	if payload.RoundID != nil && payload.Source == sharedtypes.ServiceUpdateSourceProcessScores {
		return h.handleRoundBasedAssignment(ctx, payload, requests)
	}

	result, err := h.service.ExecuteBatchTagAssignment(
		ctx,
		payload.GuildID,
		requests,
		sharedtypes.RoundID(batchUUID),
		sharedtypes.ServiceUpdateSourceAdminBatch,
	)
	if err != nil {
		return nil, err
	}

	if result.IsFailure() {
		var swapErr *leaderboardservice.TagSwapNeededError
		if errors.As(*result.Failure, &swapErr) {
			intentErr := h.sagaCoordinator.ProcessIntent(ctx, saga.SwapIntent{
				UserID:     swapErr.RequestorID,
				CurrentTag: swapErr.CurrentTag,
				TargetTag:  swapErr.TargetTag,
				GuildID:    payload.GuildID,
			})
			return []handlerwrapper.Result{}, intentErr
		}
		return nil, *result.Failure
	}

	replyTo, _ := ctx.Value(handlerwrapper.CtxKeyReplyTo).(string)
	results := h.mapSuccessResults(ctx, payload.GuildID, payload.RequestingUserID, payload.BatchID, requests, sharedtypes.ServiceUpdateSourceAdminBatch, replyTo)

	propagateCorrelationID(ctx, results)

	return results, nil
}

func (h *LeaderboardHandlers) handleRoundBasedAssignment(
	ctx context.Context,
	payload *sharedevents.BatchTagAssignmentRequestedPayloadV1,
	requests []sharedtypes.TagAssignmentRequest,
) ([]handlerwrapper.Result, error) {
	return h.handleRoundBasedAssignmentWithCommandFlow(ctx, payload, requests)
}

func (h *LeaderboardHandlers) handleRoundBasedAssignmentWithCommandFlow(
	ctx context.Context,
	payload *sharedevents.BatchTagAssignmentRequestedPayloadV1,
	requests []sharedtypes.TagAssignmentRequest,
) ([]handlerwrapper.Result, error) {
	if payload.RoundID == nil {
		return nil, fmt.Errorf("round_id is required for round-based assignment")
	}

	// Iterate payload.Assignments directly (not the requests slice) because requests strips
	// FinishRank when converting to TagAssignmentRequest. Both slices are the same length
	// and in the same order.
	participants := make([]leaderboardservice.RoundParticipantInput, 0, len(payload.Assignments))
	for _, r := range payload.Assignments {
		// Pass FinishRank as-is; 0 means "no rank" and opts into the domain's
		// unranked path. Assigning sequential i+1 positions as a fallback would
		// silently break tied finishers by giving them different ranks.
		participants = append(participants, leaderboardservice.RoundParticipantInput{
			MemberID:   string(r.UserID),
			FinishRank: r.FinishRank,
		})
	}

	output, err := h.service.ProcessRoundCommand(ctx, leaderboardservice.ProcessRoundCommand{
		GuildID:      string(payload.GuildID),
		RoundID:      uuid.UUID(*payload.RoundID),
		Participants: participants,
	})
	if err != nil {
		return nil, err
	}
	if output == nil {
		return nil, fmt.Errorf("process round returned nil output")
	}

	replyTo, _ := ctx.Value(handlerwrapper.CtxKeyReplyTo).(string)
	// Fetch the post-commit snapshot after ProcessRoundCommand has applied tag changes.
	// This event is informational; if the read fails, keep the committed round results
	// and downstream per-user updates instead of retrying the whole round.
	results := make([]handlerwrapper.Result, 0, 1)
	leaderboardUpdatedResult, err := h.buildLeaderboardUpdatedResult(ctx, payload.GuildID, *payload.RoundID)
	if err != nil {
		if h.logger != nil {
			h.logger.WarnContext(ctx, "failed to fetch leaderboard snapshot after round processing; continuing without leaderboard_updated event",
				"guild_id", payload.GuildID,
				"round_id", payload.RoundID.String(),
				"error", err,
			)
		}
	} else {
		results = append(results, leaderboardUpdatedResult)
	}

	memberIDs := make([]string, 0, len(output.FinalParticipantTags))
	for memberID := range output.FinalParticipantTags {
		memberIDs = append(memberIDs, memberID)
	}
	// Keep per-user tag events stable for tests and consumers that observe publish order.
	sort.Strings(memberIDs)

	roundRequests := make([]sharedtypes.TagAssignmentRequest, 0, len(output.FinalParticipantTags))
	for _, memberID := range memberIDs {
		tag := output.FinalParticipantTags[memberID]
		roundRequests = append(roundRequests, sharedtypes.TagAssignmentRequest{
			UserID:    sharedtypes.DiscordID(memberID),
			TagNumber: sharedtypes.TagNumber(tag),
		})
	}

	results = append(results, h.mapSuccessResults(ctx, payload.GuildID, payload.RequestingUserID, payload.BatchID, roundRequests, payload.Source, replyTo)...)

	if !output.PointsSkipped {
		pointsAwarded := make(map[sharedtypes.DiscordID]int, len(output.PointAwards))
		for _, award := range output.PointAwards {
			pointsAwarded[sharedtypes.DiscordID(award.MemberID)] = award.Points
		}

		pointsPayload := &sharedevents.PointsAwardedPayloadV1{
			GuildID: payload.GuildID,
			RoundID: *payload.RoundID,
			Points:  pointsAwarded,
		}

		if h.roundLookup != nil {
			round, err := h.roundLookup.GetRound(ctx, payload.GuildID, *payload.RoundID)
			if err != nil {
				h.logger.WarnContext(ctx, "failed to fetch round for points enrichment", "error", err)
			} else if round != nil {
				pointsPayload.EventMessageID = round.EventMessageID
				pointsPayload.Title = round.Title
				pointsPayload.Location = round.Location
				pointsPayload.StartTime = round.StartTime

				if round.Participants != nil {
					pointsPayload.Participants = make([]roundtypes.Participant, len(round.Participants))
					copy(pointsPayload.Participants, round.Participants)
					for i := range pointsPayload.Participants {
						if pts, ok := pointsPayload.Points[pointsPayload.Participants[i].UserID]; ok {
							p := pts
							pointsPayload.Participants[i].Points = &p
						}
					}
				}
				if round.Teams != nil {
					pointsPayload.Teams = make([]roundtypes.NormalizedTeam, len(round.Teams))
					for i, t := range round.Teams {
						pointsPayload.Teams[i] = t
						if t.Members != nil {
							pointsPayload.Teams[i].Members = make([]roundtypes.TeamMember, len(t.Members))
							copy(pointsPayload.Teams[i].Members, t.Members)
						}
						if t.HoleScores != nil {
							pointsPayload.Teams[i].HoleScores = make([]int, len(t.HoleScores))
							copy(pointsPayload.Teams[i].HoleScores, t.HoleScores)
						}
					}
				}
			}
		}

		pointsResult := handlerwrapper.Result{
			Topic:   sharedevents.PointsAwardedV1,
			Payload: pointsPayload,
		}
		if pointsPayload.EventMessageID != "" {
			pointsResult.Metadata = map[string]string{
				"discord_message_id": pointsPayload.EventMessageID,
			}
		}
		results = append(results, pointsResult)
	}

	results = h.addParallelIdentityResults(ctx, results, leaderboardevents.LeaderboardUpdatedV2, payload.GuildID)
	propagateCorrelationID(ctx, results)

	return results, nil
}

func (h *LeaderboardHandlers) buildLeaderboardUpdatedResult(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	roundID sharedtypes.RoundID,
) (handlerwrapper.Result, error) {
	fullLeaderboardResult, err := h.service.GetLeaderboard(ctx, guildID, "")
	if err != nil {
		return handlerwrapper.Result{}, fmt.Errorf("fetch full leaderboard after round processing: %w", err)
	}
	if fullLeaderboardResult.IsFailure() {
		if fullLeaderboardResult.Failure != nil {
			return handlerwrapper.Result{}, fmt.Errorf("fetch full leaderboard after round processing: %w", *fullLeaderboardResult.Failure)
		}
		return handlerwrapper.Result{}, fmt.Errorf("fetch full leaderboard after round processing: unknown failure")
	}
	if fullLeaderboardResult.Success == nil {
		return handlerwrapper.Result{}, fmt.Errorf("fetch full leaderboard after round processing: success data is nil")
	}

	return handlerwrapper.Result{
		Topic: leaderboardevents.LeaderboardUpdatedV2,
		Payload: &leaderboardevents.LeaderboardUpdatedPayloadV1{
			GuildID:         guildID,
			RoundID:         roundID,
			LeaderboardData: leaderboardDataMap(*fullLeaderboardResult.Success),
		},
	}, nil
}

func leaderboardDataMap(entries leaderboardtypes.LeaderboardData) map[sharedtypes.TagNumber]sharedtypes.DiscordID {
	data := make(map[sharedtypes.TagNumber]sharedtypes.DiscordID, len(entries))
	for _, entry := range entries {
		data[entry.TagNumber] = entry.UserID
	}
	return data
}

// propagateCorrelationID injects the correlation_id from context into the result metadata.
// It mutates the results slice's elements in-place.
func propagateCorrelationID(ctx context.Context, results []handlerwrapper.Result) {
	if val := ctx.Value("correlation_id"); val != nil {
		if correlationID, ok := val.(string); ok && correlationID != "" {
			// Sanitize
			if len(correlationID) > 64 {
				correlationID = correlationID[:64]
			}
			sanitizedID := ""
			for _, r := range correlationID {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
					sanitizedID += string(r)
				}
			}
			correlationID = sanitizedID

			for i := range results {
				if results[i].Metadata == nil {
					results[i].Metadata = make(map[string]string)
				}
				results[i].Metadata["correlation_id"] = correlationID
			}
		}
	}
}
