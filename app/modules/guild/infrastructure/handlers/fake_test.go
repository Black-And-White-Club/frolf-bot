package guildhandlers

import (
	"context"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guildservice "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application"
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

// Ensure the fake satisfies the Service interface
var _ guildservice.Service = (*FakeGuildService)(nil)
