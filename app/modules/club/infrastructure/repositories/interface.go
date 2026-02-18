package clubdb

import (
	"context"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Repository defines the contract for club persistence.
type Repository interface {
	// GetByUUID retrieves a club by its UUID.
	GetByUUID(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*Club, error)

	// GetByDiscordGuildID retrieves a club by its Discord guild ID.
	GetByDiscordGuildID(ctx context.Context, db bun.IDB, guildID string) (*Club, error)

	// GetClubsByDiscordGuildIDs retrieves clubs matching any of the given Discord guild IDs.
	GetClubsByDiscordGuildIDs(ctx context.Context, db bun.IDB, guildIDs []string) ([]*Club, error)

	// Upsert creates or updates a club.
	Upsert(ctx context.Context, db bun.IDB, club *Club) error

	// UpdateName updates a club's name.
	UpdateName(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, name string) error

	// CreateInvite inserts a new club invite.
	CreateInvite(ctx context.Context, db bun.IDB, invite *ClubInvite) error

	// GetInviteByCode retrieves an invite by its code.
	GetInviteByCode(ctx context.Context, db bun.IDB, code string) (*ClubInvite, error)

	// GetInvitesByClub lists active (non-revoked) invites for a club.
	GetInvitesByClub(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) ([]*ClubInvite, error)

	// RevokeInvite marks an invite as revoked.
	RevokeInvite(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, code string) error

	// IncrementInviteUseCount atomically increments the use_count for an invite.
	IncrementInviteUseCount(ctx context.Context, db bun.IDB, code string) error
}
