package clubhandlers

import (
	"context"

	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
	clubservice "github.com/Black-And-White-Club/frolf-bot/app/modules/club/application"
	"github.com/google/uuid"
)

// ------------------------
// Fake Club Service
// ------------------------

type FakeClubService struct {
	trace []string

	GetClubFunc               func(ctx context.Context, clubUUID uuid.UUID) (*clubtypes.ClubInfo, error)
	UpsertClubFromDiscordFunc func(ctx context.Context, guildID, name string, iconURL *string) (*clubtypes.ClubInfo, error)
}

func NewFakeClubService() *FakeClubService {
	return &FakeClubService{
		trace: []string{},
	}
}

func (f *FakeClubService) record(step string) {
	f.trace = append(f.trace, step)
}

// --- Service Interface Implementation ---

func (f *FakeClubService) GetClub(ctx context.Context, clubUUID uuid.UUID) (*clubtypes.ClubInfo, error) {
	f.record("GetClub")
	if f.GetClubFunc != nil {
		return f.GetClubFunc(ctx, clubUUID)
	}
	return nil, nil
}

func (f *FakeClubService) UpsertClubFromDiscord(ctx context.Context, guildID, name string, iconURL *string) (*clubtypes.ClubInfo, error) {
	f.record("UpsertClubFromDiscord")
	if f.UpsertClubFromDiscordFunc != nil {
		return f.UpsertClubFromDiscordFunc(ctx, guildID, name, iconURL)
	}
	return nil, nil
}

// --- Accessors for assertions ---

func (f *FakeClubService) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

// Ensure the fake actually satisfies the interface
var _ clubservice.Service = (*FakeClubService)(nil)
