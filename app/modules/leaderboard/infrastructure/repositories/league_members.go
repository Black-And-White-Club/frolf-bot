package leaderboarddb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// LeagueMemberRepository defines operations on the league_members table.
type LeagueMemberRepository interface {
	// GetMembersByGuild retrieves all members with tags for a guild.
	GetMembersByGuild(ctx context.Context, db bun.IDB, guildID string) ([]LeagueMember, error)

	// GetMembersByIDs retrieves specific members by guild and member IDs.
	GetMembersByIDs(ctx context.Context, db bun.IDB, guildID string, memberIDs []string) ([]LeagueMember, error)

	// GetMemberByID retrieves a single member.
	GetMemberByID(ctx context.Context, db bun.IDB, guildID, memberID string) (*LeagueMember, error)

	// GetMemberByTag retrieves the member holding a specific tag in a guild.
	GetMemberByTag(ctx context.Context, db bun.IDB, guildID string, tag int) (*LeagueMember, error)

	// GetMembersByTags retrieves members holding specific tags in a guild.
	GetMembersByTags(ctx context.Context, db bun.IDB, guildID string, tags []int) ([]LeagueMember, error)

	// UpsertMember creates or updates a league member.
	UpsertMember(ctx context.Context, db bun.IDB, member *LeagueMember) error

	// BulkUpsertMembers creates or updates multiple league members.
	BulkUpsertMembers(ctx context.Context, db bun.IDB, members []LeagueMember) error

	// ClearAllTags sets current_tag=NULL for all members in a guild (for tag reset).
	ClearAllTags(ctx context.Context, db bun.IDB, guildID string) error

	// AcquireGuildLock acquires a pg_advisory_xact_lock for the guild.
	// Must be called within a transaction.
	AcquireGuildLock(ctx context.Context, db bun.IDB, guildID string) error
}

// LeagueMemberRepo implements LeagueMemberRepository.
type LeagueMemberRepo struct{}

func NewLeagueMemberRepo() LeagueMemberRepository {
	return &LeagueMemberRepo{}
}

func (r *LeagueMemberRepo) GetMembersByGuild(ctx context.Context, db bun.IDB, guildID string) ([]LeagueMember, error) {
	var members []LeagueMember
	err := db.NewSelect().
		Model(&members).
		Where("guild_id = ?", guildID).
		Order("current_tag ASC NULLS LAST").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("leaguemember.GetMembersByGuild: %w", err)
	}
	return members, nil
}

func (r *LeagueMemberRepo) GetMembersByIDs(ctx context.Context, db bun.IDB, guildID string, memberIDs []string) ([]LeagueMember, error) {
	if len(memberIDs) == 0 {
		return nil, nil
	}
	var members []LeagueMember
	err := db.NewSelect().
		Model(&members).
		Where("guild_id = ?", guildID).
		Where("member_id IN (?)", bun.In(memberIDs)).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("leaguemember.GetMembersByIDs: %w", err)
	}
	return members, nil
}

func (r *LeagueMemberRepo) GetMemberByID(ctx context.Context, db bun.IDB, guildID, memberID string) (*LeagueMember, error) {
	member := new(LeagueMember)
	err := db.NewSelect().
		Model(member).
		Where("guild_id = ?", guildID).
		Where("member_id = ?", memberID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("leaguemember.GetMemberByID: %w", err)
	}
	return member, nil
}

func (r *LeagueMemberRepo) GetMemberByTag(ctx context.Context, db bun.IDB, guildID string, tag int) (*LeagueMember, error) {
	member := new(LeagueMember)
	err := db.NewSelect().
		Model(member).
		Where("guild_id = ?", guildID).
		Where("current_tag = ?", tag).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("leaguemember.GetMemberByTag: %w", err)
	}
	return member, nil
}

func (r *LeagueMemberRepo) GetMembersByTags(ctx context.Context, db bun.IDB, guildID string, tags []int) ([]LeagueMember, error) {
	if len(tags) == 0 {
		return nil, nil
	}
	var members []LeagueMember
	err := db.NewSelect().
		Model(&members).
		Where("guild_id = ?", guildID).
		Where("current_tag IN (?)", bun.In(tags)).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("leaguemember.GetMembersByTags: %w", err)
	}
	return members, nil
}

func (r *LeagueMemberRepo) UpsertMember(ctx context.Context, db bun.IDB, member *LeagueMember) error {
	now := time.Now().UTC()
	member.UpdatedAt = now
	member.LastActiveAt = now

	_, err := db.NewInsert().
		Model(member).
		On("CONFLICT (guild_id, member_id) DO UPDATE").
		Set("current_tag = EXCLUDED.current_tag").
		Set("last_active_at = EXCLUDED.last_active_at").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("leaguemember.UpsertMember: %w", err)
	}
	return nil
}

func (r *LeagueMemberRepo) BulkUpsertMembers(ctx context.Context, db bun.IDB, members []LeagueMember) error {
	if len(members) == 0 {
		return nil
	}
	now := time.Now().UTC()
	for i := range members {
		members[i].UpdatedAt = now
		members[i].LastActiveAt = now
	}

	_, err := db.NewInsert().
		Model(&members).
		On("CONFLICT (guild_id, member_id) DO UPDATE").
		Set("current_tag = EXCLUDED.current_tag").
		Set("last_active_at = EXCLUDED.last_active_at").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("leaguemember.BulkUpsertMembers: %w", err)
	}
	return nil
}

func (r *LeagueMemberRepo) ClearAllTags(ctx context.Context, db bun.IDB, guildID string) error {
	_, err := db.NewUpdate().
		Model((*LeagueMember)(nil)).
		Set("current_tag = NULL").
		Set("updated_at = ?", time.Now().UTC()).
		Where("guild_id = ?", guildID).
		Where("current_tag IS NOT NULL").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("leaguemember.ClearAllTags: %w", err)
	}
	return nil
}

func (r *LeagueMemberRepo) AcquireGuildLock(ctx context.Context, db bun.IDB, guildID string) error {
	// Use a 64-bit hash (MD5 prefix) to minimize collision probability.
	// pg_advisory_xact_lock(bigint)
	_, err := db.NewRaw("SELECT pg_advisory_xact_lock(('x' || substr(md5(?), 1, 16))::bit(64)::bigint)", guildID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("leaguemember.AcquireGuildLock: %w", err)
	}
	return nil
}
