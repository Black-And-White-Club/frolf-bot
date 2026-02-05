package clubservice

import (
	"context"

	clubdb "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ------------------------
// Fake Club Repo
// ------------------------

type FakeClubRepo struct {
	trace []string

	GetByUUIDFunc           func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error)
	GetByDiscordGuildIDFunc func(ctx context.Context, db bun.IDB, guildID string) (*clubdb.Club, error)
	UpsertFunc              func(ctx context.Context, db bun.IDB, club *clubdb.Club) error
	UpdateNameFunc          func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, name string) error
}

func NewFakeClubRepo() *FakeClubRepo {
	return &FakeClubRepo{
		trace: []string{},
	}
}

func (f *FakeClubRepo) record(step string) {
	f.trace = append(f.trace, step)
}

// --- Repository Interface Implementation ---

func (f *FakeClubRepo) GetByUUID(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
	f.record("GetByUUID")
	if f.GetByUUIDFunc != nil {
		return f.GetByUUIDFunc(ctx, db, clubUUID)
	}
	return nil, clubdb.ErrNotFound
}

func (f *FakeClubRepo) GetByDiscordGuildID(ctx context.Context, db bun.IDB, guildID string) (*clubdb.Club, error) {
	f.record("GetByDiscordGuildID")
	if f.GetByDiscordGuildIDFunc != nil {
		return f.GetByDiscordGuildIDFunc(ctx, db, guildID)
	}
	return nil, clubdb.ErrNotFound
}

func (f *FakeClubRepo) Upsert(ctx context.Context, db bun.IDB, club *clubdb.Club) error {
	f.record("Upsert")
	if f.UpsertFunc != nil {
		return f.UpsertFunc(ctx, db, club)
	}
	return nil
}

func (f *FakeClubRepo) UpdateName(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, name string) error {
	f.record("UpdateName")
	if f.UpdateNameFunc != nil {
		return f.UpdateNameFunc(ctx, db, clubUUID, name)
	}
	return nil
}

// --- Accessors for assertions ---

func (f *FakeClubRepo) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

// Ensure the fake actually satisfies the interface
var _ clubdb.Repository = (*FakeClubRepo)(nil)
