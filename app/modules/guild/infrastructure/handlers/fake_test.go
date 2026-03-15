package guildhandlers

import (
	"context"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guildservice "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application"
	"github.com/google/uuid"
)

// ------------------------
// Fake Guild Service
// ------------------------

// FakeGuildService provides a programmable stub for the guildservice.Service interface.
// Use this when testing handlers or other services that depend on GuildService.
type FakeGuildService struct {
	trace []string

	CreateGuildConfigFunc func(ctx context.Context, config *guildtypes.GuildConfig) (guildservice.GuildConfigResult, error)
	GetGuildConfigFunc    func(ctx context.Context, guildID sharedtypes.GuildID) (guildservice.GuildConfigResult, error)
	UpdateGuildConfigFunc func(ctx context.Context, config *guildtypes.GuildConfig) (guildservice.GuildConfigResult, error)
	DeleteGuildConfigFunc func(ctx context.Context, guildID sharedtypes.GuildID) (guildservice.GuildConfigResult, error)

	ResolveClubEntitlementsFunc     func(ctx context.Context, clubUUID uuid.UUID) (guildtypes.ResolvedClubEntitlements, error)
	ResolveClubFeatureFunc          func(ctx context.Context, clubUUID uuid.UUID, featureKey guildtypes.ClubFeatureKey) (guildtypes.ClubFeatureAccess, error)
	ResolveClubFeatureByGuildIDFunc func(ctx context.Context, guildID sharedtypes.GuildID, featureKey guildtypes.ClubFeatureKey) (guildtypes.ClubFeatureAccess, error)
	GrantFeatureAccessFunc          func(ctx context.Context, req guildservice.GrantAccessRequest) error
	RevokeFeatureAccessFunc         func(ctx context.Context, req guildservice.RevokeAccessRequest) error
	GetFeatureAccessAuditFunc       func(ctx context.Context, clubUUID uuid.UUID, featureKey guildtypes.ClubFeatureKey) ([]guildservice.FeatureAccessAuditRecord, error)
}

// NewFakeGuildService initializes a new FakeGuildService.
func NewFakeGuildService() *FakeGuildService {
	return &FakeGuildService{
		trace: []string{},
	}
}

func (f *FakeGuildService) record(step string) {
	f.trace = append(f.trace, step)
}

// Trace returns the sequence of service methods called.
func (f *FakeGuildService) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

// --- Service Interface Implementation ---

func (f *FakeGuildService) CreateGuildConfig(ctx context.Context, config *guildtypes.GuildConfig) (guildservice.GuildConfigResult, error) {
	f.record("CreateGuildConfig")
	if f.CreateGuildConfigFunc != nil {
		return f.CreateGuildConfigFunc(ctx, config)
	}
	return guildservice.GuildConfigResult{}, nil
}

func (f *FakeGuildService) GetGuildConfig(ctx context.Context, guildID sharedtypes.GuildID) (guildservice.GuildConfigResult, error) {
	f.record("GetGuildConfig")
	if f.GetGuildConfigFunc != nil {
		return f.GetGuildConfigFunc(ctx, guildID)
	}
	return guildservice.GuildConfigResult{}, nil
}

func (f *FakeGuildService) UpdateGuildConfig(ctx context.Context, config *guildtypes.GuildConfig) (guildservice.GuildConfigResult, error) {
	f.record("UpdateGuildConfig")
	if f.UpdateGuildConfigFunc != nil {
		return f.UpdateGuildConfigFunc(ctx, config)
	}
	return guildservice.GuildConfigResult{}, nil
}

func (f *FakeGuildService) DeleteGuildConfig(ctx context.Context, guildID sharedtypes.GuildID) (guildservice.GuildConfigResult, error) {
	f.record("DeleteGuildConfig")
	if f.DeleteGuildConfigFunc != nil {
		return f.DeleteGuildConfigFunc(ctx, guildID)
	}
	return guildservice.GuildConfigResult{}, nil
}

func (f *FakeGuildService) ResolveClubEntitlements(ctx context.Context, clubUUID uuid.UUID) (guildtypes.ResolvedClubEntitlements, error) {
	f.record("ResolveClubEntitlements")
	if f.ResolveClubEntitlementsFunc != nil {
		return f.ResolveClubEntitlementsFunc(ctx, clubUUID)
	}
	return guildtypes.ResolvedClubEntitlements{}, nil
}

func (f *FakeGuildService) ResolveClubFeature(ctx context.Context, clubUUID uuid.UUID, featureKey guildtypes.ClubFeatureKey) (guildtypes.ClubFeatureAccess, error) {
	f.record("ResolveClubFeature")
	if f.ResolveClubFeatureFunc != nil {
		return f.ResolveClubFeatureFunc(ctx, clubUUID, featureKey)
	}
	return guildtypes.ClubFeatureAccess{}, nil
}

func (f *FakeGuildService) ResolveClubFeatureByGuildID(ctx context.Context, guildID sharedtypes.GuildID, featureKey guildtypes.ClubFeatureKey) (guildtypes.ClubFeatureAccess, error) {
	f.record("ResolveClubFeatureByGuildID")
	if f.ResolveClubFeatureByGuildIDFunc != nil {
		return f.ResolveClubFeatureByGuildIDFunc(ctx, guildID, featureKey)
	}
	return guildtypes.ClubFeatureAccess{}, nil
}

func (f *FakeGuildService) GrantFeatureAccess(ctx context.Context, req guildservice.GrantAccessRequest) error {
	f.record("GrantFeatureAccess")
	if f.GrantFeatureAccessFunc != nil {
		return f.GrantFeatureAccessFunc(ctx, req)
	}
	return nil
}

func (f *FakeGuildService) RevokeFeatureAccess(ctx context.Context, req guildservice.RevokeAccessRequest) error {
	f.record("RevokeFeatureAccess")
	if f.RevokeFeatureAccessFunc != nil {
		return f.RevokeFeatureAccessFunc(ctx, req)
	}
	return nil
}

func (f *FakeGuildService) GetFeatureAccessAudit(ctx context.Context, clubUUID uuid.UUID, featureKey guildtypes.ClubFeatureKey) ([]guildservice.FeatureAccessAuditRecord, error) {
	f.record("GetFeatureAccessAudit")
	if f.GetFeatureAccessAuditFunc != nil {
		return f.GetFeatureAccessAuditFunc(ctx, clubUUID, featureKey)
	}
	return nil, nil
}

// Ensure the fake satisfies the Service interface
var _ guildservice.Service = (*FakeGuildService)(nil)
