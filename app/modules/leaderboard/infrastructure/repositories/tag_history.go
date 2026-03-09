package leaderboarddb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// TagHistoryRepository defines operations on the tag_history table.
type TagHistoryRepository interface {
	// BulkInsertTagHistory inserts multiple tag history entries.
	BulkInsertTagHistory(ctx context.Context, db bun.IDB, entries []TagHistoryEntry) error

	// GetTagHistoryForRound retrieves all tag changes for a specific round.
	GetTagHistoryForRound(ctx context.Context, db bun.IDB, guildID string, roundID uuid.UUID) ([]TagHistoryEntry, error)

	// GetTagHistoryForMember retrieves tag history for a specific member.
	GetTagHistoryForMember(ctx context.Context, db bun.IDB, guildID, memberID string, limit int) ([]TagHistoryEntry, error)

	// GetLatestTagHistory retrieves the most recent tag changes for a guild.
	GetLatestTagHistory(ctx context.Context, db bun.IDB, guildID string, limit int) ([]TagHistoryEntry, error)

	// GetTagHistoryForTag retrieves the full history of a specific tag number.
	GetTagHistoryForTag(ctx context.Context, db bun.IDB, guildID string, tagNumber int, limit int) ([]TagHistoryEntry, error)

	// GetTagHistoryForGuild retrieves tag history for an entire guild since a given time.
	GetTagHistoryForGuild(ctx context.Context, db bun.IDB, guildID string, since time.Time) ([]TagHistoryEntry, error)
}

// TagHistoryRepo implements TagHistoryRepository.
type TagHistoryRepo struct{}

func NewTagHistoryRepo() TagHistoryRepository {
	return &TagHistoryRepo{}
}

func (r *TagHistoryRepo) BulkInsertTagHistory(ctx context.Context, db bun.IDB, entries []TagHistoryEntry) error {
	if len(entries) == 0 {
		return nil
	}
	_, err := db.NewInsert().Model(&entries).Exec(ctx)
	if err != nil {
		return fmt.Errorf("taghistory.BulkInsertTagHistory: %w", err)
	}
	return nil
}

func (r *TagHistoryRepo) GetTagHistoryForRound(ctx context.Context, db bun.IDB, guildID string, roundID uuid.UUID) ([]TagHistoryEntry, error) {
	var entries []TagHistoryEntry
	err := db.NewSelect().
		Model(&entries).
		Where("guild_id = ?", guildID).
		Where("round_id = ?", roundID).
		Order("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("taghistory.GetTagHistoryForRound: %w", err)
	}
	return entries, nil
}

func (r *TagHistoryRepo) GetTagHistoryForMember(ctx context.Context, db bun.IDB, guildID, memberID string, limit int) ([]TagHistoryEntry, error) {
	var entries []TagHistoryEntry

	// Build UNION via bun query builder. The outer query must NOT use .Model()
	// because that injects the model's table name as FROM, overriding TableExpr.
	// Instead, pass the destination to Scan directly.
	newQ := db.NewSelect().
		TableExpr("tag_history").
		ColumnExpr("*").
		Where("guild_id = ?", guildID).
		Where("new_member_id = ?", memberID)
	oldQ := db.NewSelect().
		TableExpr("tag_history").
		ColumnExpr("*").
		Where("guild_id = ?", guildID).
		Where("old_member_id = ?", memberID)

	q := db.NewSelect().
		TableExpr("(?) AS combined", newQ.Union(oldQ)).
		ColumnExpr("combined.*").
		OrderExpr("combined.created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}

	err := q.Scan(ctx, &entries)
	if err != nil {
		return nil, fmt.Errorf("taghistory.GetTagHistoryForMember: %w", err)
	}
	return entries, nil
}

func (r *TagHistoryRepo) GetLatestTagHistory(ctx context.Context, db bun.IDB, guildID string, limit int) ([]TagHistoryEntry, error) {
	var entries []TagHistoryEntry
	q := db.NewSelect().
		Model(&entries).
		Where("guild_id = ?", guildID).
		Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("taghistory.GetLatestTagHistory: %w", err)
	}
	return entries, nil
}

func (r *TagHistoryRepo) GetTagHistoryForTag(ctx context.Context, db bun.IDB, guildID string, tagNumber int, limit int) ([]TagHistoryEntry, error) {
	var entries []TagHistoryEntry
	q := db.NewSelect().
		Model(&entries).
		Where("guild_id = ?", guildID).
		Where("tag_number = ?", tagNumber).
		Order("created_at ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("taghistory.GetTagHistoryForTag: %w", err)
	}
	return entries, nil
}

func (r *TagHistoryRepo) GetTagHistoryForGuild(ctx context.Context, db bun.IDB, guildID string, since time.Time) ([]TagHistoryEntry, error) {
	var entries []TagHistoryEntry
	err := db.NewSelect().
		Model(&entries).
		Where("guild_id = ?", guildID).
		Where("created_at >= ?", since).
		Order("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("taghistory.GetTagHistoryForGuild: %w", err)
	}
	return entries, nil
}
