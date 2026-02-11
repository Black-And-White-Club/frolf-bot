package leaderboarddb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

const defaultSeasonID = "default"

// getActiveSeasonID returns the active season ID, falling back to "default".
func (r *Impl) getActiveSeasonID(ctx context.Context, db bun.IDB) string {
	if db == nil {
		db = r.db
	}
	season, err := r.GetActiveSeason(ctx, db)
	if err != nil || season == nil {
		return defaultSeasonID
	}
	return season.ID
}

// SavePointHistory records points earned by a member.
func (r *Impl) SavePointHistory(ctx context.Context, db bun.IDB, history *PointHistory) error {
	if db == nil {
		db = r.db
	}
	if history.SeasonID == "" {
		history.SeasonID = r.getActiveSeasonID(ctx, db)
	}
	_, err := db.NewInsert().Model(history).Exec(ctx)
	if err != nil {
		return fmt.Errorf("leaderboarddb.SavePointHistory: %w", err)
	}
	return nil
}

// BulkSavePointHistory records multiple point history records efficiently.
func (r *Impl) BulkSavePointHistory(ctx context.Context, db bun.IDB, histories []*PointHistory) error {
	if len(histories) == 0 {
		return nil
	}
	if db == nil {
		db = r.db
	}
	// Pre-fetch active season if any record is missing it
	var activeSeasonID string
	for _, h := range histories {
		if h.SeasonID == "" {
			if activeSeasonID == "" {
				activeSeasonID = r.getActiveSeasonID(ctx, db)
			}
			h.SeasonID = activeSeasonID
		}
	}
	_, err := db.NewInsert().Model(&histories).Exec(ctx)
	if err != nil {
		return fmt.Errorf("leaderboarddb.BulkSavePointHistory: %w", err)
	}
	return nil
}

// GetPointHistoryForRound retrieves all point history records for a specific round.
func (r *Impl) GetPointHistoryForRound(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) ([]PointHistory, error) {
	if db == nil {
		db = r.db
	}
	var history []PointHistory
	err := db.NewSelect().
		Model(&history).
		Where("round_id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("leaderboarddb.GetPointHistoryForRound: %w", err)
	}
	return history, nil
}

// DeletePointHistoryForRound deletes all point history records for a specific round.
func (r *Impl) DeletePointHistoryForRound(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) error {
	if db == nil {
		db = r.db
	}
	_, err := db.NewDelete().
		Model((*PointHistory)(nil)).
		Where("round_id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("leaderboarddb.DeletePointHistoryForRound: %w", err)
	}
	return nil
}

// UpsertSeasonStanding updates or creates a season standing record.
func (r *Impl) UpsertSeasonStanding(ctx context.Context, db bun.IDB, standing *SeasonStanding) error {
	if db == nil {
		db = r.db
	}
	if standing.SeasonID == "" {
		standing.SeasonID = r.getActiveSeasonID(ctx, db)
	}
	_, err := db.NewInsert().
		Model(standing).
		On("CONFLICT (season_id, member_id) DO UPDATE").
		Set("total_points = EXCLUDED.total_points").
		Set("current_tier = EXCLUDED.current_tier").
		Set("season_best_tag = EXCLUDED.season_best_tag").
		Set("rounds_played = EXCLUDED.rounds_played").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("leaderboarddb.UpsertSeasonStanding: %w", err)
	}
	return nil
}

// BulkUpsertSeasonStandings updates or creates multiple season standing records efficiently.
func (r *Impl) BulkUpsertSeasonStandings(ctx context.Context, db bun.IDB, standings []*SeasonStanding) error {
	if len(standings) == 0 {
		return nil
	}
	if db == nil {
		db = r.db
	}
	var activeSeasonID string
	for _, s := range standings {
		if s.SeasonID == "" {
			if activeSeasonID == "" {
				activeSeasonID = r.getActiveSeasonID(ctx, db)
			}
			s.SeasonID = activeSeasonID
		}
	}
	_, err := db.NewInsert().
		Model(&standings).
		On("CONFLICT (season_id, member_id) DO UPDATE").
		Set("total_points = EXCLUDED.total_points").
		Set("current_tier = EXCLUDED.current_tier").
		Set("season_best_tag = EXCLUDED.season_best_tag").
		Set("rounds_played = EXCLUDED.rounds_played").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("leaderboarddb.BulkUpsertSeasonStandings: %w", err)
	}
	return nil
}

// DecrementSeasonStanding decrements a member's season standing points and rounds played.
// If seasonID is empty, the active season is used.
func (r *Impl) DecrementSeasonStanding(ctx context.Context, db bun.IDB, memberID sharedtypes.DiscordID, seasonID string, pointsToRemove int) error {
	if db == nil {
		db = r.db
	}
	if seasonID == "" {
		seasonID = r.getActiveSeasonID(ctx, db)
	}
	_, err := db.NewUpdate().
		Model((*SeasonStanding)(nil)).
		Set("total_points = GREATEST(total_points - ?, 0)", pointsToRemove).
		Set("rounds_played = GREATEST(rounds_played - 1, 0)").
		Where("member_id = ?", memberID).
		Where("season_id = ?", seasonID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("leaderboarddb.DecrementSeasonStanding: %w", err)
	}
	return nil
}

// GetSeasonStanding retrieves a member's season standing.
func (r *Impl) GetSeasonStanding(ctx context.Context, db bun.IDB, memberID sharedtypes.DiscordID) (*SeasonStanding, error) {
	if db == nil {
		db = r.db
	}
	seasonID := r.getActiveSeasonID(ctx, db)
	standing := new(SeasonStanding)
	err := db.NewSelect().
		Model(standing).
		Where("member_id = ?", memberID).
		Where("season_id = ?", seasonID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("leaderboarddb.GetSeasonStanding: %w", err)
	}
	return standing, nil
}

// GetSeasonBestTags retrieves the best tag for a list of members for a season.
// If seasonID is empty, the active season is used.
func (r *Impl) GetSeasonBestTags(ctx context.Context, db bun.IDB, seasonID string, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]int, error) {
	if db == nil {
		db = r.db
	}

	if len(memberIDs) == 0 {
		return make(map[sharedtypes.DiscordID]int), nil
	}

	if seasonID == "" {
		seasonID = r.getActiveSeasonID(ctx, db)
	}

	var results []struct {
		MemberID sharedtypes.DiscordID
		BestTag  int
	}

	err := db.NewSelect().
		Model((*SeasonStanding)(nil)).
		ColumnExpr("member_id, season_best_tag as best_tag").
		Where("member_id IN (?)", bun.In(memberIDs)).
		Where("season_id = ?", seasonID).
		Scan(ctx, &results)

	if err != nil {
		return nil, fmt.Errorf("leaderboarddb.GetSeasonBestTags: %w", err)
	}

	bestTags := make(map[sharedtypes.DiscordID]int)
	for _, res := range results {
		bestTags[res.MemberID] = res.BestTag
	}
	return bestTags, nil
}

// GetSeasonStandings retrieves season standings for a batch of members in a season.
// If seasonID is empty, the active season is used.
func (r *Impl) GetSeasonStandings(ctx context.Context, db bun.IDB, seasonID string, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]*SeasonStanding, error) {
	if db == nil {
		db = r.db
	}

	if len(memberIDs) == 0 {
		return make(map[sharedtypes.DiscordID]*SeasonStanding), nil
	}

	if seasonID == "" {
		seasonID = r.getActiveSeasonID(ctx, db)
	}

	var standings []SeasonStanding
	err := db.NewSelect().
		Model(&standings).
		Where("member_id IN (?)", bun.In(memberIDs)).
		Where("season_id = ?", seasonID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("leaderboarddb.GetSeasonStandings: %w", err)
	}

	result := make(map[sharedtypes.DiscordID]*SeasonStanding, len(standings))
	for i := range standings {
		result[standings[i].MemberID] = &standings[i]
	}
	return result, nil
}

// CountSeasonMembers returns the total number of members with a standing in a season.
// If seasonID is empty, the active season is used.
func (r *Impl) CountSeasonMembers(ctx context.Context, db bun.IDB, seasonID string) (int, error) {
	if db == nil {
		db = r.db
	}
	if seasonID == "" {
		seasonID = r.getActiveSeasonID(ctx, db)
	}
	count, err := db.NewSelect().
		Model((*SeasonStanding)(nil)).
		Where("season_id = ?", seasonID).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("leaderboarddb.CountSeasonMembers: %w", err)
	}
	return count, nil
}

// --- Season Management ---

// GetActiveSeason retrieves the currently active season.
func (r *Impl) GetActiveSeason(ctx context.Context, db bun.IDB) (*Season, error) {
	if db == nil {
		db = r.db
	}
	season := new(Season)
	err := db.NewSelect().
		Model(season).
		Where("is_active = true").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("leaderboarddb.GetActiveSeason: %w", err)
	}
	return season, nil
}

// CreateSeason creates a new season record.
func (r *Impl) CreateSeason(ctx context.Context, db bun.IDB, season *Season) error {
	if db == nil {
		db = r.db
	}
	_, err := db.NewInsert().Model(season).Exec(ctx)
	if err != nil {
		return fmt.Errorf("leaderboarddb.CreateSeason: %w", err)
	}
	return nil
}

// DeactivateAllSeasons sets is_active=false for all seasons.
func (r *Impl) DeactivateAllSeasons(ctx context.Context, db bun.IDB) error {
	if db == nil {
		db = r.db
	}
	_, err := db.NewUpdate().
		Model((*Season)(nil)).
		Set("is_active = false").
		Where("is_active = true").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("leaderboarddb.DeactivateAllSeasons: %w", err)
	}
	return nil
}

// GetPointHistoryForMember retrieves point history for a member, ordered by created_at desc.
func (r *Impl) GetPointHistoryForMember(ctx context.Context, db bun.IDB, memberID sharedtypes.DiscordID, limit int) ([]PointHistory, error) {
	if db == nil {
		db = r.db
	}
	var history []PointHistory
	q := db.NewSelect().
		Model(&history).
		Where("member_id = ?", memberID).
		Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("leaderboarddb.GetPointHistoryForMember: %w", err)
	}
	return history, nil
}

// GetSeasonStandingsBySeasonID retrieves all standings for a specific season.
func (r *Impl) GetSeasonStandingsBySeasonID(ctx context.Context, db bun.IDB, seasonID string) ([]SeasonStanding, error) {
	if db == nil {
		db = r.db
	}
	var standings []SeasonStanding
	err := db.NewSelect().
		Model(&standings).
		Where("season_id = ?", seasonID).
		Order("total_points DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("leaderboarddb.GetSeasonStandingsBySeasonID: %w", err)
	}
	return standings, nil
}
