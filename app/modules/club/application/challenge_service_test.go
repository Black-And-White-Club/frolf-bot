package clubservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	clubmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/club"
	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	clubdb "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

type challengeTestFixture struct {
	club                 *clubdb.Club
	challengerMembership *userdb.ClubMembership
	defenderMembership   *userdb.ClubMembership
	challenge            *clubdb.ClubChallenge
	roundID              uuid.UUID
}

func TestClubService_OpenChallenge(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) (*ClubService, ChallengeOpenRequest, func(*testing.T, *clubtypes.ChallengeDetail, error))
		wantErrMsg string
	}{
		{
			name: "opens challenge and schedules open expiry",
			setup: func(t *testing.T) (*ClubService, ChallengeOpenRequest, func(*testing.T, *clubtypes.ChallengeDetail, error)) {
				t.Helper()

				fx := newChallengeFixture()
				repo := NewFakeClubRepo()
				queue := &FakeChallengeQueueService{}
				var created *clubdb.ClubChallenge
				var scheduledID uuid.UUID
				var scheduledAt time.Time

				repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
					return fx.club, nil
				}
				repo.GetOpenOutgoingChallengeFunc = func(ctx context.Context, db bun.IDB, clubUUID, challengerUserUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return nil, clubdb.ErrNotFound
				}
				repo.GetAcceptedChallengeForUserFunc = func(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return nil, clubdb.ErrNotFound
				}
				repo.GetActiveChallengeByPairFunc = func(ctx context.Context, db bun.IDB, clubUUID, userA, userB uuid.UUID) (*clubdb.ClubChallenge, error) {
					return nil, clubdb.ErrNotFound
				}
				repo.CreateChallengeFunc = func(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error {
					created = cloneChallengeModel(challenge)
					return nil
				}
				repo.GetActiveChallengeRoundLinkFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallengeRoundLink, error) {
					return nil, clubdb.ErrNotFound
				}

				queue.ScheduleOpenExpiryFunc = func(ctx context.Context, challengeID uuid.UUID, expiresAt time.Time) error {
					scheduledID = challengeID
					scheduledAt = expiresAt
					return nil
				}

				req := ChallengeOpenRequest{
					Scope:  ChallengeScope{ClubUUID: &fx.club.UUID},
					Actor:  ChallengeActorIdentity{UserUUID: &fx.challengerMembership.UserUUID},
					Target: ChallengeActorIdentity{UserUUID: &fx.defenderMembership.UserUUID},
				}

				service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), queue, newChallengeTagReader(fx, 18, 7), nil)
				assertions := func(t *testing.T, detail *clubtypes.ChallengeDetail, err error) {
					t.Helper()
					require.NoError(t, err)
					require.NotNil(t, detail)
					require.NotNil(t, created)
					assert.Equal(t, clubtypes.ChallengeStatusOpen, detail.Status)
					assert.Equal(t, clubtypes.ChallengeStatusOpen, created.Status)
					require.NotNil(t, detail.OpenExpiresAt)
					assert.Equal(t, created.UUID, scheduledID)
					assert.WithinDuration(t, *detail.OpenExpiresAt, scheduledAt, time.Second)
					require.NotNil(t, detail.OriginalTags.Challenger)
					require.NotNil(t, detail.OriginalTags.Defender)
					assert.Equal(t, 18, *detail.OriginalTags.Challenger)
					assert.Equal(t, 7, *detail.OriginalTags.Defender)
				}
				return service, req, assertions
			},
		},
		{
			name: "rejects when challenger already has open challenge",
			setup: func(t *testing.T) (*ClubService, ChallengeOpenRequest, func(*testing.T, *clubtypes.ChallengeDetail, error)) {
				t.Helper()

				fx := newChallengeFixture()
				repo := NewFakeClubRepo()
				queue := &FakeChallengeQueueService{}
				var createCalled bool
				var scheduleCalled bool

				repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
					return fx.club, nil
				}
				repo.GetOpenOutgoingChallengeFunc = func(ctx context.Context, db bun.IDB, clubUUID, challengerUserUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return cloneChallengeModel(fx.challenge), nil
				}
				repo.CreateChallengeFunc = func(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error {
					createCalled = true
					return nil
				}
				queue.ScheduleOpenExpiryFunc = func(ctx context.Context, challengeID uuid.UUID, expiresAt time.Time) error {
					scheduleCalled = true
					return nil
				}

				req := ChallengeOpenRequest{
					Scope:  ChallengeScope{ClubUUID: &fx.club.UUID},
					Actor:  ChallengeActorIdentity{UserUUID: &fx.challengerMembership.UserUUID},
					Target: ChallengeActorIdentity{UserUUID: &fx.defenderMembership.UserUUID},
				}

				service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), queue, newChallengeTagReader(fx, 18, 7), nil)
				assertions := func(t *testing.T, detail *clubtypes.ChallengeDetail, err error) {
					t.Helper()
					require.Error(t, err)
					assert.Nil(t, detail)
					assert.ErrorContains(t, err, "you already have an open outgoing challenge")
					assert.False(t, createCalled)
					assert.False(t, scheduleCalled)
				}
				return service, req, assertions
			},
		},
		{
			name: "rejects when challenger is not chasing a better tag",
			setup: func(t *testing.T) (*ClubService, ChallengeOpenRequest, func(*testing.T, *clubtypes.ChallengeDetail, error)) {
				t.Helper()

				fx := newChallengeFixture()
				repo := NewFakeClubRepo()

				repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
					return fx.club, nil
				}
				repo.GetOpenOutgoingChallengeFunc = func(ctx context.Context, db bun.IDB, clubUUID, challengerUserUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return nil, clubdb.ErrNotFound
				}
				repo.GetAcceptedChallengeForUserFunc = func(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return nil, clubdb.ErrNotFound
				}
				repo.GetActiveChallengeByPairFunc = func(ctx context.Context, db bun.IDB, clubUUID, userA, userB uuid.UUID) (*clubdb.ClubChallenge, error) {
					return nil, clubdb.ErrNotFound
				}

				req := ChallengeOpenRequest{
					Scope:  ChallengeScope{ClubUUID: &fx.club.UUID},
					Actor:  ChallengeActorIdentity{UserUUID: &fx.challengerMembership.UserUUID},
					Target: ChallengeActorIdentity{UserUUID: &fx.defenderMembership.UserUUID},
				}

				service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), &FakeChallengeQueueService{}, newChallengeTagReader(fx, 4, 7), nil)
				assertions := func(t *testing.T, detail *clubtypes.ChallengeDetail, err error) {
					t.Helper()
					require.Error(t, err)
					assert.Nil(t, detail)
					assert.ErrorContains(t, err, "you can only challenge a better tag than your own")
				}
				return service, req, assertions
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, req, assertions := tt.setup(t)
			detail, err := service.OpenChallenge(context.Background(), req)
			assertions(t, detail, err)
		})
	}
}

func TestClubService_RespondToChallenge(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		setup      func(t *testing.T, response string) (*ClubService, ChallengeRespondRequest, func(*testing.T, *clubtypes.ChallengeDetail, error))
		wantErrMsg string
	}{
		{
			name:     "accepts challenge and reschedules expiry",
			response: ChallengeResponseAccept,
			setup: func(t *testing.T, response string) (*ClubService, ChallengeRespondRequest, func(*testing.T, *clubtypes.ChallengeDetail, error)) {
				t.Helper()

				fx := newChallengeFixture()
				repo := NewFakeClubRepo()
				queue := &FakeChallengeQueueService{}
				challenge := cloneChallengeModel(fx.challenge)
				var updated *clubdb.ClubChallenge
				var cancelID uuid.UUID
				var acceptedScheduleID uuid.UUID

				repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
					return fx.club, nil
				}
				repo.GetChallengeByUUIDFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return challenge, nil
				}
				repo.UpdateChallengeFunc = func(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error {
					updated = cloneChallengeModel(challenge)
					challenge = cloneChallengeModel(challenge)
					return nil
				}
				repo.GetActiveChallengeRoundLinkFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallengeRoundLink, error) {
					return nil, clubdb.ErrNotFound
				}

				queue.CancelChallengeJobsFunc = func(ctx context.Context, challengeID uuid.UUID) error {
					cancelID = challengeID
					return nil
				}
				queue.ScheduleAcceptedExpiryFunc = func(ctx context.Context, challengeID uuid.UUID, expiresAt time.Time) error {
					acceptedScheduleID = challengeID
					return nil
				}

				req := ChallengeRespondRequest{
					Scope:       ChallengeScope{ClubUUID: &fx.club.UUID},
					Actor:       ChallengeActorIdentity{UserUUID: &fx.defenderMembership.UserUUID},
					ChallengeID: fx.challenge.UUID,
					Response:    response,
				}

				service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), queue, newChallengeTagReader(fx, 18, 7), nil)
				assertions := func(t *testing.T, detail *clubtypes.ChallengeDetail, err error) {
					t.Helper()
					require.NoError(t, err)
					require.NotNil(t, detail)
					require.NotNil(t, updated)
					assert.Equal(t, clubtypes.ChallengeStatusAccepted, detail.Status)
					assert.Equal(t, clubtypes.ChallengeStatusAccepted, updated.Status)
					assert.Equal(t, fx.challenge.UUID, cancelID)
					assert.Equal(t, fx.challenge.UUID, acceptedScheduleID)
					assert.Nil(t, updated.OpenExpiresAt)
					require.NotNil(t, updated.AcceptedAt)
					require.NotNil(t, updated.AcceptedExpiresAt)
				}
				return service, req, assertions
			},
		},
		{
			name:     "declines challenge and only cancels jobs",
			response: ChallengeResponseDecline,
			setup: func(t *testing.T, response string) (*ClubService, ChallengeRespondRequest, func(*testing.T, *clubtypes.ChallengeDetail, error)) {
				t.Helper()

				fx := newChallengeFixture()
				repo := NewFakeClubRepo()
				queue := &FakeChallengeQueueService{}
				challenge := cloneChallengeModel(fx.challenge)
				var updated *clubdb.ClubChallenge
				var cancelID uuid.UUID
				var acceptedScheduleCalls int

				repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
					return fx.club, nil
				}
				repo.GetChallengeByUUIDFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return challenge, nil
				}
				repo.UpdateChallengeFunc = func(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error {
					updated = cloneChallengeModel(challenge)
					return nil
				}
				repo.GetActiveChallengeRoundLinkFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallengeRoundLink, error) {
					return nil, clubdb.ErrNotFound
				}

				queue.CancelChallengeJobsFunc = func(ctx context.Context, challengeID uuid.UUID) error {
					cancelID = challengeID
					return nil
				}
				queue.ScheduleAcceptedExpiryFunc = func(ctx context.Context, challengeID uuid.UUID, expiresAt time.Time) error {
					acceptedScheduleCalls++
					return nil
				}

				req := ChallengeRespondRequest{
					Scope:       ChallengeScope{ClubUUID: &fx.club.UUID},
					Actor:       ChallengeActorIdentity{UserUUID: &fx.defenderMembership.UserUUID},
					ChallengeID: fx.challenge.UUID,
					Response:    response,
				}

				service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), queue, newChallengeTagReader(fx, 18, 7), nil)
				assertions := func(t *testing.T, detail *clubtypes.ChallengeDetail, err error) {
					t.Helper()
					require.NoError(t, err)
					require.NotNil(t, detail)
					require.NotNil(t, updated)
					assert.Equal(t, clubtypes.ChallengeStatusDeclined, detail.Status)
					assert.Equal(t, clubtypes.ChallengeStatusDeclined, updated.Status)
					assert.Equal(t, fx.challenge.UUID, cancelID)
					assert.Zero(t, acceptedScheduleCalls)
					assert.Nil(t, updated.AcceptedAt)
					assert.Nil(t, updated.AcceptedExpiresAt)
				}
				return service, req, assertions
			},
		},
		{
			name:     "rejects invalid response",
			response: "maybe",
			setup: func(t *testing.T, response string) (*ClubService, ChallengeRespondRequest, func(*testing.T, *clubtypes.ChallengeDetail, error)) {
				t.Helper()

				fx := newChallengeFixture()
				req := ChallengeRespondRequest{
					Scope:       ChallengeScope{ClubUUID: &fx.club.UUID},
					Actor:       ChallengeActorIdentity{UserUUID: &fx.defenderMembership.UserUUID},
					ChallengeID: fx.challenge.UUID,
					Response:    response,
				}

				service := newChallengeTestService(t, NewFakeClubRepo(), newChallengeUserRepo(fx), &FakeChallengeQueueService{}, newChallengeTagReader(fx, 18, 7), nil)
				assertions := func(t *testing.T, detail *clubtypes.ChallengeDetail, err error) {
					t.Helper()
					require.Error(t, err)
					assert.Nil(t, detail)
					assert.ErrorContains(t, err, "response must be accept or decline")
				}
				return service, req, assertions
			},
		},
		{
			name:     "rejects accept when defender already has another accepted challenge",
			response: ChallengeResponseAccept,
			setup: func(t *testing.T, response string) (*ClubService, ChallengeRespondRequest, func(*testing.T, *clubtypes.ChallengeDetail, error)) {
				t.Helper()

				fx := newChallengeFixture()
				repo := NewFakeClubRepo()
				challenge := cloneChallengeModel(fx.challenge)
				var updateCalled bool

				repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
					return fx.club, nil
				}
				repo.GetChallengeByUUIDFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return challenge, nil
				}
				repo.GetAcceptedChallengeForUserFunc = func(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
					if userUUID == fx.defenderMembership.UserUUID {
						return &clubdb.ClubChallenge{UUID: uuid.New(), Status: clubtypes.ChallengeStatusAccepted}, nil
					}
					return nil, clubdb.ErrNotFound
				}
				repo.UpdateChallengeFunc = func(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error {
					updateCalled = true
					return nil
				}

				req := ChallengeRespondRequest{
					Scope:       ChallengeScope{ClubUUID: &fx.club.UUID},
					Actor:       ChallengeActorIdentity{UserUUID: &fx.defenderMembership.UserUUID},
					ChallengeID: fx.challenge.UUID,
					Response:    response,
				}

				service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), &FakeChallengeQueueService{}, newChallengeTagReader(fx, 18, 7), nil)
				assertions := func(t *testing.T, detail *clubtypes.ChallengeDetail, err error) {
					t.Helper()
					require.Error(t, err)
					assert.Nil(t, detail)
					assert.ErrorContains(t, err, "you already have an accepted challenge")
					assert.False(t, updateCalled)
				}
				return service, req, assertions
			},
		},
		{
			name:     "rejects accept when challenger already has another accepted challenge",
			response: ChallengeResponseAccept,
			setup: func(t *testing.T, response string) (*ClubService, ChallengeRespondRequest, func(*testing.T, *clubtypes.ChallengeDetail, error)) {
				t.Helper()

				fx := newChallengeFixture()
				repo := NewFakeClubRepo()
				challenge := cloneChallengeModel(fx.challenge)
				var updateCalled bool

				repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
					return fx.club, nil
				}
				repo.GetChallengeByUUIDFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return challenge, nil
				}
				repo.GetAcceptedChallengeForUserFunc = func(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
					if userUUID == fx.challengerMembership.UserUUID {
						return &clubdb.ClubChallenge{UUID: uuid.New(), Status: clubtypes.ChallengeStatusAccepted}, nil
					}
					return nil, clubdb.ErrNotFound
				}
				repo.UpdateChallengeFunc = func(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error {
					updateCalled = true
					return nil
				}

				req := ChallengeRespondRequest{
					Scope:       ChallengeScope{ClubUUID: &fx.club.UUID},
					Actor:       ChallengeActorIdentity{UserUUID: &fx.defenderMembership.UserUUID},
					ChallengeID: fx.challenge.UUID,
					Response:    response,
				}

				service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), &FakeChallengeQueueService{}, newChallengeTagReader(fx, 18, 7), nil)
				assertions := func(t *testing.T, detail *clubtypes.ChallengeDetail, err error) {
					t.Helper()
					require.Error(t, err)
					assert.Nil(t, detail)
					assert.ErrorContains(t, err, "that player already has an accepted challenge")
					assert.False(t, updateCalled)
				}
				return service, req, assertions
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, req, assertions := tt.setup(t, tt.response)
			detail, err := service.RespondToChallenge(context.Background(), req)
			assertions(t, detail, err)
		})
	}
}

func TestClubService_LinkChallengeRound(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) (*ClubService, ChallengeRoundLinkRequest, func(*testing.T, *clubtypes.ChallengeDetail, error))
	}{
		{
			name: "links accepted challenge to round and clears accepted expiry",
			setup: func(t *testing.T) (*ClubService, ChallengeRoundLinkRequest, func(*testing.T, *clubtypes.ChallengeDetail, error)) {
				t.Helper()

				fx := newChallengeFixture()
				repo := NewFakeClubRepo()
				queue := &FakeChallengeQueueService{}
				challenge := cloneChallengeModel(fx.challenge)
				now := time.Now().UTC()
				challenge.Status = clubtypes.ChallengeStatusAccepted
				challenge.AcceptedAt = &now
				challenge.AcceptedExpiresAt = ptrTime(now.Add(24 * time.Hour))
				var activeLink *clubdb.ClubChallengeRoundLink
				var updated *clubdb.ClubChallenge
				var cancelledID uuid.UUID

				repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
					return fx.club, nil
				}
				repo.GetChallengeByUUIDFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return challenge, nil
				}
				repo.GetChallengeByActiveRoundFunc = func(ctx context.Context, db bun.IDB, roundID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return nil, clubdb.ErrNotFound
				}
				repo.GetActiveChallengeRoundLinkFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallengeRoundLink, error) {
					if activeLink == nil {
						return nil, clubdb.ErrNotFound
					}
					return activeLink, nil
				}
				repo.CreateChallengeRoundLinkFunc = func(ctx context.Context, db bun.IDB, link *clubdb.ClubChallengeRoundLink) error {
					activeLink = cloneChallengeRoundLink(link)
					return nil
				}
				repo.UpdateChallengeFunc = func(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error {
					updated = cloneChallengeModel(challenge)
					return nil
				}
				queue.CancelChallengeJobsFunc = func(ctx context.Context, challengeID uuid.UUID) error {
					cancelledID = challengeID
					return nil
				}

				roundReader := &FakeChallengeRoundReader{
					GetRoundFunc: func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
						return results.SuccessResult[*roundtypes.Round, error](newChallengeRound(fx)), nil
					},
				}

				req := ChallengeRoundLinkRequest{
					Scope:       ChallengeScope{ClubUUID: &fx.club.UUID},
					Actor:       ChallengeActorIdentity{UserUUID: &fx.challengerMembership.UserUUID},
					ChallengeID: fx.challenge.UUID,
					RoundID:     fx.roundID,
				}

				service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), queue, newChallengeTagReader(fx, 18, 7), roundReader)
				assertions := func(t *testing.T, detail *clubtypes.ChallengeDetail, err error) {
					t.Helper()
					require.NoError(t, err)
					require.NotNil(t, detail)
					require.NotNil(t, activeLink)
					require.NotNil(t, updated)
					assert.Equal(t, fx.challenge.UUID, cancelledID)
					require.NotNil(t, detail.LinkedRound)
					assert.Equal(t, fx.roundID.String(), detail.LinkedRound.RoundID)
					assert.Nil(t, updated.AcceptedExpiresAt)
				}
				return service, req, assertions
			},
		},
		{
			name: "rejects when round is already linked to another challenge",
			setup: func(t *testing.T) (*ClubService, ChallengeRoundLinkRequest, func(*testing.T, *clubtypes.ChallengeDetail, error)) {
				t.Helper()

				fx := newChallengeFixture()
				repo := NewFakeClubRepo()
				challenge := cloneChallengeModel(fx.challenge)
				now := time.Now().UTC()
				challenge.Status = clubtypes.ChallengeStatusAccepted
				challenge.AcceptedAt = &now
				challenge.AcceptedExpiresAt = ptrTime(now.Add(24 * time.Hour))
				var createLinkCalled bool

				repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
					return fx.club, nil
				}
				repo.GetChallengeByUUIDFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return challenge, nil
				}
				repo.GetChallengeByActiveRoundFunc = func(ctx context.Context, db bun.IDB, roundID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return &clubdb.ClubChallenge{UUID: uuid.New()}, nil
				}
				repo.CreateChallengeRoundLinkFunc = func(ctx context.Context, db bun.IDB, link *clubdb.ClubChallengeRoundLink) error {
					createLinkCalled = true
					return nil
				}

				roundReader := &FakeChallengeRoundReader{
					GetRoundFunc: func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
						return results.SuccessResult[*roundtypes.Round, error](newChallengeRound(fx)), nil
					},
				}

				req := ChallengeRoundLinkRequest{
					Scope:       ChallengeScope{ClubUUID: &fx.club.UUID},
					Actor:       ChallengeActorIdentity{UserUUID: &fx.challengerMembership.UserUUID},
					ChallengeID: fx.challenge.UUID,
					RoundID:     fx.roundID,
				}

				service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), &FakeChallengeQueueService{}, newChallengeTagReader(fx, 18, 7), roundReader)
				assertions := func(t *testing.T, detail *clubtypes.ChallengeDetail, err error) {
					t.Helper()
					require.Error(t, err)
					assert.Nil(t, detail)
					assert.ErrorContains(t, err, "that round is already linked to a challenge")
					assert.False(t, createLinkCalled)
				}
				return service, req, assertions
			},
		},
		{
			name: "treats duplicate link to the same round as success",
			setup: func(t *testing.T) (*ClubService, ChallengeRoundLinkRequest, func(*testing.T, *clubtypes.ChallengeDetail, error)) {
				t.Helper()

				fx := newChallengeFixture()
				repo := NewFakeClubRepo()
				queue := &FakeChallengeQueueService{}
				challenge := cloneChallengeModel(fx.challenge)
				now := time.Now().UTC()
				challenge.Status = clubtypes.ChallengeStatusAccepted
				challenge.AcceptedAt = &now
				challenge.AcceptedExpiresAt = ptrTime(now.Add(24 * time.Hour))
				existingLink := &clubdb.ClubChallengeRoundLink{
					UUID:             uuid.New(),
					ChallengeUUID:    fx.challenge.UUID,
					RoundID:          fx.roundID,
					LinkedByUserUUID: &fx.challengerMembership.UserUUID,
					LinkedAt:         now,
				}
				var createCalled bool
				var updateCalled bool
				var cancelCalls int

				repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
					return fx.club, nil
				}
				repo.GetChallengeByUUIDFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return challenge, nil
				}
				repo.GetActiveChallengeRoundLinkFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallengeRoundLink, error) {
					return cloneChallengeRoundLink(existingLink), nil
				}
				repo.GetChallengeByActiveRoundFunc = func(ctx context.Context, db bun.IDB, roundID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return challenge, nil
				}
				repo.CreateChallengeRoundLinkFunc = func(ctx context.Context, db bun.IDB, link *clubdb.ClubChallengeRoundLink) error {
					createCalled = true
					return nil
				}
				repo.UpdateChallengeFunc = func(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error {
					updateCalled = true
					return nil
				}
				queue.CancelChallengeJobsFunc = func(ctx context.Context, challengeID uuid.UUID) error {
					cancelCalls++
					return nil
				}

				roundReader := &FakeChallengeRoundReader{
					GetRoundFunc: func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
						return results.SuccessResult[*roundtypes.Round, error](newChallengeRound(fx)), nil
					},
				}

				req := ChallengeRoundLinkRequest{
					Scope:       ChallengeScope{ClubUUID: &fx.club.UUID},
					Actor:       ChallengeActorIdentity{UserUUID: &fx.challengerMembership.UserUUID},
					ChallengeID: fx.challenge.UUID,
					RoundID:     fx.roundID,
				}

				service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), queue, newChallengeTagReader(fx, 18, 7), roundReader)
				assertions := func(t *testing.T, detail *clubtypes.ChallengeDetail, err error) {
					t.Helper()
					require.NoError(t, err)
					require.NotNil(t, detail)
					require.NotNil(t, detail.LinkedRound)
					assert.Equal(t, fx.roundID.String(), detail.LinkedRound.RoundID)
					assert.False(t, createCalled)
					assert.False(t, updateCalled)
					assert.Equal(t, 1, cancelCalls)
				}
				return service, req, assertions
			},
		},
		{
			name: "rejects round that does not include both challenge participants",
			setup: func(t *testing.T) (*ClubService, ChallengeRoundLinkRequest, func(*testing.T, *clubtypes.ChallengeDetail, error)) {
				t.Helper()

				fx := newChallengeFixture()
				repo := NewFakeClubRepo()
				challenge := cloneChallengeModel(fx.challenge)
				now := time.Now().UTC()
				challenge.Status = clubtypes.ChallengeStatusAccepted
				challenge.AcceptedAt = &now
				challenge.AcceptedExpiresAt = ptrTime(now.Add(24 * time.Hour))
				var createCalled bool
				var updateCalled bool

				repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
					return fx.club, nil
				}
				repo.GetChallengeByUUIDFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return challenge, nil
				}
				repo.GetChallengeByActiveRoundFunc = func(ctx context.Context, db bun.IDB, roundID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return nil, clubdb.ErrNotFound
				}
				repo.CreateChallengeRoundLinkFunc = func(ctx context.Context, db bun.IDB, link *clubdb.ClubChallengeRoundLink) error {
					createCalled = true
					return nil
				}
				repo.UpdateChallengeFunc = func(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error {
					updateCalled = true
					return nil
				}

				roundReader := &FakeChallengeRoundReader{
					GetRoundFunc: func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (results.OperationResult[*roundtypes.Round, error], error) {
						return results.SuccessResult[*roundtypes.Round, error](newChallengeRound(
							fx,
							sharedtypes.DiscordID(valueOrEmpty(fx.challengerMembership.ExternalID)),
							sharedtypes.DiscordID("spectator-ext"),
						)), nil
					},
				}

				req := ChallengeRoundLinkRequest{
					Scope:       ChallengeScope{ClubUUID: &fx.club.UUID},
					Actor:       ChallengeActorIdentity{UserUUID: &fx.challengerMembership.UserUUID},
					ChallengeID: fx.challenge.UUID,
					RoundID:     fx.roundID,
				}

				service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), &FakeChallengeQueueService{}, newChallengeTagReader(fx, 18, 7), roundReader)
				assertions := func(t *testing.T, detail *clubtypes.ChallengeDetail, err error) {
					t.Helper()
					require.Error(t, err)
					assert.Nil(t, detail)
					assert.ErrorContains(t, err, "linked round must include both challenge participants")
					assert.False(t, createCalled)
					assert.False(t, updateCalled)
				}
				return service, req, assertions
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, req, assertions := tt.setup(t)
			detail, err := service.LinkChallengeRound(context.Background(), req)
			assertions(t, detail, err)
		})
	}
}

func TestClubService_WithdrawChallenge_RechecksChallengeBeforeUpdating(t *testing.T) {
	fx := newChallengeFixture()
	repo := NewFakeClubRepo()
	queue := &FakeChallengeQueueService{}
	openChallenge := cloneChallengeModel(fx.challenge)
	openChallenge.OpenExpiresAt = ptrTime(time.Now().UTC().Add(24 * time.Hour))

	acceptedChallenge := cloneChallengeModel(fx.challenge)
	acceptedAt := time.Now().UTC()
	acceptedChallenge.Status = clubtypes.ChallengeStatusAccepted
	acceptedChallenge.OpenExpiresAt = nil
	acceptedChallenge.AcceptedAt = &acceptedAt
	acceptedChallenge.AcceptedExpiresAt = ptrTime(acceptedAt.Add(24 * time.Hour))

	var getChallengeCalls int
	var updated *clubdb.ClubChallenge
	var cancelledID uuid.UUID

	repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
		return fx.club, nil
	}
	repo.GetChallengeByUUIDFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
		getChallengeCalls++
		if getChallengeCalls == 1 {
			return cloneChallengeModel(openChallenge), nil
		}
		return cloneChallengeModel(acceptedChallenge), nil
	}
	repo.UpdateChallengeFunc = func(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error {
		updated = cloneChallengeModel(challenge)
		return nil
	}
	repo.GetActiveChallengeRoundLinkFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallengeRoundLink, error) {
		return nil, clubdb.ErrNotFound
	}
	queue.CancelChallengeJobsFunc = func(ctx context.Context, challengeID uuid.UUID) error {
		cancelledID = challengeID
		return nil
	}

	service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), queue, newChallengeTagReader(fx, 18, 7), nil)
	detail, err := service.WithdrawChallenge(context.Background(), ChallengeActionRequest{
		Scope:       ChallengeScope{ClubUUID: &fx.club.UUID},
		Actor:       ChallengeActorIdentity{UserUUID: &fx.challengerMembership.UserUUID},
		ChallengeID: fx.challenge.UUID,
	})

	require.NoError(t, err)
	require.NotNil(t, detail)
	require.NotNil(t, updated)
	assert.Equal(t, 2, getChallengeCalls)
	assert.Equal(t, clubtypes.ChallengeStatusWithdrawn, updated.Status)
	assert.Nil(t, updated.OpenExpiresAt)
	assert.Nil(t, updated.AcceptedExpiresAt)
	require.NotNil(t, updated.AcceptedAt)
	assert.WithinDuration(t, acceptedAt, *updated.AcceptedAt, time.Second)
	assert.Equal(t, fx.challenge.UUID, cancelledID)
	require.NotNil(t, detail.AcceptedAt)
	assert.WithinDuration(t, acceptedAt, *detail.AcceptedAt, time.Second)
}

func TestClubService_ExpireChallenge(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) (*ClubService, ChallengeExpireRequest, func(*testing.T, *clubtypes.ChallengeDetail, error))
	}{
		{
			name: "expires stale open challenge",
			setup: func(t *testing.T) (*ClubService, ChallengeExpireRequest, func(*testing.T, *clubtypes.ChallengeDetail, error)) {
				t.Helper()

				fx := newChallengeFixture()
				repo := NewFakeClubRepo()
				challenge := cloneChallengeModel(fx.challenge)
				challenge.OpenExpiresAt = ptrTime(time.Now().UTC().Add(-1 * time.Minute))
				var updated *clubdb.ClubChallenge

				repo.GetChallengeByUUIDFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return challenge, nil
				}
				repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
					return fx.club, nil
				}
				repo.UpdateChallengeFunc = func(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error {
					updated = cloneChallengeModel(challenge)
					return nil
				}
				repo.GetActiveChallengeRoundLinkFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallengeRoundLink, error) {
					return nil, clubdb.ErrNotFound
				}

				req := ChallengeExpireRequest{ChallengeID: fx.challenge.UUID, Reason: "open_expired"}
				service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), &FakeChallengeQueueService{}, newChallengeTagReader(fx, 18, 7), nil)
				assertions := func(t *testing.T, detail *clubtypes.ChallengeDetail, err error) {
					t.Helper()
					require.NoError(t, err)
					require.NotNil(t, detail)
					require.NotNil(t, updated)
					assert.Equal(t, clubtypes.ChallengeStatusExpired, detail.Status)
					assert.Equal(t, clubtypes.ChallengeStatusExpired, updated.Status)
					assert.Nil(t, updated.OpenExpiresAt)
				}
				return service, req, assertions
			},
		},
		{
			name: "does not expire accepted challenge with active round link",
			setup: func(t *testing.T) (*ClubService, ChallengeExpireRequest, func(*testing.T, *clubtypes.ChallengeDetail, error)) {
				t.Helper()

				fx := newChallengeFixture()
				repo := NewFakeClubRepo()
				challenge := cloneChallengeModel(fx.challenge)
				now := time.Now().UTC().Add(-1 * time.Minute)
				challenge.Status = clubtypes.ChallengeStatusAccepted
				challenge.AcceptedAt = &now
				challenge.AcceptedExpiresAt = &now
				var updateCalls int

				repo.GetChallengeByUUIDFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return challenge, nil
				}
				repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
					return fx.club, nil
				}
				repo.GetActiveChallengeRoundLinkFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallengeRoundLink, error) {
					return &clubdb.ClubChallengeRoundLink{UUID: uuid.New(), ChallengeUUID: fx.challenge.UUID, RoundID: fx.roundID, LinkedAt: time.Now().UTC()}, nil
				}
				repo.UpdateChallengeFunc = func(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error {
					updateCalls++
					return nil
				}

				req := ChallengeExpireRequest{ChallengeID: fx.challenge.UUID, Reason: "accepted_expired"}
				service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), &FakeChallengeQueueService{}, newChallengeTagReader(fx, 18, 7), nil)
				assertions := func(t *testing.T, detail *clubtypes.ChallengeDetail, err error) {
					t.Helper()
					require.NoError(t, err)
					assert.Nil(t, detail)
					assert.Zero(t, updateCalls)
				}
				return service, req, assertions
			},
		},
		{
			name: "expires accepted challenge without round link",
			setup: func(t *testing.T) (*ClubService, ChallengeExpireRequest, func(*testing.T, *clubtypes.ChallengeDetail, error)) {
				t.Helper()

				fx := newChallengeFixture()
				repo := NewFakeClubRepo()
				challenge := cloneChallengeModel(fx.challenge)
				now := time.Now().UTC().Add(-1 * time.Minute)
				challenge.Status = clubtypes.ChallengeStatusAccepted
				challenge.AcceptedAt = &now
				challenge.AcceptedExpiresAt = &now
				var updated *clubdb.ClubChallenge

				repo.GetChallengeByUUIDFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
					return challenge, nil
				}
				repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
					return fx.club, nil
				}
				repo.GetActiveChallengeRoundLinkFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallengeRoundLink, error) {
					return nil, clubdb.ErrNotFound
				}
				repo.UpdateChallengeFunc = func(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error {
					updated = cloneChallengeModel(challenge)
					return nil
				}

				req := ChallengeExpireRequest{ChallengeID: fx.challenge.UUID, Reason: "accepted_expired"}
				service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), &FakeChallengeQueueService{}, newChallengeTagReader(fx, 18, 7), nil)
				assertions := func(t *testing.T, detail *clubtypes.ChallengeDetail, err error) {
					t.Helper()
					require.NoError(t, err)
					require.NotNil(t, detail)
					require.NotNil(t, updated)
					assert.Equal(t, clubtypes.ChallengeStatusExpired, updated.Status)
					assert.Nil(t, updated.AcceptedExpiresAt)
				}
				return service, req, assertions
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, req, assertions := tt.setup(t)
			detail, err := service.ExpireChallenge(context.Background(), req)
			assertions(t, detail, err)
		})
	}
}

func TestClubService_OpenChallenge_PropagatesScheduleFailure(t *testing.T) {
	fx := newChallengeFixture()
	repo := NewFakeClubRepo()
	queue := &FakeChallengeQueueService{}
	scheduleErr := errors.New("queue unavailable")
	var scheduleCalls int

	repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
		return fx.club, nil
	}
	repo.GetOpenOutgoingChallengeFunc = func(ctx context.Context, db bun.IDB, clubUUID, challengerUserUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
		return nil, clubdb.ErrNotFound
	}
	repo.GetAcceptedChallengeForUserFunc = func(ctx context.Context, db bun.IDB, clubUUID, userUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
		return nil, clubdb.ErrNotFound
	}
	repo.GetActiveChallengeByPairFunc = func(ctx context.Context, db bun.IDB, clubUUID, userA, userB uuid.UUID) (*clubdb.ClubChallenge, error) {
		return nil, clubdb.ErrNotFound
	}
	repo.CreateChallengeFunc = func(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error {
		return nil
	}
	repo.GetActiveChallengeRoundLinkFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallengeRoundLink, error) {
		return nil, clubdb.ErrNotFound
	}
	queue.ScheduleOpenExpiryFunc = func(ctx context.Context, challengeID uuid.UUID, expiresAt time.Time) error {
		scheduleCalls++
		return scheduleErr
	}

	service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), queue, newChallengeTagReader(fx, 18, 7), nil)
	detail, err := service.OpenChallenge(context.Background(), ChallengeOpenRequest{
		Scope:  ChallengeScope{ClubUUID: &fx.club.UUID},
		Actor:  ChallengeActorIdentity{UserUUID: &fx.challengerMembership.UserUUID},
		Target: ChallengeActorIdentity{UserUUID: &fx.defenderMembership.UserUUID},
	})

	require.Error(t, err)
	assert.Nil(t, detail)
	assert.ErrorContains(t, err, "schedule open challenge expiry")
	assert.ErrorIs(t, err, scheduleErr)
	assert.Equal(t, 1, scheduleCalls)
}

func TestClubService_UnlinkChallengeRound_PropagatesScheduleFailure(t *testing.T) {
	fx := newChallengeFixture()
	repo := NewFakeClubRepo()
	queue := &FakeChallengeQueueService{}
	scheduleErr := errors.New("queue unavailable")
	now := time.Now().UTC()
	challenge := cloneChallengeModel(fx.challenge)
	challenge.Status = clubtypes.ChallengeStatusAccepted
	challenge.OpenExpiresAt = nil
	challenge.AcceptedAt = &now
	challenge.AcceptedExpiresAt = ptrTime(now.Add(24 * time.Hour))
	activeLink := &clubdb.ClubChallengeRoundLink{
		UUID:             uuid.New(),
		ChallengeUUID:    challenge.UUID,
		RoundID:          fx.roundID,
		LinkedByUserUUID: &fx.challengerMembership.UserUUID,
		LinkedAt:         now,
	}
	var unlinkCalls int
	var scheduleCalls int

	repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
		return fx.club, nil
	}
	repo.GetChallengeByUUIDFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
		return cloneChallengeModel(challenge), nil
	}
	repo.GetActiveChallengeRoundLinkFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallengeRoundLink, error) {
		if activeLink == nil || activeLink.UnlinkedAt != nil {
			return nil, clubdb.ErrNotFound
		}
		return cloneChallengeRoundLink(activeLink), nil
	}
	repo.UnlinkActiveChallengeRoundFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID, actorUserUUID *uuid.UUID, unlinkedAt time.Time) error {
		unlinkCalls++
		activeLink.UnlinkedAt = ptrTime(unlinkedAt)
		activeLink.UnlinkedByUserUUID = cloneUUID(actorUserUUID)
		return nil
	}
	repo.UpdateChallengeFunc = func(ctx context.Context, db bun.IDB, next *clubdb.ClubChallenge) error {
		challenge = cloneChallengeModel(next)
		return nil
	}
	queue.ScheduleAcceptedExpiryFunc = func(ctx context.Context, challengeID uuid.UUID, expiresAt time.Time) error {
		scheduleCalls++
		return scheduleErr
	}

	service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), queue, newChallengeTagReader(fx, 18, 7), nil)
	detail, err := service.UnlinkChallengeRound(context.Background(), ChallengeActionRequest{
		Scope:       ChallengeScope{ClubUUID: &fx.club.UUID},
		Actor:       ChallengeActorIdentity{UserUUID: &fx.challengerMembership.UserUUID},
		ChallengeID: fx.challenge.UUID,
	})

	require.Error(t, err)
	assert.Nil(t, detail)
	assert.ErrorContains(t, err, "schedule accepted challenge expiry")
	assert.ErrorIs(t, err, scheduleErr)
	assert.Equal(t, 1, unlinkCalls)
	assert.Equal(t, 1, scheduleCalls)
}

func TestClubService_ExpireChallenge_RechecksChallengeAfterLock(t *testing.T) {
	fx := newChallengeFixture()
	repo := NewFakeClubRepo()
	openChallenge := cloneChallengeModel(fx.challenge)
	openChallenge.OpenExpiresAt = ptrTime(time.Now().UTC().Add(-1 * time.Minute))

	acceptedAt := time.Now().UTC().Add(-2 * time.Minute)
	acceptedChallenge := cloneChallengeModel(fx.challenge)
	acceptedChallenge.Status = clubtypes.ChallengeStatusAccepted
	acceptedChallenge.OpenExpiresAt = nil
	acceptedChallenge.AcceptedAt = &acceptedAt
	acceptedChallenge.AcceptedExpiresAt = ptrTime(acceptedAt.Add(24 * time.Hour))

	var getChallengeCalls int
	var updateCalls int

	repo.GetChallengeByUUIDFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallenge, error) {
		getChallengeCalls++
		if getChallengeCalls == 1 {
			return cloneChallengeModel(openChallenge), nil
		}
		return cloneChallengeModel(acceptedChallenge), nil
	}
	repo.GetActiveChallengeRoundLinkFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallengeRoundLink, error) {
		return &clubdb.ClubChallengeRoundLink{
			UUID:          uuid.New(),
			ChallengeUUID: acceptedChallenge.UUID,
			RoundID:       fx.roundID,
			LinkedAt:      time.Now().UTC(),
		}, nil
	}
	repo.UpdateChallengeFunc = func(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error {
		updateCalls++
		return nil
	}

	service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), &FakeChallengeQueueService{}, newChallengeTagReader(fx, 18, 7), nil)
	detail, err := service.ExpireChallenge(context.Background(), ChallengeExpireRequest{
		ChallengeID: fx.challenge.UUID,
		Reason:      "open_expired",
	})

	require.NoError(t, err)
	assert.Nil(t, detail)
	assert.Equal(t, 2, getChallengeCalls)
	assert.Zero(t, updateCalls)
}

func TestClubService_CompleteChallengeForRound_RechecksActiveRoundBeforeMutating(t *testing.T) {
	fx := newChallengeFixture()
	repo := NewFakeClubRepo()
	now := time.Now().UTC()
	challenge := cloneChallengeModel(fx.challenge)
	challenge.Status = clubtypes.ChallengeStatusAccepted
	challenge.OpenExpiresAt = nil
	challenge.AcceptedAt = &now
	challenge.AcceptedExpiresAt = ptrTime(now.Add(24 * time.Hour))
	var getRoundCalls int
	var updateCalls int

	repo.GetChallengeByActiveRoundFunc = func(ctx context.Context, db bun.IDB, roundID uuid.UUID) (*clubdb.ClubChallenge, error) {
		getRoundCalls++
		if getRoundCalls == 1 {
			return cloneChallengeModel(challenge), nil
		}
		return nil, clubdb.ErrNotFound
	}
	repo.UpdateChallengeFunc = func(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error {
		updateCalls++
		return nil
	}

	service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), &FakeChallengeQueueService{}, newChallengeTagReader(fx, 18, 7), nil)
	detail, err := service.CompleteChallengeForRound(context.Background(), ChallengeRoundEventRequest{RoundID: fx.roundID})

	require.NoError(t, err)
	assert.Nil(t, detail)
	assert.Equal(t, 2, getRoundCalls)
	assert.Zero(t, updateCalls)
}

func TestClubService_ResetChallengeForRound_RechecksActiveRoundBeforeMutating(t *testing.T) {
	fx := newChallengeFixture()
	repo := NewFakeClubRepo()
	queue := &FakeChallengeQueueService{}
	now := time.Now().UTC()
	challenge := cloneChallengeModel(fx.challenge)
	challenge.Status = clubtypes.ChallengeStatusAccepted
	challenge.OpenExpiresAt = nil
	challenge.AcceptedAt = &now
	challenge.AcceptedExpiresAt = ptrTime(now.Add(24 * time.Hour))
	var getRoundCalls int
	var unlinkCalls int
	var updateCalls int
	var scheduleCalls int

	repo.GetChallengeByActiveRoundFunc = func(ctx context.Context, db bun.IDB, roundID uuid.UUID) (*clubdb.ClubChallenge, error) {
		getRoundCalls++
		if getRoundCalls == 1 {
			return cloneChallengeModel(challenge), nil
		}
		return nil, clubdb.ErrNotFound
	}
	repo.UnlinkActiveChallengeRoundFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID, actorUserUUID *uuid.UUID, unlinkedAt time.Time) error {
		unlinkCalls++
		return nil
	}
	repo.UpdateChallengeFunc = func(ctx context.Context, db bun.IDB, challenge *clubdb.ClubChallenge) error {
		updateCalls++
		return nil
	}
	queue.ScheduleAcceptedExpiryFunc = func(ctx context.Context, challengeID uuid.UUID, expiresAt time.Time) error {
		scheduleCalls++
		return nil
	}

	service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), queue, newChallengeTagReader(fx, 18, 7), nil)
	detail, err := service.ResetChallengeForRound(context.Background(), ChallengeRoundEventRequest{RoundID: fx.roundID})

	require.NoError(t, err)
	assert.Nil(t, detail)
	assert.Equal(t, 2, getRoundCalls)
	assert.Zero(t, unlinkCalls)
	assert.Zero(t, updateCalls)
	assert.Zero(t, scheduleCalls)
}

func TestClubService_ListChallenges_FetchesTagListOnce(t *testing.T) {
	fx := newChallengeFixture()

	challenge1 := cloneChallengeModel(fx.challenge)
	challenge2 := cloneChallengeModel(fx.challenge)
	challenge2.UUID = uuid.New()

	repo := NewFakeClubRepo()
	repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
		return fx.club, nil
	}
	repo.ListChallengesFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, statuses []clubtypes.ChallengeStatus) ([]*clubdb.ClubChallenge, error) {
		return []*clubdb.ClubChallenge{challenge1, challenge2}, nil
	}
	repo.GetActiveChallengeRoundLinkFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallengeRoundLink, error) {
		return nil, clubdb.ErrNotFound
	}

	var tagListCalls int
	tagReader := &FakeChallengeTagReader{
		GetTagListFunc: func(ctx context.Context, guildID sharedtypes.GuildID, clubUUID *string) ([]leaderboardservice.MemberTagView, error) {
			tagListCalls++
			return []leaderboardservice.MemberTagView{
				{MemberID: valueOrEmpty(fx.challengerMembership.ExternalID), Tag: intPtr(18)},
				{MemberID: valueOrEmpty(fx.defenderMembership.ExternalID), Tag: intPtr(7)},
			}, nil
		},
	}

	service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), &FakeChallengeQueueService{}, tagReader, nil)
	summaries, err := service.ListChallenges(context.Background(), ChallengeListRequest{
		Scope: ChallengeScope{ClubUUID: &fx.club.UUID},
	})

	require.NoError(t, err)
	require.Len(t, summaries, 2)
	assert.Equal(t, 1, tagListCalls, "GetTagList must be called exactly once for all challenges, not per challenge")
}

func TestClubService_ListChallenges_PopulatesLinkedRound(t *testing.T) {
	fx := newChallengeFixture()
	challenge := cloneChallengeModel(fx.challenge)
	now := time.Now().UTC()
	challenge.Status = clubtypes.ChallengeStatusAccepted
	challenge.AcceptedAt = &now
	challenge.AcceptedExpiresAt = ptrTime(now.Add(24 * time.Hour))

	repo := NewFakeClubRepo()
	repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
		return fx.club, nil
	}
	repo.ListChallengesFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, statuses []clubtypes.ChallengeStatus) ([]*clubdb.ClubChallenge, error) {
		return []*clubdb.ClubChallenge{challenge}, nil
	}
	repo.GetActiveChallengeRoundLinkFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallengeRoundLink, error) {
		return &clubdb.ClubChallengeRoundLink{
			UUID:             uuid.New(),
			ChallengeUUID:    challenge.UUID,
			RoundID:          fx.roundID,
			LinkedByUserUUID: &fx.challengerMembership.UserUUID,
			LinkedAt:         now,
		}, nil
	}

	service := newChallengeTestService(t, repo, newChallengeUserRepo(fx), &FakeChallengeQueueService{}, newChallengeTagReader(fx, 18, 7), nil)
	summaries, err := service.ListChallenges(context.Background(), ChallengeListRequest{
		Scope: ChallengeScope{ClubUUID: &fx.club.UUID},
	})

	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.NotNil(t, summaries[0].LinkedRound)
	assert.Equal(t, fx.roundID.String(), summaries[0].LinkedRound.RoundID)
	assert.True(t, summaries[0].LinkedRound.IsActive)
	require.NotNil(t, summaries[0].LinkedRound.LinkedBy)
	assert.Equal(t, fx.challengerMembership.UserUUID.String(), *summaries[0].LinkedRound.LinkedBy)
}

func TestClubService_RefreshChallengesForMembers_UsesClubUUIDWithoutDiscordLookup(t *testing.T) {
	fx := newChallengeFixture()
	fx.club.DiscordGuildID = nil

	repo := NewFakeClubRepo()
	repo.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
		return fx.club, nil
	}
	repo.ListActiveChallengesByUsersFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, userUUIDs []uuid.UUID) ([]*clubdb.ClubChallenge, error) {
		require.ElementsMatch(t, []uuid.UUID{fx.challengerMembership.UserUUID, fx.defenderMembership.UserUUID}, userUUIDs)
		return []*clubdb.ClubChallenge{cloneChallengeModel(fx.challenge)}, nil
	}
	repo.GetActiveChallengeRoundLinkFunc = func(ctx context.Context, db bun.IDB, challengeUUID uuid.UUID) (*clubdb.ClubChallengeRoundLink, error) {
		return nil, clubdb.ErrNotFound
	}

	userRepo := newChallengeUserRepo(fx)
	userRepo.GetClubUUIDByDiscordGuildIDFn = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (uuid.UUID, error) {
		t.Fatalf("GetClubUUIDByDiscordGuildID should not be called when ClubUUID is provided")
		return uuid.Nil, nil
	}

	tagReader := &FakeChallengeTagReader{
		GetTagListFunc: func(ctx context.Context, guildID sharedtypes.GuildID, clubUUID *string) ([]leaderboardservice.MemberTagView, error) {
			require.Equal(t, sharedtypes.GuildID(fx.club.UUID.String()), guildID)
			require.NotNil(t, clubUUID)
			require.Equal(t, fx.club.UUID.String(), *clubUUID)
			return []leaderboardservice.MemberTagView{
				{MemberID: valueOrEmpty(fx.challengerMembership.ExternalID), Tag: intPtr(18)},
				{MemberID: valueOrEmpty(fx.defenderMembership.ExternalID), Tag: intPtr(7)},
			}, nil
		},
	}

	service := newChallengeTestService(t, repo, userRepo, &FakeChallengeQueueService{}, tagReader, nil)
	details, err := service.RefreshChallengesForMembers(context.Background(), ChallengeRefreshRequest{
		ClubUUID:    &fx.club.UUID,
		ExternalIDs: []string{valueOrEmpty(fx.challengerMembership.ExternalID), valueOrEmpty(fx.defenderMembership.ExternalID)},
	})

	require.NoError(t, err)
	require.Len(t, details, 1)
	assert.Equal(t, fx.club.UUID.String(), details[0].ClubUUID)
	assert.Equal(t, clubtypes.ChallengeStatusOpen, details[0].Status)
}

func newChallengeTestService(
	t *testing.T,
	repo *FakeClubRepo,
	userRepo *userdb.FakeRepository,
	queue *FakeChallengeQueueService,
	tagReader *FakeChallengeTagReader,
	roundReader *FakeChallengeRoundReader,
) *ClubService {
	t.Helper()

	if repo == nil {
		repo = NewFakeClubRepo()
	}
	if userRepo == nil {
		userRepo = &userdb.FakeRepository{}
	}
	if queue == nil {
		queue = &FakeChallengeQueueService{}
	}
	if tagReader == nil {
		tagReader = &FakeChallengeTagReader{}
	}

	var queuePort ChallengeQueueService
	if queue != nil {
		queuePort = queue
	}

	var tagReaderPort ChallengeTagReader
	if tagReader != nil {
		tagReaderPort = tagReader
	}

	var roundReaderPort ChallengeRoundReader
	if roundReader != nil {
		roundReaderPort = roundReader
	}

	return NewClubService(
		repo,
		userRepo,
		queuePort,
		tagReaderPort,
		roundReaderPort,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		clubmetrics.NewNoop(),
		noop.NewTracerProvider().Tracer("club-service-test"),
		nil,
	)
}

func newChallengeFixture() challengeTestFixture {
	clubUUID := uuid.New()
	challengeUUID := uuid.New()
	roundID := uuid.New()
	discordGuildID := "guild-1"
	challengerExternalID := "challenger-ext"
	defenderExternalID := "defender-ext"

	challengerUserUUID := uuid.New()
	defenderUserUUID := uuid.New()
	now := time.Now().UTC()

	return challengeTestFixture{
		club: &clubdb.Club{
			UUID:           clubUUID,
			Name:           "Club Test",
			DiscordGuildID: &discordGuildID,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		challengerMembership: &userdb.ClubMembership{
			ID:         uuid.New(),
			UserUUID:   challengerUserUUID,
			ClubUUID:   clubUUID,
			Role:       sharedtypes.UserRoleUser,
			Source:     "test",
			ExternalID: &challengerExternalID,
			JoinedAt:   now,
			UpdatedAt:  now,
		},
		defenderMembership: &userdb.ClubMembership{
			ID:         uuid.New(),
			UserUUID:   defenderUserUUID,
			ClubUUID:   clubUUID,
			Role:       sharedtypes.UserRoleUser,
			Source:     "test",
			ExternalID: &defenderExternalID,
			JoinedAt:   now,
			UpdatedAt:  now,
		},
		challenge: &clubdb.ClubChallenge{
			UUID:               challengeUUID,
			ClubUUID:           clubUUID,
			ChallengerUserUUID: challengerUserUUID,
			DefenderUserUUID:   defenderUserUUID,
			Status:             clubtypes.ChallengeStatusOpen,
			OpenedAt:           now,
			OpenExpiresAt:      ptrTime(now.Add(48 * time.Hour)),
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		roundID: roundID,
	}
}

func newChallengeUserRepo(fx challengeTestFixture) *userdb.FakeRepository {
	byUser := map[uuid.UUID]*userdb.ClubMembership{
		fx.challengerMembership.UserUUID: fx.challengerMembership,
		fx.defenderMembership.UserUUID:   fx.defenderMembership,
	}
	byExternalID := map[string]*userdb.ClubMembership{
		valueOrEmpty(fx.challengerMembership.ExternalID): fx.challengerMembership,
		valueOrEmpty(fx.defenderMembership.ExternalID):   fx.defenderMembership,
	}

	return &userdb.FakeRepository{
		GetClubMembershipFn: func(ctx context.Context, db bun.IDB, userUUID, clubUUID uuid.UUID) (*userdb.ClubMembership, error) {
			membership, ok := byUser[userUUID]
			if !ok || membership.ClubUUID != clubUUID {
				return nil, userdb.ErrNotFound
			}
			return membership, nil
		},
		GetClubMembershipByExternalIDFn: func(ctx context.Context, db bun.IDB, externalID string, clubUUID uuid.UUID) (*userdb.ClubMembership, error) {
			membership, ok := byExternalID[externalID]
			if !ok || membership.ClubUUID != clubUUID {
				return nil, userdb.ErrNotFound
			}
			return membership, nil
		},
		GetClubMembershipsByUserUUIDsFn: func(ctx context.Context, db bun.IDB, userUUIDs []uuid.UUID) ([]*userdb.ClubMembership, error) {
			memberships := make([]*userdb.ClubMembership, 0, len(userUUIDs))
			for _, userUUID := range userUUIDs {
				if membership, ok := byUser[userUUID]; ok {
					memberships = append(memberships, membership)
				}
			}
			return memberships, nil
		},
	}
}

func newChallengeTagReader(fx challengeTestFixture, challengerTag, defenderTag int) *FakeChallengeTagReader {
	return &FakeChallengeTagReader{
		GetTagListFunc: func(ctx context.Context, guildID sharedtypes.GuildID, clubUUID *string) ([]leaderboardservice.MemberTagView, error) {
			return []leaderboardservice.MemberTagView{
				{MemberID: valueOrEmpty(fx.challengerMembership.ExternalID), Tag: intPtr(challengerTag)},
				{MemberID: valueOrEmpty(fx.defenderMembership.ExternalID), Tag: intPtr(defenderTag)},
			}, nil
		},
	}
}

func newChallengeRound(fx challengeTestFixture, participantIDs ...sharedtypes.DiscordID) *roundtypes.Round {
	if len(participantIDs) == 0 {
		participantIDs = []sharedtypes.DiscordID{
			sharedtypes.DiscordID(valueOrEmpty(fx.challengerMembership.ExternalID)),
			sharedtypes.DiscordID(valueOrEmpty(fx.defenderMembership.ExternalID)),
		}
	}

	participants := make([]roundtypes.Participant, 0, len(participantIDs))
	for _, participantID := range participantIDs {
		participants = append(participants, roundtypes.Participant{
			UserID:   participantID,
			Response: roundtypes.ResponseAccept,
		})
	}

	return &roundtypes.Round{
		ID:           sharedtypes.RoundID(fx.roundID),
		State:        roundtypes.RoundStateUpcoming,
		Participants: participants,
	}
}

func cloneChallengeModel(challenge *clubdb.ClubChallenge) *clubdb.ClubChallenge {
	if challenge == nil {
		return nil
	}
	clone := *challenge
	clone.OriginalChallengerTag = cloneInt(challenge.OriginalChallengerTag)
	clone.OriginalDefenderTag = cloneInt(challenge.OriginalDefenderTag)
	clone.OpenExpiresAt = cloneTime(challenge.OpenExpiresAt)
	clone.AcceptedAt = cloneTime(challenge.AcceptedAt)
	clone.AcceptedExpiresAt = cloneTime(challenge.AcceptedExpiresAt)
	clone.CompletedAt = cloneTime(challenge.CompletedAt)
	clone.HiddenAt = cloneTime(challenge.HiddenAt)
	clone.HiddenByUserUUID = cloneUUID(challenge.HiddenByUserUUID)
	clone.DiscordGuildID = cloneString(challenge.DiscordGuildID)
	clone.DiscordChannelID = cloneString(challenge.DiscordChannelID)
	clone.DiscordMessageID = cloneString(challenge.DiscordMessageID)
	return &clone
}

func cloneChallengeRoundLink(link *clubdb.ClubChallengeRoundLink) *clubdb.ClubChallengeRoundLink {
	if link == nil {
		return nil
	}
	clone := *link
	clone.LinkedByUserUUID = cloneUUID(link.LinkedByUserUUID)
	clone.UnlinkedByUserUUID = cloneUUID(link.UnlinkedByUserUUID)
	clone.UnlinkedAt = cloneTime(link.UnlinkedAt)
	return &clone
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func cloneUUID(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func intPtr(value int) *int {
	return &value
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
