package leaderboardhandlers

import (
	"context"
	"fmt"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
)

// HandlePointHistoryRequested returns point history for a member.
func (h *LeaderboardHandlers) HandlePointHistoryRequested(
	ctx context.Context,
	payload *PointHistoryRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	limit := payload.Limit
	if limit <= 0 {
		limit = 50
	}

	result, err := h.service.GetPointHistoryForMember(ctx, payload.GuildID, payload.MemberID, limit)
	if err != nil {
		return []handlerwrapper.Result{{
			Topic:   LeaderboardPointHistoryFailedV1,
			Payload: &FailedPayloadV1{GuildID: payload.GuildID, Reason: err.Error()},
		}}, nil
	}

	if result.IsFailure() {
		return []handlerwrapper.Result{{
			Topic:   LeaderboardPointHistoryFailedV1,
			Payload: &FailedPayloadV1{GuildID: payload.GuildID, Reason: fmt.Sprintf("%v", *result.Failure)},
		}}, nil
	}

	items := make([]PointHistoryItemV1, len(*result.Success))
	for i, entry := range *result.Success {
		items[i] = PointHistoryItemV1{
			RoundID:   entry.RoundID,
			SeasonID:  entry.SeasonID,
			Points:    entry.Points,
			Reason:    entry.Reason,
			Tier:      entry.Tier,
			Opponents: entry.Opponents,
			CreatedAt: entry.CreatedAt,
		}
	}

	return []handlerwrapper.Result{{
		Topic: LeaderboardPointHistoryResponseV1,
		Payload: &PointHistoryResponsePayloadV1{
			GuildID:  payload.GuildID,
			MemberID: payload.MemberID,
			History:  items,
		},
	}}, nil
}

// HandleManualPointAdjustment processes a manual point adjustment request.
func (h *LeaderboardHandlers) HandleManualPointAdjustment(
	ctx context.Context,
	payload *ManualPointAdjustmentPayloadV1,
) ([]handlerwrapper.Result, error) {
	reason := payload.Reason
	if payload.AdminID != "" {
		reason = fmt.Sprintf("Admin adjustment by %s: %s", payload.AdminID, payload.Reason)
	}

	result, err := h.service.AdjustPoints(ctx, payload.GuildID, payload.MemberID, payload.PointsDelta, reason)
	if err != nil {
		return []handlerwrapper.Result{{
			Topic:   LeaderboardManualPointAdjustmentFailedV1,
			Payload: &FailedPayloadV1{GuildID: payload.GuildID, Reason: err.Error()},
		}}, nil
	}

	if result.IsFailure() {
		return []handlerwrapper.Result{{
			Topic:   LeaderboardManualPointAdjustmentFailedV1,
			Payload: &FailedPayloadV1{GuildID: payload.GuildID, Reason: fmt.Sprintf("%v", *result.Failure)},
		}}, nil
	}

	return []handlerwrapper.Result{{
		Topic: LeaderboardManualPointAdjustmentSuccessV1,
		Payload: &ManualPointAdjustmentSuccessPayloadV1{
			GuildID:     payload.GuildID,
			MemberID:    payload.MemberID,
			PointsDelta: payload.PointsDelta,
			Reason:      payload.Reason,
		},
	}}, nil
}

// HandleRecalculateRound triggers recalculation of a round's points.
func (h *LeaderboardHandlers) HandleRecalculateRound(
	ctx context.Context,
	payload *RecalculateRoundPayloadV1,
) ([]handlerwrapper.Result, error) {
	// Fetch round data to get participant info
	if h.roundLookup == nil {
		return []handlerwrapper.Result{{
			Topic:   LeaderboardRecalculateRoundFailedV1,
			Payload: &FailedPayloadV1{GuildID: payload.GuildID, Reason: "round lookup not available"},
		}}, nil
	}

	round, err := h.roundLookup.GetRound(ctx, payload.GuildID, payload.RoundID)
	if err != nil {
		return []handlerwrapper.Result{{
			Topic:   LeaderboardRecalculateRoundFailedV1,
			Payload: &FailedPayloadV1{GuildID: payload.GuildID, Reason: fmt.Sprintf("failed to fetch round: %v", err)},
		}}, nil
	}
	if round == nil || len(round.Participants) == 0 {
		return []handlerwrapper.Result{{
			Topic:   LeaderboardRecalculateRoundFailedV1,
			Payload: &FailedPayloadV1{GuildID: payload.GuildID, Reason: "round not found or has no participants"},
		}}, nil
	}

	// Build player results from round participants
	playerResults := make([]leaderboardservice.PlayerResult, 0, len(round.Participants))
	for _, p := range round.Participants {
		if p.TagNumber != nil {
			playerResults = append(playerResults, leaderboardservice.PlayerResult{
				PlayerID:  p.UserID,
				TagNumber: int(*p.TagNumber),
			})
		}
	}

	if len(playerResults) == 0 {
		return []handlerwrapper.Result{{
			Topic:   LeaderboardRecalculateRoundFailedV1,
			Payload: &FailedPayloadV1{GuildID: payload.GuildID, Reason: "no participants with tags found"},
		}}, nil
	}

	// Call ProcessRound which handles idempotency via rollback
	result, err := h.service.ProcessRound(
		ctx,
		payload.GuildID,
		payload.RoundID,
		playerResults,
		sharedtypes.ServiceUpdateSourceProcessScores,
	)
	if err != nil {
		return []handlerwrapper.Result{{
			Topic:   LeaderboardRecalculateRoundFailedV1,
			Payload: &FailedPayloadV1{GuildID: payload.GuildID, Reason: err.Error()},
		}}, nil
	}

	if result.IsFailure() {
		return []handlerwrapper.Result{{
			Topic:   LeaderboardRecalculateRoundFailedV1,
			Payload: &FailedPayloadV1{GuildID: payload.GuildID, Reason: fmt.Sprintf("%v", *result.Failure)},
		}}, nil
	}

	return []handlerwrapper.Result{{
		Topic: LeaderboardRecalculateRoundSuccessV1,
		Payload: &RecalculateRoundSuccessPayloadV1{
			GuildID:       payload.GuildID,
			RoundID:       payload.RoundID,
			PointsAwarded: result.Success.PointsAwarded,
		},
	}}, nil
}

// HandleStartNewSeason creates a new season.
func (h *LeaderboardHandlers) HandleStartNewSeason(
	ctx context.Context,
	payload *StartNewSeasonPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.StartNewSeason(ctx, payload.GuildID, payload.SeasonID, payload.SeasonName)
	if err != nil {
		return []handlerwrapper.Result{{
			Topic:   LeaderboardStartNewSeasonFailedV1,
			Payload: &FailedPayloadV1{GuildID: payload.GuildID, Reason: err.Error()},
		}}, nil
	}

	if result.IsFailure() {
		return []handlerwrapper.Result{{
			Topic:   LeaderboardStartNewSeasonFailedV1,
			Payload: &FailedPayloadV1{GuildID: payload.GuildID, Reason: fmt.Sprintf("%v", *result.Failure)},
		}}, nil
	}

	return []handlerwrapper.Result{{
		Topic: LeaderboardStartNewSeasonSuccessV1,
		Payload: &StartNewSeasonSuccessPayloadV1{
			GuildID:    payload.GuildID,
			SeasonID:   payload.SeasonID,
			SeasonName: payload.SeasonName,
		},
	}}, nil
}

// HandleGetSeasonStandings returns standings for a specific season.
func (h *LeaderboardHandlers) HandleGetSeasonStandings(
	ctx context.Context,
	payload *GetSeasonStandingsPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.GetSeasonStandingsForSeason(ctx, payload.GuildID, payload.SeasonID)
	if err != nil {
		return []handlerwrapper.Result{{
			Topic:   LeaderboardGetSeasonStandingsFailedV1,
			Payload: &FailedPayloadV1{GuildID: payload.GuildID, Reason: err.Error()},
		}}, nil
	}

	if result.IsFailure() {
		return []handlerwrapper.Result{{
			Topic:   LeaderboardGetSeasonStandingsFailedV1,
			Payload: &FailedPayloadV1{GuildID: payload.GuildID, Reason: fmt.Sprintf("%v", *result.Failure)},
		}}, nil
	}

	items := make([]SeasonStandingItemV1, len(*result.Success))
	for i, entry := range *result.Success {
		items[i] = SeasonStandingItemV1{
			MemberID:      entry.MemberID,
			TotalPoints:   entry.TotalPoints,
			CurrentTier:   entry.CurrentTier,
			SeasonBestTag: entry.SeasonBestTag,
			RoundsPlayed:  entry.RoundsPlayed,
		}
	}

	return []handlerwrapper.Result{{
		Topic: LeaderboardGetSeasonStandingsResponseV1,
		Payload: &GetSeasonStandingsResponsePayloadV1{
			GuildID:   payload.GuildID,
			SeasonID:  payload.SeasonID,
			Standings: items,
		},
	}}, nil
}
