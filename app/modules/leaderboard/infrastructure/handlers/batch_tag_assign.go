package leaderboardhandlers

import (
	"context"
	"errors"
	"fmt"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
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

	results := h.mapSuccessResults(payload.GuildID, payload.RequestingUserID, payload.BatchID, *result.Success, "batch_assignment")

	propagateCorrelationID(ctx, results)

	return results, nil
}

func (h *LeaderboardHandlers) handleRoundBasedAssignment(
	ctx context.Context,
	payload *sharedevents.BatchTagAssignmentRequestedPayloadV1,
	requests []sharedtypes.TagAssignmentRequest,
) ([]handlerwrapper.Result, error) {
	// Round-based update: Includes point calculations
	playerResults := make([]leaderboardservice.PlayerResult, len(requests))

	for i, r := range requests {
		playerResults[i] = leaderboardservice.PlayerResult{
			PlayerID:  r.UserID,
			TagNumber: int(r.TagNumber),
		}
	}

	result, err := h.service.ProcessRound(
		ctx,
		payload.GuildID,
		*payload.RoundID,
		playerResults,
		payload.Source,
	)
	if err != nil {
		return nil, err
	}

	// ProcessRound returns OperationResult[ProcessRoundResult, error]
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

	// Success logic for ProcessRound
	results := h.mapSuccessResults(payload.GuildID, payload.RequestingUserID, payload.BatchID, result.Success.LeaderboardData, "batch_assignment")

	// Emit points awarded event
	pointsPayload := &sharedevents.PointsAwardedPayloadV1{
		GuildID: payload.GuildID,
		RoundID: *payload.RoundID,
		Points:  result.Success.PointsAwarded,
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

			// Deep copy participants to avoid mutating the source round object
			if round.Participants != nil {
				pointsPayload.Participants = make([]roundtypes.Participant, len(round.Participants))
				copy(pointsPayload.Participants, round.Participants)

				// Merge points into participants for pre-populated display
				for i := range pointsPayload.Participants {
					if pts, ok := pointsPayload.Points[pointsPayload.Participants[i].UserID]; ok {
						p := pts
						pointsPayload.Participants[i].Points = &p
					}
				}
			}
			// Deep copy Teams to avoid mutating source round
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

	results = append(results, handlerwrapper.Result{
		Topic:   sharedevents.PointsAwardedV1,
		Payload: pointsPayload,
	})

	// Add metadata if available
	if pointsPayload.EventMessageID != "" {
		results[len(results)-1].Metadata = map[string]string{
			"discord_message_id": pointsPayload.EventMessageID,
		}
	}

	propagateCorrelationID(ctx, results)

	return results, nil
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
