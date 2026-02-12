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
		// 0. Get Active Season
		season, err := s.repo.GetActiveSeason(ctx, db, string(guildID))
		if err != nil {
			return results.OperationResult[bool, error]{}, fmt.Errorf("failed to get active season: %w", err)
		}
		if season == nil {
			return results.FailureResult[bool, error](fmt.Errorf("no active season found")), nil
		}

		// 1. Save a PointHistory record with the adjustment reason
		history := &leaderboarddb.PointHistory{
			MemberID: memberID,
			RoundID:  sharedtypes.RoundID(uuid.Nil), // Zero UUID for manual adjustments
			SeasonID: season.ID,
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
				SeasonID: season.ID,
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

// ListSeasons returns all seasons for a guild, ordered by active first then start_date descending.
func (s *LeaderboardService) ListSeasons(
	ctx context.Context,
	guildID sharedtypes.GuildID,
) (results.OperationResult[[]SeasonInfo, error], error) {
	return withTelemetry(s, ctx, "ListSeasons", guildID, func(ctx context.Context) (results.OperationResult[[]SeasonInfo, error], error) {
		seasons, err := s.repo.ListSeasons(ctx, nil, string(guildID))
		if err != nil {
			return results.OperationResult[[]SeasonInfo, error]{}, fmt.Errorf("failed to list seasons: %w", err)
		}

		entries := make([]SeasonInfo, len(seasons))
		for i, season := range seasons {
			entry := SeasonInfo{
				ID:        season.ID,
				Name:      season.Name,
				IsActive:  season.IsActive,
				StartDate: season.StartDate.Format(time.RFC3339),
			}
			if !season.EndDate.IsZero() {
				endStr := season.EndDate.Format(time.RFC3339)
				entry.EndDate = &endStr
			}
			entries[i] = entry
		}
		return results.SuccessResult[[]SeasonInfo, error](entries), nil
	})
}

// GetSeasonName retrieves the display name for a specific season.
func (s *LeaderboardService) GetSeasonName(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	seasonID string,
) (string, error) {
	season, err := s.repo.GetSeasonByID(ctx, nil, string(guildID), seasonID)
	if err != nil {
		return "", fmt.Errorf("failed to get season: %w", err)
	}
	if season == nil {
		return "", nil
	}
	return season.Name, nil
}
