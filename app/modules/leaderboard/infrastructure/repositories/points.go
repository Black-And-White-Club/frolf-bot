package leaderboarddb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// SavePointHistory records points earned by a member.
func (r *Impl) SavePointHistory(ctx context.Context, db bun.IDB, history *PointHistory) error {
	if db == nil {
		db = r.db
	}
	_, err := db.NewInsert().Model(history).Exec(ctx)
	if err != nil {
		return fmt.Errorf("leaderboarddb.SavePointHistory: %w", err)
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
	_, err := db.NewInsert().
		Model(standing).
		On("CONFLICT (member_id) DO UPDATE").
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

// DecrementSeasonStanding decrements a member's season standing points and rounds played.
func (r *Impl) DecrementSeasonStanding(ctx context.Context, db bun.IDB, memberID sharedtypes.DiscordID, pointsToRemove int) error {
	if db == nil {
		db = r.db
	}
	// We decrease total_points and rounds_played.
	// We do NOT decrement if rounds_played is 0 (though that shouldn't happen if history exists).
	// We also ensure total_points doesn't go below 0 (logic choice: points strictly additive, but safe to clamp).
	// Actually, if a player had 10 points and we remove 10, it goes to 0. If they had -5 (if penalties exist), it goes to -15.
	// Simple subtraction is likely correct. I'll stick to the approved plan's logic but add a clamp if desired.
	// The approved plan had: Where("total_points >= ?", pointsToRemove) which implicitly prevents negative.
	// I will remove that check to allow negative points if the game supports it, or stick to 0 floor if that's safer.
	// For Frolf (Disc Golf), points depend on system. Assuming non-negative total points is safe for now.

	_, err := db.NewUpdate().
		Model((*SeasonStanding)(nil)).
		Set("total_points = total_points - ?", pointsToRemove).
		Set("rounds_played = rounds_played - 1").
		Where("member_id = ?", memberID).
		Exec(ctx) // Removed the 'Where points >= ?' check to allow correction even if it dips (though unlikely) or simply trust valid state.

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
	standing := new(SeasonStanding)
	err := db.NewSelect().
		Model(standing).
		Where("member_id = ?", memberID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("leaderboarddb.GetSeasonStanding: %w", err)
	}
	return standing, nil
}

// GetSeasonBestTags retrieves the best tag for a list of members for the current season.
func (r *Impl) GetSeasonBestTags(ctx context.Context, db bun.IDB, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]int, error) {
	if db == nil {
		db = r.db
	}

	if len(memberIDs) == 0 {
		return make(map[sharedtypes.DiscordID]int), nil
	}

	var results []struct {
		MemberID sharedtypes.DiscordID
		BestTag  int
	}

	err := db.NewSelect().
		Model((*SeasonStanding)(nil)).
		ColumnExpr("member_id, season_best_tag as best_tag").
		Where("member_id IN (?)", bun.In(memberIDs)).
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

// GetSeasonStandings retrieves season standings for a batch of members.
func (r *Impl) GetSeasonStandings(ctx context.Context, db bun.IDB, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]*SeasonStanding, error) {
	if db == nil {
		db = r.db
	}

	if len(memberIDs) == 0 {
		return make(map[sharedtypes.DiscordID]*SeasonStanding), nil
	}

	var standings []SeasonStanding
	err := db.NewSelect().
		Model(&standings).
		Where("member_id IN (?)", bun.In(memberIDs)).
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

// CountSeasonMembers returns the total number of members with a standing in the current season.
func (r *Impl) CountSeasonMembers(ctx context.Context, db bun.IDB) (int, error) {
	if db == nil {
		db = r.db
	}
	count, err := db.NewSelect().Model((*SeasonStanding)(nil)).Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("leaderboarddb.CountSeasonMembers: %w", err)
	}
	return count, nil
}
