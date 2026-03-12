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

	GetClubFunc                     func(ctx context.Context, clubUUID uuid.UUID) (*clubtypes.ClubInfo, error)
	UpsertClubFromDiscordFunc       func(ctx context.Context, guildID, name string, iconURL *string) (*clubtypes.ClubInfo, error)
	GetClubSuggestionsFunc          func(ctx context.Context, userUUID uuid.UUID) ([]clubservice.ClubSuggestion, error)
	JoinClubFunc                    func(ctx context.Context, userUUID, clubUUID uuid.UUID) error
	CreateInviteFunc                func(ctx context.Context, callerUUID, clubUUID uuid.UUID, req clubservice.CreateInviteRequest) (*clubservice.InviteInfo, error)
	ListInvitesFunc                 func(ctx context.Context, callerUUID, clubUUID uuid.UUID) ([]*clubservice.InviteInfo, error)
	RevokeInviteFunc                func(ctx context.Context, callerUUID, clubUUID uuid.UUID, code string) error
	GetInvitePreviewFunc            func(ctx context.Context, code string) (*clubservice.InvitePreview, error)
	JoinByCodeFunc                  func(ctx context.Context, userUUID uuid.UUID, code string) error
	ListChallengesFunc              func(ctx context.Context, req clubservice.ChallengeListRequest) ([]clubtypes.ChallengeSummary, error)
	GetChallengeDetailFunc          func(ctx context.Context, req clubservice.ChallengeDetailRequest) (*clubtypes.ChallengeDetail, error)
	OpenChallengeFunc               func(ctx context.Context, req clubservice.ChallengeOpenRequest) (*clubtypes.ChallengeDetail, error)
	RespondToChallengeFunc          func(ctx context.Context, req clubservice.ChallengeRespondRequest) (*clubtypes.ChallengeDetail, error)
	WithdrawChallengeFunc           func(ctx context.Context, req clubservice.ChallengeActionRequest) (*clubtypes.ChallengeDetail, error)
	HideChallengeFunc               func(ctx context.Context, req clubservice.ChallengeActionRequest) (*clubtypes.ChallengeDetail, error)
	LinkChallengeRoundFunc          func(ctx context.Context, req clubservice.ChallengeRoundLinkRequest) (*clubtypes.ChallengeDetail, error)
	UnlinkChallengeRoundFunc        func(ctx context.Context, req clubservice.ChallengeActionRequest) (*clubtypes.ChallengeDetail, error)
	BindChallengeMessageFunc        func(ctx context.Context, req clubservice.ChallengeMessageBindingRequest) (*clubtypes.ChallengeDetail, error)
	ExpireChallengeFunc             func(ctx context.Context, req clubservice.ChallengeExpireRequest) (*clubtypes.ChallengeDetail, error)
	CompleteChallengeForRoundFunc   func(ctx context.Context, req clubservice.ChallengeRoundEventRequest) (*clubtypes.ChallengeDetail, error)
	ResetChallengeForRoundFunc      func(ctx context.Context, req clubservice.ChallengeRoundEventRequest) (*clubtypes.ChallengeDetail, error)
	RefreshChallengesForMembersFunc func(ctx context.Context, req clubservice.ChallengeRefreshRequest) ([]clubtypes.ChallengeDetail, error)
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

func (f *FakeClubService) GetClubSuggestions(ctx context.Context, userUUID uuid.UUID) ([]clubservice.ClubSuggestion, error) {
	f.record("GetClubSuggestions")
	if f.GetClubSuggestionsFunc != nil {
		return f.GetClubSuggestionsFunc(ctx, userUUID)
	}
	return nil, nil
}

func (f *FakeClubService) JoinClub(ctx context.Context, userUUID, clubUUID uuid.UUID) error {
	f.record("JoinClub")
	if f.JoinClubFunc != nil {
		return f.JoinClubFunc(ctx, userUUID, clubUUID)
	}
	return nil
}

func (f *FakeClubService) CreateInvite(ctx context.Context, callerUUID, clubUUID uuid.UUID, req clubservice.CreateInviteRequest) (*clubservice.InviteInfo, error) {
	f.record("CreateInvite")
	if f.CreateInviteFunc != nil {
		return f.CreateInviteFunc(ctx, callerUUID, clubUUID, req)
	}
	return nil, nil
}

func (f *FakeClubService) ListInvites(ctx context.Context, callerUUID, clubUUID uuid.UUID) ([]*clubservice.InviteInfo, error) {
	f.record("ListInvites")
	if f.ListInvitesFunc != nil {
		return f.ListInvitesFunc(ctx, callerUUID, clubUUID)
	}
	return nil, nil
}

func (f *FakeClubService) RevokeInvite(ctx context.Context, callerUUID, clubUUID uuid.UUID, code string) error {
	f.record("RevokeInvite")
	if f.RevokeInviteFunc != nil {
		return f.RevokeInviteFunc(ctx, callerUUID, clubUUID, code)
	}
	return nil
}

func (f *FakeClubService) GetInvitePreview(ctx context.Context, code string) (*clubservice.InvitePreview, error) {
	f.record("GetInvitePreview")
	if f.GetInvitePreviewFunc != nil {
		return f.GetInvitePreviewFunc(ctx, code)
	}
	return nil, nil
}

func (f *FakeClubService) JoinByCode(ctx context.Context, userUUID uuid.UUID, code string) error {
	f.record("JoinByCode")
	if f.JoinByCodeFunc != nil {
		return f.JoinByCodeFunc(ctx, userUUID, code)
	}
	return nil
}

func (f *FakeClubService) ListChallenges(ctx context.Context, req clubservice.ChallengeListRequest) ([]clubtypes.ChallengeSummary, error) {
	f.record("ListChallenges")
	if f.ListChallengesFunc != nil {
		return f.ListChallengesFunc(ctx, req)
	}
	return nil, nil
}

func (f *FakeClubService) GetChallengeDetail(ctx context.Context, req clubservice.ChallengeDetailRequest) (*clubtypes.ChallengeDetail, error) {
	f.record("GetChallengeDetail")
	if f.GetChallengeDetailFunc != nil {
		return f.GetChallengeDetailFunc(ctx, req)
	}
	return nil, nil
}

func (f *FakeClubService) OpenChallenge(ctx context.Context, req clubservice.ChallengeOpenRequest) (*clubtypes.ChallengeDetail, error) {
	f.record("OpenChallenge")
	if f.OpenChallengeFunc != nil {
		return f.OpenChallengeFunc(ctx, req)
	}
	return nil, nil
}

func (f *FakeClubService) RespondToChallenge(ctx context.Context, req clubservice.ChallengeRespondRequest) (*clubtypes.ChallengeDetail, error) {
	f.record("RespondToChallenge")
	if f.RespondToChallengeFunc != nil {
		return f.RespondToChallengeFunc(ctx, req)
	}
	return nil, nil
}

func (f *FakeClubService) WithdrawChallenge(ctx context.Context, req clubservice.ChallengeActionRequest) (*clubtypes.ChallengeDetail, error) {
	f.record("WithdrawChallenge")
	if f.WithdrawChallengeFunc != nil {
		return f.WithdrawChallengeFunc(ctx, req)
	}
	return nil, nil
}

func (f *FakeClubService) HideChallenge(ctx context.Context, req clubservice.ChallengeActionRequest) (*clubtypes.ChallengeDetail, error) {
	f.record("HideChallenge")
	if f.HideChallengeFunc != nil {
		return f.HideChallengeFunc(ctx, req)
	}
	return nil, nil
}

func (f *FakeClubService) LinkChallengeRound(ctx context.Context, req clubservice.ChallengeRoundLinkRequest) (*clubtypes.ChallengeDetail, error) {
	f.record("LinkChallengeRound")
	if f.LinkChallengeRoundFunc != nil {
		return f.LinkChallengeRoundFunc(ctx, req)
	}
	return nil, nil
}

func (f *FakeClubService) UnlinkChallengeRound(ctx context.Context, req clubservice.ChallengeActionRequest) (*clubtypes.ChallengeDetail, error) {
	f.record("UnlinkChallengeRound")
	if f.UnlinkChallengeRoundFunc != nil {
		return f.UnlinkChallengeRoundFunc(ctx, req)
	}
	return nil, nil
}

func (f *FakeClubService) BindChallengeMessage(ctx context.Context, req clubservice.ChallengeMessageBindingRequest) (*clubtypes.ChallengeDetail, error) {
	f.record("BindChallengeMessage")
	if f.BindChallengeMessageFunc != nil {
		return f.BindChallengeMessageFunc(ctx, req)
	}
	return nil, nil
}

func (f *FakeClubService) ExpireChallenge(ctx context.Context, req clubservice.ChallengeExpireRequest) (*clubtypes.ChallengeDetail, error) {
	f.record("ExpireChallenge")
	if f.ExpireChallengeFunc != nil {
		return f.ExpireChallengeFunc(ctx, req)
	}
	return nil, nil
}

func (f *FakeClubService) CompleteChallengeForRound(ctx context.Context, req clubservice.ChallengeRoundEventRequest) (*clubtypes.ChallengeDetail, error) {
	f.record("CompleteChallengeForRound")
	if f.CompleteChallengeForRoundFunc != nil {
		return f.CompleteChallengeForRoundFunc(ctx, req)
	}
	return nil, nil
}

func (f *FakeClubService) ResetChallengeForRound(ctx context.Context, req clubservice.ChallengeRoundEventRequest) (*clubtypes.ChallengeDetail, error) {
	f.record("ResetChallengeForRound")
	if f.ResetChallengeForRoundFunc != nil {
		return f.ResetChallengeForRoundFunc(ctx, req)
	}
	return nil, nil
}

func (f *FakeClubService) RefreshChallengesForMembers(ctx context.Context, req clubservice.ChallengeRefreshRequest) ([]clubtypes.ChallengeDetail, error) {
	f.record("RefreshChallengesForMembers")
	if f.RefreshChallengesForMembersFunc != nil {
		return f.RefreshChallengesForMembersFunc(ctx, req)
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
