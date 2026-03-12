package clubdb

import (
	"context"
	"time"

	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
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

	// GetChallengeByUUID retrieves a challenge by its UUID.
	GetChallengeByUUID(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*ClubChallenge, error)

	// CreateChallenge inserts a new club challenge.
	CreateChallenge(ctx context.Context, db bun.IDB, challenge *ClubChallenge) error

	// UpdateChallenge persists challenge changes.
	UpdateChallenge(ctx context.Context, db bun.IDB, challenge *ClubChallenge) error

	// ListChallenges returns challenges for a club, optionally filtered by status.
	ListChallenges(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, statuses []clubtypes.ChallengeStatus) ([]*ClubChallenge, error)

	// GetOpenOutgoingChallenge returns the challenger's currently open outgoing challenge.
	GetOpenOutgoingChallenge(ctx context.Context, db bun.IDB, clubUUID, challengerUserUUID uuid.UUID) (*ClubChallenge, error)

	// GetAcceptedChallengeForUser returns the user's currently accepted challenge.
	GetAcceptedChallengeForUser(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID) (*ClubChallenge, error)

	// GetActiveChallengeByPair returns an active challenge for the same participant pair in either order.
	GetActiveChallengeByPair(ctx context.Context, db bun.IDB, clubUUID, userA, userB uuid.UUID) (*ClubChallenge, error)

	// ListActiveChallengesByUsers returns active challenges involving any of the provided users.
	ListActiveChallengesByUsers(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, userUUIDs []uuid.UUID) ([]*ClubChallenge, error)

	// BindChallengeMessage stores the persistent Discord message reference.
	BindChallengeMessage(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID, guildID, channelID, messageID string) error

	// CreateChallengeRoundLink inserts a new challenge-round link.
	CreateChallengeRoundLink(ctx context.Context, db bun.IDB, link *ClubChallengeRoundLink) error

	// GetActiveChallengeRoundLink returns the active round link for a challenge.
	GetActiveChallengeRoundLink(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*ClubChallengeRoundLink, error)

	// GetChallengeByActiveRound returns the active challenge linked to a round.
	GetChallengeByActiveRound(ctx context.Context, db bun.IDB, roundID uuid.UUID) (*ClubChallenge, error)

	// UnlinkActiveChallengeRound marks the current active round link as inactive.
	UnlinkActiveChallengeRound(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID, actorUserUUID *uuid.UUID, unlinkedAt time.Time) error
}
