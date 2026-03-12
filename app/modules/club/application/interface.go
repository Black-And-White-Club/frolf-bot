package clubservice

import (
	"context"
	"time"

	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
	"github.com/google/uuid"
)

// ClubSuggestion represents a club that a user could join via Discord guild match.
type ClubSuggestion struct {
	UUID    string  `json:"uuid"`
	Name    string  `json:"name"`
	IconURL *string `json:"icon_url,omitempty"`
}

// InviteInfo holds the public view of an invite code.
type InviteInfo struct {
	Code      string     `json:"code"`
	ClubUUID  string     `json:"club_uuid"`
	Role      string     `json:"role"`
	MaxUses   *int       `json:"max_uses,omitempty"`
	UseCount  int        `json:"use_count"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// InvitePreview is the public club preview returned when validating an invite code.
type InvitePreview struct {
	ClubUUID string  `json:"club_uuid"`
	ClubName string  `json:"club_name"`
	IconURL  *string `json:"icon_url,omitempty"`
	Role     string  `json:"role"`
}

// CreateInviteRequest holds the parameters for creating an invite.
type CreateInviteRequest struct {
	Role          string `json:"role"`
	MaxUses       *int   `json:"max_uses,omitempty"`
	ExpiresInDays *int   `json:"expires_in_days,omitempty"`
}

// Service defines the club service contract.
type Service interface {
	// GetClub retrieves club info by UUID.
	GetClub(ctx context.Context, clubUUID uuid.UUID) (*clubtypes.ClubInfo, error)

	// UpsertClubFromDiscord creates or updates a club from Discord guild info.
	UpsertClubFromDiscord(ctx context.Context, guildID, name string, iconURL *string) (*clubtypes.ClubInfo, error)

	// GetClubSuggestions returns clubs that match Discord guilds the user belongs to
	// and that the user is not already a member of.
	GetClubSuggestions(ctx context.Context, userUUID uuid.UUID) ([]ClubSuggestion, error)

	// JoinClub adds the user as a member of the specified club, verifying Discord guild
	// membership and assigning the appropriate role.
	JoinClub(ctx context.Context, userUUID, clubUUID uuid.UUID) error

	// CreateInvite generates a new invite code for a club. The caller must have
	// admin or editor role in the club.
	CreateInvite(ctx context.Context, callerUUID, clubUUID uuid.UUID, req CreateInviteRequest) (*InviteInfo, error)

	// ListInvites returns active invite codes for a club. Requires admin or editor role.
	ListInvites(ctx context.Context, callerUUID, clubUUID uuid.UUID) ([]*InviteInfo, error)

	// RevokeInvite revokes an invite code. Requires admin or editor role.
	RevokeInvite(ctx context.Context, callerUUID, clubUUID uuid.UUID, code string) error

	// GetInvitePreview validates an invite code and returns club preview info.
	// This is a public endpoint — no auth required.
	GetInvitePreview(ctx context.Context, code string) (*InvitePreview, error)

	// JoinByCode uses an invite code to join a club. The user must be authenticated.
	JoinByCode(ctx context.Context, userUUID uuid.UUID, code string) error

	// ListChallenges returns the board view for the requested club.
	ListChallenges(ctx context.Context, req ChallengeListRequest) ([]clubtypes.ChallengeSummary, error)

	// GetChallengeDetail returns a single challenge detail view.
	GetChallengeDetail(ctx context.Context, req ChallengeDetailRequest) (*clubtypes.ChallengeDetail, error)

	// OpenChallenge creates a new challenge between two club members.
	OpenChallenge(ctx context.Context, req ChallengeOpenRequest) (*clubtypes.ChallengeDetail, error)

	// RespondToChallenge accepts or declines an open challenge.
	RespondToChallenge(ctx context.Context, req ChallengeRespondRequest) (*clubtypes.ChallengeDetail, error)

	// WithdrawChallenge withdraws a challenge.
	WithdrawChallenge(ctx context.Context, req ChallengeActionRequest) (*clubtypes.ChallengeDetail, error)

	// HideChallenge hides a challenge from the board.
	HideChallenge(ctx context.Context, req ChallengeActionRequest) (*clubtypes.ChallengeDetail, error)

	// LinkChallengeRound links a normal round to an accepted challenge.
	LinkChallengeRound(ctx context.Context, req ChallengeRoundLinkRequest) (*clubtypes.ChallengeDetail, error)

	// UnlinkChallengeRound detaches the current active round link from a challenge.
	UnlinkChallengeRound(ctx context.Context, req ChallengeActionRequest) (*clubtypes.ChallengeDetail, error)

	// BindChallengeMessage stores the persistent Discord message reference.
	BindChallengeMessage(ctx context.Context, req ChallengeMessageBindingRequest) (*clubtypes.ChallengeDetail, error)

	// ExpireChallenge expires an open or accepted-unlinked challenge.
	ExpireChallenge(ctx context.Context, req ChallengeExpireRequest) (*clubtypes.ChallengeDetail, error)

	// CompleteChallengeForRound completes a challenge when a linked round finalizes.
	CompleteChallengeForRound(ctx context.Context, req ChallengeRoundEventRequest) (*clubtypes.ChallengeDetail, error)

	// ResetChallengeForRound unlinks a challenge when its linked round is deleted.
	ResetChallengeForRound(ctx context.Context, req ChallengeRoundEventRequest) (*clubtypes.ChallengeDetail, error)

	// RefreshChallengesForMembers emits refreshed detail for active challenges affected by tag changes.
	RefreshChallengesForMembers(ctx context.Context, req ChallengeRefreshRequest) ([]clubtypes.ChallengeDetail, error)
}
