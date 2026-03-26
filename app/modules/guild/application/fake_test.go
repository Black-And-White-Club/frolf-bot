package guildservice

import (
	"context"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ------------------------
// Fake Guild Repo
// ------------------------

// FakeGuildRepository provides a programmable stub for the guilddb.Repository interface.
type FakeGuildRepository struct {
	trace []string

	GetConfigFunc                 func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error)
	GetConfigIncludeDeletedFunc   func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error)
	SaveConfigFunc                func(ctx context.Context, db bun.IDB, config *guildtypes.GuildConfig) error
	UpdateConfigFunc              func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, updates *guilddb.UpdateFields) error
	DeleteConfigFunc              func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) error
	ResolveEntitlementsFunc       func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error)
	UpsertFeatureOverrideFunc     func(ctx context.Context, db bun.IDB, override *guilddb.ClubFeatureOverride, audit *guilddb.ClubFeatureAccessAudit) error
	DeleteFeatureOverrideFunc     func(ctx context.Context, db bun.IDB, clubUUID string, featureKey string, audit *guilddb.ClubFeatureAccessAudit) error
	ListFeatureAccessAuditFunc    func(ctx context.Context, db bun.IDB, clubUUID string, featureKey string) ([]guilddb.ClubFeatureAccessAudit, error)
	InsertOutboxEventFunc         func(ctx context.Context, db bun.IDB, topic string, payload []byte) error
	PollAndLockOutboxEventsFunc   func(ctx context.Context, db bun.IDB, limit int) ([]guilddb.GuildOutboxEvent, error)
	MarkOutboxEventPublishedFunc  func(ctx context.Context, db bun.IDB, id string) error
	GetClubIDByDiscordGuildIDFunc func(ctx context.Context, guildID string) (uuid.UUID, error)
}

// Trace returns the sequence of method calls made to the fake.
func (f *FakeGuildRepository) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

// NewFakeGuildRepository initializes a new FakeGuildRepository with an empty trace.
func NewFakeGuildRepository() *FakeGuildRepository {
	return &FakeGuildRepository{
		trace: []string{},
	}
}

func (f *FakeGuildRepository) record(step string) {
	f.trace = append(f.trace, step)
}

// --- Repository Interface Implementation ---

func (f *FakeGuildRepository) GetConfig(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error) {
	f.record("GetConfig")
	if f.GetConfigFunc != nil {
		return f.GetConfigFunc(ctx, db, guildID)
	}
	// Default: Return ErrNotFound to simulate a clean state
	return nil, guilddb.ErrNotFound
}

func (f *FakeGuildRepository) GetConfigIncludeDeleted(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error) {
	f.record("GetConfigIncludeDeleted")
	if f.GetConfigIncludeDeletedFunc != nil {
		return f.GetConfigIncludeDeletedFunc(ctx, db, guildID)
	}
	return nil, guilddb.ErrNotFound
}

func (f *FakeGuildRepository) SaveConfig(ctx context.Context, db bun.IDB, config *guildtypes.GuildConfig) error {
	f.record("SaveConfig")
	if f.SaveConfigFunc != nil {
		return f.SaveConfigFunc(ctx, db, config)
	}
	return nil
}

func (f *FakeGuildRepository) UpdateConfig(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, updates *guilddb.UpdateFields) error {
	f.record("UpdateConfig")
	if f.UpdateConfigFunc != nil {
		return f.UpdateConfigFunc(ctx, db, guildID, updates)
	}
	return nil
}

func (f *FakeGuildRepository) DeleteConfig(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) error {
	f.record("DeleteConfig")
	if f.DeleteConfigFunc != nil {
		return f.DeleteConfigFunc(ctx, db, guildID)
	}
	return nil
}

func (f *FakeGuildRepository) ResolveEntitlements(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
	f.record("ResolveEntitlements")
	if f.ResolveEntitlementsFunc != nil {
		return f.ResolveEntitlementsFunc(ctx, db, guildID)
	}
	return guildtypes.ResolvedClubEntitlements{}, nil
}

func (f *FakeGuildRepository) UpsertFeatureOverride(ctx context.Context, db bun.IDB, override *guilddb.ClubFeatureOverride, audit *guilddb.ClubFeatureAccessAudit) error {
	f.record("UpsertFeatureOverride")
	if f.UpsertFeatureOverrideFunc != nil {
		return f.UpsertFeatureOverrideFunc(ctx, db, override, audit)
	}
	return nil
}

func (f *FakeGuildRepository) DeleteFeatureOverride(ctx context.Context, db bun.IDB, clubUUID string, featureKey string, audit *guilddb.ClubFeatureAccessAudit) error {
	f.record("DeleteFeatureOverride")
	if f.DeleteFeatureOverrideFunc != nil {
		return f.DeleteFeatureOverrideFunc(ctx, db, clubUUID, featureKey, audit)
	}
	return nil
}

func (f *FakeGuildRepository) ListFeatureAccessAudit(ctx context.Context, db bun.IDB, clubUUID string, featureKey string) ([]guilddb.ClubFeatureAccessAudit, error) {
	f.record("ListFeatureAccessAudit")
	if f.ListFeatureAccessAuditFunc != nil {
		return f.ListFeatureAccessAuditFunc(ctx, db, clubUUID, featureKey)
	}
	return nil, nil
}

func (f *FakeGuildRepository) InsertOutboxEvent(ctx context.Context, db bun.IDB, topic string, payload []byte) error {
	f.record("InsertOutboxEvent")
	if f.InsertOutboxEventFunc != nil {
		return f.InsertOutboxEventFunc(ctx, db, topic, payload)
	}
	return nil
}

func (f *FakeGuildRepository) PollAndLockOutboxEvents(ctx context.Context, db bun.IDB, limit int) ([]guilddb.GuildOutboxEvent, error) {
	f.record("PollAndLockOutboxEvents")
	if f.PollAndLockOutboxEventsFunc != nil {
		return f.PollAndLockOutboxEventsFunc(ctx, db, limit)
	}
	return nil, nil
}

func (f *FakeGuildRepository) MarkOutboxEventPublished(ctx context.Context, db bun.IDB, id string) error {
	f.record("MarkOutboxEventPublished")
	if f.MarkOutboxEventPublishedFunc != nil {
		return f.MarkOutboxEventPublishedFunc(ctx, db, id)
	}
	return nil
}

func (f *FakeGuildRepository) GetClubIDByDiscordGuildID(ctx context.Context, guildID string) (uuid.UUID, error) {
	f.record("GetClubIDByDiscordGuildID")
	if f.GetClubIDByDiscordGuildIDFunc != nil {
		return f.GetClubIDByDiscordGuildIDFunc(ctx, guildID)
	}
	return uuid.Nil, nil
}

// Ensure the fake actually satisfies the interface
var _ guilddb.Repository = (*FakeGuildRepository)(nil)
