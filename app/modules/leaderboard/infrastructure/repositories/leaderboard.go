package leaderboarddb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
)

// LeaderboardDBImpl implements the LeaderboardDB interface using bun.
type LeaderboardDBImpl struct {
	DB *bun.DB
}

// NewLeaderboardDBImpl creates a new LeaderboardDBImpl.
func NewLeaderboardDBImpl(db *bun.DB) *LeaderboardDBImpl {
	return &LeaderboardDBImpl{DB: db}
}

// GetLeaderboard retrieves the active leaderboard.
func (ldb *LeaderboardDBImpl) GetLeaderboard(ctx context.Context) (*Leaderboard, error) {
	var leaderboard Leaderboard
	err := ldb.DB.NewSelect().
		Model(&leaderboard).
		Where("active = ?", true).
		OrderExpr("id ASC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active leaderboard found")
		}
		return nil, fmt.Errorf("failed to fetch leaderboard: %w", err)
	}
	return &leaderboard, nil
}

// DeactivateCurrentLeaderboard deactivates the currently active leaderboard.
func (ldb *LeaderboardDBImpl) DeactivateCurrentLeaderboard(ctx context.Context) error {
	_, err := ldb.DB.NewUpdate().
		Model((*Leaderboard)(nil)).
		Set("active = ?", false).
		Where("active = ?", true).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to deactivate current leaderboard: %w", err)
	}
	return nil
}

// InsertLeaderboard inserts a new leaderboard into the database.
func (ldb *LeaderboardDBImpl) InsertLeaderboard(ctx context.Context, leaderboardData map[int]string, active bool) error {
	newLeaderboard := &Leaderboard{
		LeaderboardData: leaderboardData,
		Active:          active,
	}

	_, err := ldb.DB.NewInsert().
		Model(newLeaderboard).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert new leaderboard: %w", err)
	}
	return nil
}

// UpdateLeaderboard updates the leaderboard with new entries (replaces the old one).
func (ldb *LeaderboardDBImpl) UpdateLeaderboard(ctx context.Context, entries map[int]string) error {
	tx, err := ldb.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = ldb.DeactivateCurrentLeaderboard(ctx); err != nil {
		return fmt.Errorf("failed to deactivate current leaderboard: %w", err)
	}

	if err := ldb.InsertLeaderboard(ctx, entries, true); err != nil {
		return fmt.Errorf("failed to insert new leaderboard: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// AssignTag assigns a tag to a user.
func (ldb *LeaderboardDBImpl) AssignTag(ctx context.Context, discordID string, tagNumber int) error {
	return ldb.updateLeaderboardData(ctx, func(leaderboardData map[int]string) error {
		if _, taken := leaderboardData[tagNumber]; taken {
			return fmt.Errorf("tag %d is already assigned", tagNumber)
		}
		leaderboardData[tagNumber] = discordID
		return nil
	})
}

// SwapTags swaps the tags of two users in the leaderboard.
func (ldb *LeaderboardDBImpl) SwapTags(ctx context.Context, requestorID, targetID string) error {
	return ldb.updateLeaderboardData(ctx, func(leaderboardData map[int]string) error {
		var requestorTag, targetTag int
		for tag, id := range leaderboardData {
			if id == requestorID {
				requestorTag = tag
			} else if id == targetID {
				targetTag = tag
			}
		}

		if requestorTag == 0 || targetTag == 0 {
			return fmt.Errorf("one or both users not found on the leaderboard")
		}

		leaderboardData[requestorTag], leaderboardData[targetTag] = leaderboardData[targetTag], leaderboardData[requestorTag]
		return nil
	})
}

// GetTagByDiscordID retrieves the tag number associated with a Discord ID.
func (ldb *LeaderboardDBImpl) GetTagByDiscordID(ctx context.Context, discordID string) (int, error) {
	leaderboard, err := ldb.GetLeaderboard(ctx)
	if err != nil {
		return 0, fmt.Errorf("GetTagByDiscordID: failed to get leaderboard: %w", err)
	}

	for tag, id := range leaderboard.LeaderboardData {
		if id == discordID {
			return tag, nil
		}
	}

	return 0, nil
}

// CheckTagAvailability checks if a tag number is available.
func (ldb *LeaderboardDBImpl) CheckTagAvailability(ctx context.Context, tagNumber int) (bool, error) {
	leaderboard, err := ldb.GetLeaderboard(ctx)
	if err != nil {
		return false, fmt.Errorf("CheckTagAvailability: failed to get leaderboard: %w", err)
	}

	_, taken := leaderboard.LeaderboardData[tagNumber]
	return !taken, nil
}

// updateLeaderboardData retrieves the active leaderboard, applies the provided function to modify its data,
// and then updates the leaderboard in the database within a transaction.
func (ldb *LeaderboardDBImpl) updateLeaderboardData(ctx context.Context, updateFunc func(map[int]string) error) error {
	tx, err := ldb.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	var leaderboard Leaderboard
	err = tx.NewSelect().
		Model(&leaderboard).
		Where("active = ?", true).
		OrderExpr("id ASC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no active leaderboard found")
		}
		return fmt.Errorf("failed to fetch leaderboard: %w", err)
	}

	if err := updateFunc(leaderboard.LeaderboardData); err != nil {
		return err
	}

	_, err = tx.NewUpdate().
		Model(&leaderboard).
		Set("leaderboard_data = ?", leaderboard.LeaderboardData).
		Where("id = ?", leaderboard.ID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update leaderboard data: %w", err)
	}

	return err
}
