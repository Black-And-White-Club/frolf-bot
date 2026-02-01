package guildservice

import (
	"context"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// ------------------------
// Fake Guild Repo
// ------------------------

// FakeGuildRepository provides a programmable stub for the guilddb.Repository interface.
type FakeGuildRepository struct {
	trace []string

	GetConfigFunc               func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error)
	GetConfigIncludeDeletedFunc func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error)
	SaveConfigFunc              func(ctx context.Context, db bun.IDB, config *guildtypes.GuildConfig) error
	UpdateConfigFunc            func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, updates *guilddb.UpdateFields) error
	DeleteConfigFunc            func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) error
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

// Ensure the fake actually satisfies the interface
var _ guilddb.Repository = (*FakeGuildRepository)(nil)
