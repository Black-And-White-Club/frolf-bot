package leaderboardhandlers

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
)

// HandlePointHistoryRequested returns point history for a member.
func (h *LeaderboardHandlers) HandlePointHistoryRequested(
	ctx context.Context,
	payload *leaderboardevents.PointHistoryRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	limit := payload.Limit
	if limit <= 0 {
		limit = 50
	}

	result, err := h.service.GetPointHistoryForMember(ctx, payload.GuildID, payload.MemberID, limit)
	if err != nil {
		return []handlerwrapper.Result{{
			Topic:   leaderboardevents.LeaderboardPointHistoryFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: err.Error()},
		}}, nil
	}

	if result.IsFailure() {
		return []handlerwrapper.Result{{
			Topic:   leaderboardevents.LeaderboardPointHistoryFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: fmt.Sprintf("%v", *result.Failure)},
		}}, nil
	}

	items := make([]leaderboardevents.PointHistoryItemV1, len(*result.Success))
	for i, entry := range *result.Success {
		items[i] = leaderboardevents.PointHistoryItemV1{
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
		Topic: leaderboardevents.LeaderboardPointHistoryResponseV1,
		Payload: &leaderboardevents.PointHistoryResponsePayloadV1{
			GuildID:  payload.GuildID,
			MemberID: payload.MemberID,
			History:  items,
		},
	}}, nil
}

// HandleManualPointAdjustment processes a manual point adjustment request.
func (h *LeaderboardHandlers) HandleManualPointAdjustment(
	ctx context.Context,
	payload *leaderboardevents.ManualPointAdjustmentPayloadV1,
) ([]handlerwrapper.Result, error) {
	reason := payload.Reason
	if payload.AdminID != "" {
		reason = fmt.Sprintf("Admin adjustment by %s: %s", payload.AdminID, payload.Reason)
	}

	result, err := h.service.AdjustPoints(ctx, payload.GuildID, payload.MemberID, payload.PointsDelta, reason)
	if err != nil {
		return []handlerwrapper.Result{{
			Topic:   leaderboardevents.LeaderboardManualPointAdjustmentFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: err.Error()},
		}}, nil
	}

	if result.IsFailure() {
		return []handlerwrapper.Result{{
			Topic:   leaderboardevents.LeaderboardManualPointAdjustmentFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: fmt.Sprintf("%v", *result.Failure)},
		}}, nil
	}

	return []handlerwrapper.Result{{
		Topic: leaderboardevents.LeaderboardManualPointAdjustmentSuccessV1,
		Payload: &leaderboardevents.ManualPointAdjustmentSuccessPayloadV1{
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
	payload *leaderboardevents.RecalculateRoundPayloadV1,
) ([]handlerwrapper.Result, error) {
	// Fetch round data to get participant info
	if h.roundLookup == nil {
		return []handlerwrapper.Result{{
			Topic:   leaderboardevents.LeaderboardRecalculateRoundFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: "round lookup not available"},
		}}, nil
	}

	round, err := h.roundLookup.GetRound(ctx, payload.GuildID, payload.RoundID)
	if err != nil {
		return []handlerwrapper.Result{{
			Topic:   leaderboardevents.LeaderboardRecalculateRoundFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: fmt.Sprintf("failed to fetch round: %v", err)},
		}}, nil
	}
	if round == nil || len(round.Participants) == 0 {
		return []handlerwrapper.Result{{
			Topic:   leaderboardevents.LeaderboardRecalculateRoundFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: "round not found or has no participants"},
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
			Topic:   leaderboardevents.LeaderboardRecalculateRoundFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: "no participants with tags found"},
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
			Topic:   leaderboardevents.LeaderboardRecalculateRoundFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: err.Error()},
		}}, nil
	}

	if result.IsFailure() {
		return []handlerwrapper.Result{{
			Topic:   leaderboardevents.LeaderboardRecalculateRoundFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: fmt.Sprintf("%v", *result.Failure)},
		}}, nil
	}

	return []handlerwrapper.Result{{
		Topic: leaderboardevents.LeaderboardRecalculateRoundSuccessV1,
		Payload: &leaderboardevents.RecalculateRoundSuccessPayloadV1{
			GuildID:       payload.GuildID,
			RoundID:       payload.RoundID,
			PointsAwarded: result.Success.PointsAwarded,
		},
	}}, nil
}

// HandleStartNewSeason creates a new season.
func (h *LeaderboardHandlers) HandleStartNewSeason(
	ctx context.Context,
	payload *leaderboardevents.StartNewSeasonPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.StartNewSeason(ctx, payload.GuildID, payload.SeasonID, payload.SeasonName)
	if err != nil {
		return []handlerwrapper.Result{{
			Topic:   leaderboardevents.LeaderboardStartNewSeasonFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: err.Error()},
		}}, nil
	}

	if result.IsFailure() {
		return []handlerwrapper.Result{{
			Topic:   leaderboardevents.LeaderboardStartNewSeasonFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: fmt.Sprintf("%v", *result.Failure)},
		}}, nil
	}

	return []handlerwrapper.Result{{
		Topic: leaderboardevents.LeaderboardStartNewSeasonSuccessV1,
		Payload: &leaderboardevents.StartNewSeasonSuccessPayloadV1{
			GuildID:    payload.GuildID,
			SeasonID:   payload.SeasonID,
			SeasonName: payload.SeasonName,
		},
	}}, nil
}

// HandleGetSeasonStandings returns standings for a specific season.
func (h *LeaderboardHandlers) HandleGetSeasonStandings(
	ctx context.Context,
	payload *leaderboardevents.GetSeasonStandingsPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.GetSeasonStandingsForSeason(ctx, payload.GuildID, payload.SeasonID)
	if err != nil {
		return []handlerwrapper.Result{{
			Topic:   leaderboardevents.LeaderboardGetSeasonStandingsFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: err.Error()},
		}}, nil
	}

	if result.IsFailure() {
		return []handlerwrapper.Result{{
			Topic:   leaderboardevents.LeaderboardGetSeasonStandingsFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: fmt.Sprintf("%v", *result.Failure)},
		}}, nil
	}

	items := make([]leaderboardevents.SeasonStandingItemV1, len(*result.Success))
	for i, entry := range *result.Success {
		items[i] = leaderboardevents.SeasonStandingItemV1{
			MemberID:      entry.MemberID,
			TotalPoints:   entry.TotalPoints,
			CurrentTier:   entry.CurrentTier,
			SeasonBestTag: entry.SeasonBestTag,
			RoundsPlayed:  entry.RoundsPlayed,
		}
	}

	return []handlerwrapper.Result{{
		Topic: leaderboardevents.LeaderboardGetSeasonStandingsResponseV1,
		Payload: &leaderboardevents.GetSeasonStandingsResponsePayloadV1{
			GuildID:   payload.GuildID,
			SeasonID:  payload.SeasonID,
			Standings: items,
		},
	}}, nil
}

// HandleListSeasonsRequest returns all seasons for a guild via NATS request-reply.
func (h *LeaderboardHandlers) HandleListSeasonsRequest(
	ctx context.Context,
	payload *leaderboardevents.ListSeasonsRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.ListSeasons(ctx, payload.GuildID)
	if err != nil {
		return []handlerwrapper.Result{{
			Topic:   leaderboardevents.LeaderboardListSeasonsFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: err.Error()},
		}}, nil
	}

	if result.IsFailure() {
		return []handlerwrapper.Result{{
			Topic:   leaderboardevents.LeaderboardListSeasonsFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: fmt.Sprintf("%v", *result.Failure)},
		}}, nil
	}

	seasons := make([]leaderboardevents.SeasonInfoV1, len(*result.Success))
	for i, s := range *result.Success {
		seasons[i] = leaderboardevents.SeasonInfoV1{
			ID:        s.ID,
			Name:      s.Name,
			IsActive:  s.IsActive,
			StartDate: s.StartDate,
			EndDate:   s.EndDate,
		}
	}

	resp := &leaderboardevents.ListSeasonsResponsePayloadV1{
		GuildID: payload.GuildID,
		Seasons: seasons,
	}

	// Use reply_to for request-reply pattern
	topic := leaderboardevents.LeaderboardListSeasonsResponseV1
	if replyTo, ok := ctx.Value(handlerwrapper.CtxKeyReplyTo).(string); ok && replyTo != "" {
		topic = replyTo
	}

	return []handlerwrapper.Result{{Topic: topic, Payload: resp}}, nil
}

// HandleSeasonStandingsRequest returns standings for a season via NATS request-reply.
func (h *LeaderboardHandlers) HandleSeasonStandingsRequest(
	ctx context.Context,
	payload *leaderboardevents.SeasonStandingsRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.GetSeasonStandingsForSeason(ctx, payload.GuildID, payload.SeasonID)
	if err != nil {
		return []handlerwrapper.Result{{
			Topic:   leaderboardevents.LeaderboardSeasonStandingsFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: err.Error()},
		}}, nil
	}

	if result.IsFailure() {
		return []handlerwrapper.Result{{
			Topic:   leaderboardevents.LeaderboardSeasonStandingsFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: fmt.Sprintf("%v", *result.Failure)},
		}}, nil
	}

	// Get season name for display
	seasonName, _ := h.service.GetSeasonName(ctx, payload.GuildID, payload.SeasonID)

	items := make([]leaderboardevents.SeasonStandingItemV1, len(*result.Success))
	for i, entry := range *result.Success {
		items[i] = leaderboardevents.SeasonStandingItemV1{
			MemberID:      entry.MemberID,
			TotalPoints:   entry.TotalPoints,
			CurrentTier:   entry.CurrentTier,
			SeasonBestTag: entry.SeasonBestTag,
			RoundsPlayed:  entry.RoundsPlayed,
		}
	}

	resp := &leaderboardevents.SeasonStandingsResponsePayloadV1{
		GuildID:    payload.GuildID,
		SeasonID:   payload.SeasonID,
		SeasonName: seasonName,
		Standings:  items,
	}

	// Use reply_to for request-reply pattern
	topic := leaderboardevents.LeaderboardSeasonStandingsResponseV1
	if replyTo, ok := ctx.Value(handlerwrapper.CtxKeyReplyTo).(string); ok && replyTo != "" {
		topic = replyTo
	}

	return []handlerwrapper.Result{{Topic: topic, Payload: resp}}, nil
}

// HandleEndSeason ends the active season.
func (h *LeaderboardHandlers) HandleEndSeason(
	ctx context.Context,
	payload *leaderboardevents.EndSeasonPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.EndSeason(ctx, payload.GuildID)
	if err != nil {
		return []handlerwrapper.Result{{
			Topic:   leaderboardevents.LeaderboardEndSeasonFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: err.Error()},
		}}, nil
	}

	if result.IsFailure() {
		return []handlerwrapper.Result{{
			Topic:   leaderboardevents.LeaderboardEndSeasonFailedV1,
			Payload: &leaderboardevents.AdminFailedPayloadV1{GuildID: payload.GuildID, Reason: fmt.Sprintf("%v", *result.Failure)},
		}}, nil
	}

	return []handlerwrapper.Result{{
		Topic: leaderboardevents.LeaderboardEndSeasonSuccessV1,
		Payload: &leaderboardevents.EndSeasonSuccessPayloadV1{
			GuildID: payload.GuildID,
		},
	}}, nil
}
