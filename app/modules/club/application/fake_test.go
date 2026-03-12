package clubservice

import (
	"context"
	"time"

	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	clubdb "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ------------------------
// Fake Club Repo
// ------------------------

type FakeClubRepo struct {
	trace []string

	GetByUUIDFunc                   func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error)
	GetByDiscordGuildIDFunc         func(ctx context.Context, db bun.IDB, guildID string) (*clubdb.Club, error)
	GetClubsByDiscordGuildIDsFunc   func(ctx context.Context, db bun.IDB, guildIDs []string) ([]*clubdb.Club, error)
	UpsertFunc                      func(ctx context.Context, db bun.IDB, club *clubdb.Club) error
	UpdateNameFunc                  func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, name string) error
	CreateInviteFunc                func(ctx context.Context, db bun.IDB, invite *clubdb.ClubInvite) error
	GetInviteByCodeFunc             func(ctx context.Context, db bun.IDB, code string) (*clubdb.ClubInvite, error)
	GetInvitesByClubFunc            func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) ([]*clubdb.ClubInvite, error)
	RevokeInviteFunc                func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, code string) error
	IncrementInviteUseCountFunc     func(ctx context.Context, db bun.IDB, code string) error
	GetChallengeByUUIDFunc          func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallenge, error)
	CreateChallengeFunc             func(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error
	UpdateChallengeFunc             func(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error
	ListChallengesFunc              func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, statuses []clubtypes.ChallengeStatus) ([]*clubdb.ClubChallenge, error)
	GetOpenOutgoingChallengeFunc    func(ctx context.Context, db bun.IDB, clubUUID, challengerUserUUID uuid.UUID) (*clubdb.ClubChallenge, error)
	GetAcceptedChallengeForUserFunc func(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID) (*clubdb.ClubChallenge, error)
	GetActiveChallengeByPairFunc    func(ctx context.Context, db bun.IDB, clubUUID, userA, userB uuid.UUID) (*clubdb.ClubChallenge, error)
	ListActiveChallengesByUsersFunc func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, userUUIDs []uuid.UUID) ([]*clubdb.ClubChallenge, error)
	BindChallengeMessageFunc        func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID, guildID, channelID, messageID string) error
	CreateChallengeRoundLinkFunc    func(ctx context.Context, db bun.IDB, link *clubdb.ClubChallengeRoundLink) error
	GetActiveChallengeRoundLinkFunc func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallengeRoundLink, error)
	GetChallengeByActiveRoundFunc   func(ctx context.Context, db bun.IDB, roundID uuid.UUID) (*clubdb.ClubChallenge, error)
	UnlinkActiveChallengeRoundFunc  func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID, actorUserUUID *uuid.UUID, unlinkedAt time.Time) error
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

func (f *FakeClubRepo) GetClubsByDiscordGuildIDs(ctx context.Context, db bun.IDB, guildIDs []string) ([]*clubdb.Club, error) {
	f.record("GetClubsByDiscordGuildIDs")
	if f.GetClubsByDiscordGuildIDsFunc != nil {
		return f.GetClubsByDiscordGuildIDsFunc(ctx, db, guildIDs)
	}
	return nil, nil
}

func (f *FakeClubRepo) CreateInvite(ctx context.Context, db bun.IDB, invite *clubdb.ClubInvite) error {
	f.record("CreateInvite")
	if f.CreateInviteFunc != nil {
		return f.CreateInviteFunc(ctx, db, invite)
	}
	return nil
}

func (f *FakeClubRepo) GetInviteByCode(ctx context.Context, db bun.IDB, code string) (*clubdb.ClubInvite, error) {
	f.record("GetInviteByCode")
	if f.GetInviteByCodeFunc != nil {
		return f.GetInviteByCodeFunc(ctx, db, code)
	}
	return nil, clubdb.ErrNotFound
}

func (f *FakeClubRepo) GetInvitesByClub(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) ([]*clubdb.ClubInvite, error) {
	f.record("GetInvitesByClub")
	if f.GetInvitesByClubFunc != nil {
		return f.GetInvitesByClubFunc(ctx, db, clubUUID)
	}
	return nil, nil
}

func (f *FakeClubRepo) RevokeInvite(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, code string) error {
	f.record("RevokeInvite")
	if f.RevokeInviteFunc != nil {
		return f.RevokeInviteFunc(ctx, db, clubUUID, code)
	}
	return nil
}

func (f *FakeClubRepo) IncrementInviteUseCount(ctx context.Context, db bun.IDB, code string) error {
	f.record("IncrementInviteUseCount")
	if f.IncrementInviteUseCountFunc != nil {
		return f.IncrementInviteUseCountFunc(ctx, db, code)
	}
	return nil
}

func (f *FakeClubRepo) GetChallengeByUUID(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
	f.record("GetChallengeByUUID")
	if f.GetChallengeByUUIDFunc != nil {
		return f.GetChallengeByUUIDFunc(ctx, db, challengeUUID)
	}
	return nil, clubdb.ErrNotFound
}

func (f *FakeClubRepo) CreateChallenge(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error {
	f.record("CreateChallenge")
	if f.CreateChallengeFunc != nil {
		return f.CreateChallengeFunc(ctx, db, challenge)
	}
	return nil
}

func (f *FakeClubRepo) UpdateChallenge(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error {
	f.record("UpdateChallenge")
	if f.UpdateChallengeFunc != nil {
		return f.UpdateChallengeFunc(ctx, db, challenge)
	}
	return nil
}

func (f *FakeClubRepo) ListChallenges(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, statuses []clubtypes.ChallengeStatus) ([]*clubdb.ClubChallenge, error) {
	f.record("ListChallenges")
	if f.ListChallengesFunc != nil {
		return f.ListChallengesFunc(ctx, db, clubUUID, statuses)
	}
	return nil, nil
}

func (f *FakeClubRepo) GetOpenOutgoingChallenge(ctx context.Context, db bun.IDB, clubUUID, challengerUserUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
	f.record("GetOpenOutgoingChallenge")
	if f.GetOpenOutgoingChallengeFunc != nil {
		return f.GetOpenOutgoingChallengeFunc(ctx, db, clubUUID, challengerUserUUID)
	}
	return nil, clubdb.ErrNotFound
}

func (f *FakeClubRepo) GetAcceptedChallengeForUser(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
	f.record("GetAcceptedChallengeForUser")
	if f.GetAcceptedChallengeForUserFunc != nil {
		return f.GetAcceptedChallengeForUserFunc(ctx, db, clubUUID, userUUID)
	}
	return nil, clubdb.ErrNotFound
}

func (f *FakeClubRepo) GetActiveChallengeByPair(ctx context.Context, db bun.IDB, clubUUID, userA, userB uuid.UUID) (*clubdb.ClubChallenge, error) {
	f.record("GetActiveChallengeByPair")
	if f.GetActiveChallengeByPairFunc != nil {
		return f.GetActiveChallengeByPairFunc(ctx, db, clubUUID, userA, userB)
	}
	return nil, clubdb.ErrNotFound
}

func (f *FakeClubRepo) ListActiveChallengesByUsers(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, userUUIDs []uuid.UUID) ([]*clubdb.ClubChallenge, error) {
	f.record("ListActiveChallengesByUsers")
	if f.ListActiveChallengesByUsersFunc != nil {
		return f.ListActiveChallengesByUsersFunc(ctx, db, clubUUID, userUUIDs)
	}
	return nil, nil
}

func (f *FakeClubRepo) BindChallengeMessage(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID, guildID, channelID, messageID string) error {
	f.record("BindChallengeMessage")
	if f.BindChallengeMessageFunc != nil {
		return f.BindChallengeMessageFunc(ctx, db, challengeUUID, guildID, channelID, messageID)
	}
	return nil
}

func (f *FakeClubRepo) CreateChallengeRoundLink(ctx context.Context, db bun.IDB, link *clubdb.ClubChallengeRoundLink) error {
	f.record("CreateChallengeRoundLink")
	if f.CreateChallengeRoundLinkFunc != nil {
		return f.CreateChallengeRoundLinkFunc(ctx, db, link)
	}
	return nil
}

func (f *FakeClubRepo) GetActiveChallengeRoundLink(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallengeRoundLink, error) {
	f.record("GetActiveChallengeRoundLink")
	if f.GetActiveChallengeRoundLinkFunc != nil {
		return f.GetActiveChallengeRoundLinkFunc(ctx, db, challengeUUID)
	}
	return nil, clubdb.ErrNotFound
}

func (f *FakeClubRepo) GetChallengeByActiveRound(ctx context.Context, db bun.IDB, roundID uuid.UUID) (*clubdb.ClubChallenge, error) {
	f.record("GetChallengeByActiveRound")
	if f.GetChallengeByActiveRoundFunc != nil {
		return f.GetChallengeByActiveRoundFunc(ctx, db, roundID)
	}
	return nil, clubdb.ErrNotFound
}

func (f *FakeClubRepo) UnlinkActiveChallengeRound(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID, actorUserUUID *uuid.UUID, unlinkedAt time.Time) error {
	f.record("UnlinkActiveChallengeRound")
	if f.UnlinkActiveChallengeRoundFunc != nil {
		return f.UnlinkActiveChallengeRoundFunc(ctx, db, challengeUUID, actorUserUUID, unlinkedAt)
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

type FakeChallengeQueueService struct {
	ScheduleOpenExpiryFunc     func(ctx context.Context, challengeID uuid.UUID, expiresAt time.Time) error
	ScheduleAcceptedExpiryFunc func(ctx context.Context, challengeID uuid.UUID, expiresAt time.Time) error
	CancelChallengeJobsFunc    func(ctx context.Context, challengeID uuid.UUID) error
	HealthCheckFunc            func(ctx context.Context) error
	StartFunc                  func(ctx context.Context) error
	StopFunc                   func(ctx context.Context) error
}

func (f *FakeChallengeQueueService) ScheduleOpenExpiry(ctx context.Context, challengeID uuid.UUID, expiresAt time.Time) error {
	if f.ScheduleOpenExpiryFunc != nil {
		return f.ScheduleOpenExpiryFunc(ctx, challengeID, expiresAt)
	}
	return nil
}

func (f *FakeChallengeQueueService) ScheduleAcceptedExpiry(ctx context.Context, challengeID uuid.UUID, expiresAt time.Time) error {
	if f.ScheduleAcceptedExpiryFunc != nil {
		return f.ScheduleAcceptedExpiryFunc(ctx, challengeID, expiresAt)
	}
	return nil
}

func (f *FakeChallengeQueueService) CancelChallengeJobs(ctx context.Context, challengeID uuid.UUID) error {
	if f.CancelChallengeJobsFunc != nil {
		return f.CancelChallengeJobsFunc(ctx, challengeID)
	}
	return nil
}

func (f *FakeChallengeQueueService) HealthCheck(ctx context.Context) error {
	if f.HealthCheckFunc != nil {
		return f.HealthCheckFunc(ctx)
	}
	return nil
}

func (f *FakeChallengeQueueService) Start(ctx context.Context) error {
	if f.StartFunc != nil {
		return f.StartFunc(ctx)
	}
	return nil
}

func (f *FakeChallengeQueueService) Stop(ctx context.Context) error {
	if f.StopFunc != nil {
		return f.StopFunc(ctx)
	}
	return nil
}

var _ ChallengeQueueService = (*FakeChallengeQueueService)(nil)

type FakeChallengeTagReader struct {
	GetTagListFunc func(ctx context.Context, guildID sharedtypes.GuildID, clubUUID *string) ([]leaderboardservice.MemberTagView, error)
}

func (f *FakeChallengeTagReader) GetTagList(ctx context.Context, guildID sharedtypes.GuildID, clubUUID *string) ([]leaderboardservice.MemberTagView, error) {
	if f.GetTagListFunc != nil {
		return f.GetTagListFunc(ctx, guildID, clubUUID)
	}
	return nil, nil
}

var _ ChallengeTagReader = (*FakeChallengeTagReader)(nil)

type FakeChallengeRoundReader struct {
	GetRoundFunc func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error)
}

func (f *FakeChallengeRoundReader) GetRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
	if f.GetRoundFunc != nil {
		return f.GetRoundFunc(ctx, guildID, roundID)
	}
	return results.OperationResult[*roundtypes.Round, error]{}, nil
}

var _ ChallengeRoundReader = (*FakeChallengeRoundReader)(nil)
