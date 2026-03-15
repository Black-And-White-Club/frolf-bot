package guildservice

import (
	"context"
	"time"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
)

// GuildConfigResult is a type alias to reduce generic verbosity.
// Defined here so it's available to both the interface and the implementation.
type GuildConfigResult = results.OperationResult[*guildtypes.GuildConfig, error]

type GrantAccessRequest struct {
	ClubUUID   uuid.UUID
	FeatureKey guildtypes.ClubFeatureKey
	Reason     string
	ExpiresAt  *time.Time
	ActorUUID  string
}

type RevokeAccessRequest struct {
	ClubUUID   uuid.UUID
	FeatureKey guildtypes.ClubFeatureKey
	Reason     string
	ActorUUID  string
}

type FeatureAccessAuditRecord struct {
	ID         int64
	ClubUUID   uuid.UUID
	GuildID    string
	FeatureKey string
	State      string
	Source     string
	Reason     string
	UpdatedBy  string
	ExpiresAt  *time.Time
	CreatedAt  time.Time
}

// Service defines the interface for guild operations.
type Service interface {
	CreateGuildConfig(ctx context.Context, config *guildtypes.GuildConfig) (GuildConfigResult, error)
	GetGuildConfig(ctx context.Context, guildID sharedtypes.GuildID) (GuildConfigResult, error)
	UpdateGuildConfig(ctx context.Context, config *guildtypes.GuildConfig) (GuildConfigResult, error)
	DeleteGuildConfig(ctx context.Context, guildID sharedtypes.GuildID) (GuildConfigResult, error)

	ResolveClubEntitlements(ctx context.Context, clubUUID uuid.UUID) (guildtypes.ResolvedClubEntitlements, error)
	ResolveClubFeature(ctx context.Context, clubUUID uuid.UUID, featureKey guildtypes.ClubFeatureKey) (guildtypes.ClubFeatureAccess, error)
	ResolveClubFeatureByGuildID(ctx context.Context, guildID sharedtypes.GuildID, featureKey guildtypes.ClubFeatureKey) (guildtypes.ClubFeatureAccess, error)

	GrantFeatureAccess(ctx context.Context, req GrantAccessRequest) error
	RevokeFeatureAccess(ctx context.Context, req RevokeAccessRequest) error
	GetFeatureAccessAudit(ctx context.Context, clubUUID uuid.UUID, featureKey guildtypes.ClubFeatureKey) ([]FeatureAccessAuditRecord, error)
}
