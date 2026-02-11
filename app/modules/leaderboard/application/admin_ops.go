package leaderboardservice

import (
	"context"
	"fmt"
	"time"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// GetPointHistoryForMember returns the point history for a given member.
func (s *LeaderboardService) GetPointHistoryForMember(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	memberID sharedtypes.DiscordID,
	limit int,
) (results.OperationResult[[]PointHistoryEntry, error], error) {
	return withTelemetry(s, ctx, "GetPointHistoryForMember", guildID, func(ctx context.Context) (results.OperationResult[[]PointHistoryEntry, error], error) {
		history, err := s.repo.GetPointHistoryForMember(ctx, nil, string(guildID), memberID, limit)
		if err != nil {
			return results.OperationResult[[]PointHistoryEntry, error]{}, fmt.Errorf("failed to get point history: %w", err)
		}

		entries := make([]PointHistoryEntry, len(history))
		for i, h := range history {
			entries[i] = PointHistoryEntry{
				MemberID:  h.MemberID,
				RoundID:   h.RoundID,
				SeasonID:  h.SeasonID,
				Points:    h.Points,
				Reason:    h.Reason,
				Tier:      h.Tier,
				Opponents: h.Opponents,
				CreatedAt: h.CreatedAt.Format(time.RFC3339),
			}
		}
		return results.SuccessResult[[]PointHistoryEntry, error](entries), nil
	})
}

// AdjustPoints manually adjusts a member's points with a reason.
func (s *LeaderboardService) AdjustPoints(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	memberID sharedtypes.DiscordID,
	pointsDelta int,
	reason string,
) (results.OperationResult[bool, error], error) {

	adjustTx := func(ctx context.Context, db bun.IDB) (results.OperationResult[bool, error], error) {
		// 1. Save a PointHistory record with the adjustment reason
		history := &leaderboarddb.PointHistory{
			MemberID: memberID,
			RoundID:  sharedtypes.RoundID(uuid.Nil), // Zero UUID for manual adjustments
			Points:   pointsDelta,
			Reason:   reason,
		}
		if err := s.repo.SavePointHistory(ctx, db, string(guildID), history); err != nil {
			return results.OperationResult[bool, error]{}, fmt.Errorf("failed to save adjustment history: %w", err)
		}

		// 2. Get current standing and update
		standing, err := s.repo.GetSeasonStanding(ctx, db, string(guildID), memberID)
		if err != nil {
			return results.OperationResult[bool, error]{}, fmt.Errorf("failed to get season standing: %w", err)
		}
		if standing == nil {
			standing = &leaderboarddb.SeasonStanding{
				MemberID: memberID,
			}
		}

		standing.TotalPoints += pointsDelta
		if standing.TotalPoints < 0 {
			standing.TotalPoints = 0
		}
		standing.UpdatedAt = time.Now()

		if err := s.repo.UpsertSeasonStanding(ctx, db, string(guildID), standing); err != nil {
			return results.OperationResult[bool, error]{}, fmt.Errorf("failed to update season standing: %w", err)
		}

		return results.SuccessResult[bool, error](true), nil
	}

	return withTelemetry(s, ctx, "AdjustPoints", guildID, func(ctx context.Context) (results.OperationResult[bool, error], error) {
		return runInTx(s, ctx, adjustTx)
	})
}

// RecalculateRound re-triggers ProcessRound for a given round.
// The caller must provide the round's participant data via the RoundLookup adapter.
// This method is a thin wrapper that delegates to ProcessRound, which handles idempotency via rollback.
func (s *LeaderboardService) RecalculateRound(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	roundID sharedtypes.RoundID,
) (results.OperationResult[ProcessRoundResult, error], error) {
	// RecalculateRound needs participant data to call ProcessRound.
	// Since the service layer doesn't have direct access to RoundLookup,
	// the handler layer must fetch the participant data and call ProcessRound directly.
	// This method exists as a placeholder for the service interface.
	// The handler will call ProcessRound with the fetched player results.
	err := fmt.Errorf("RecalculateRound must be called through the handler layer which provides participant data")
	return results.OperationResult[ProcessRoundResult, error]{}, err
}

// StartNewSeason creates a new season, deactivating the old one. Existing data is preserved under the old season_id.
func (s *LeaderboardService) StartNewSeason(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	seasonID string,
	seasonName string,
) (results.OperationResult[bool, error], error) {
	return withTelemetry(s, ctx, "StartNewSeason", guildID, func(ctx context.Context) (results.OperationResult[bool, error], error) {
		if s.commandPipeline == nil {
			return results.OperationResult[bool, error]{}, ErrCommandPipelineUnavailable
		}
		if err := s.commandPipeline.StartSeason(ctx, string(guildID), seasonID, seasonName); err != nil {
			return results.OperationResult[bool, error]{}, err
		}
		return results.SuccessResult[bool, error](true), nil
	})
}

// GetSeasonStandingsForSeason retrieves standings for a specific season.
func (s *LeaderboardService) GetSeasonStandingsForSeason(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	seasonID string,
) (results.OperationResult[[]SeasonStandingEntry, error], error) {
	return withTelemetry(s, ctx, "GetSeasonStandings", guildID, func(ctx context.Context) (results.OperationResult[[]SeasonStandingEntry, error], error) {
		standings, err := s.repo.GetSeasonStandingsBySeasonID(ctx, nil, string(guildID), seasonID)
		if err != nil {
			return results.OperationResult[[]SeasonStandingEntry, error]{}, fmt.Errorf("failed to get season standings: %w", err)
		}

		entries := make([]SeasonStandingEntry, len(standings))
		for i, s := range standings {
			entries[i] = SeasonStandingEntry{
				MemberID:      s.MemberID,
				SeasonID:      s.SeasonID,
				TotalPoints:   s.TotalPoints,
				CurrentTier:   s.CurrentTier,
				SeasonBestTag: s.SeasonBestTag,
				RoundsPlayed:  s.RoundsPlayed,
			}
		}
		return results.SuccessResult[[]SeasonStandingEntry, error](entries), nil
	})
}
