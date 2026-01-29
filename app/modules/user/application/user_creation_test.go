package userservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestUserService_CreateUser(t *testing.T) {
	ctx := context.Background()
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("98765432109876543")
	testTag := sharedtypes.TagNumber(42)

	logger := loggerfrolfbot.NoOpLogger
	metrics := &usermetrics.NoOpMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name           string
		userID         sharedtypes.DiscordID
		guildID        sharedtypes.GuildID
		tag            *sharedtypes.TagNumber
		setupFake      func(*FakeUserRepository)
		expectInfraErr bool
		verify         func(t *testing.T, res UserResult, infraErr error, fake *FakeUserRepository)
	}{
		{
			name:    "success - new user",
			userID:  testUserID,
			guildID: testGuildID,
			tag:     &testTag,
			setupFake: func(f *FakeUserRepository) {
				f.GetUserGlobalFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.DiscordID) (*userdb.User, error) {
					return nil, userdb.ErrNotFound
				}
				f.SaveGlobalUserFunc = func(ctx context.Context, db bun.IDB, user *userdb.User) error {
					return nil
				}
				f.CreateGuildMembershipFunc = func(ctx context.Context, db bun.IDB, m *userdb.GuildMembership) error {
					return nil
				}
			},
			verify: func(t *testing.T, res UserResult, infraErr error, fake *FakeUserRepository) {
				if infraErr != nil {
					t.Fatalf("unexpected infra error: %v", infraErr)
				}
				if res.Success == nil {
					t.Fatalf("expected success, got failure: %v", res.Failure)
				}
				if (*res.Success).GetUserID() != testUserID {
					t.Errorf("expected userID %s, got %s", testUserID, (*res.Success).GetUserID())
				}
				if (*res.Success).TagNumber == nil || *(*res.Success).TagNumber != testTag {
					t.Errorf("expected tag %v, got %v", testTag, (*res.Success).TagNumber)
				}
				if (*res.Success).IsReturningUser {
					t.Errorf("expected IsReturningUser to be false for new user")
				}
			},
		},
		{
			name:    "success - returning user (existing global, new membership)",
			userID:  testUserID,
			guildID: testGuildID,
			setupFake: func(f *FakeUserRepository) {
				f.GetUserGlobalFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.DiscordID) (*userdb.User, error) {
					return &userdb.User{UserID: testUserID}, nil
				}
				f.GetGuildMembershipFunc = func(ctx context.Context, db bun.IDB, uID sharedtypes.DiscordID, gID sharedtypes.GuildID) (*userdb.GuildMembership, error) {
					return nil, userdb.ErrNotFound
				}
				f.CreateGuildMembershipFunc = func(ctx context.Context, db bun.IDB, m *userdb.GuildMembership) error {
					return nil
				}
			},
			verify: func(t *testing.T, res UserResult, infraErr error, fake *FakeUserRepository) {
				if res.Success == nil {
					t.Fatalf("expected success for returning user, got failure: %v", res.Failure)
				}
				// Verify mapping happened
				if (*res.Success).GetUserID() != testUserID {
					t.Errorf("expected userID %s, got %s", testUserID, (*res.Success).GetUserID())
				}
				if !(*res.Success).IsReturningUser {
					t.Errorf("expected IsReturningUser to be true for existing user")
				}
			},
		},
		{
			name:    "domain failure - user already exists in guild",
			userID:  testUserID,
			guildID: testGuildID,
			setupFake: func(f *FakeUserRepository) {
				f.GetUserGlobalFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.DiscordID) (*userdb.User, error) {
					return &userdb.User{UserID: testUserID}, nil
				}
				f.GetGuildMembershipFunc = func(ctx context.Context, db bun.IDB, uID sharedtypes.DiscordID, gID sharedtypes.GuildID) (*userdb.GuildMembership, error) {
					return &userdb.GuildMembership{UserID: uID, GuildID: gID}, nil
				}
			},
			verify: func(t *testing.T, res UserResult, infraErr error, fake *FakeUserRepository) {
				if infraErr != nil {
					t.Fatalf("unexpected infra error: %v", infraErr)
				}
				if res.Failure == nil || !errors.Is(*res.Failure, ErrUserAlreadyExists) {
					t.Errorf("expected ErrUserAlreadyExists, got %v", res.Failure)
				}
			},
		},
		{
			name:           "domain failure - invalid discord ID",
			userID:         "",
			guildID:        testGuildID,
			expectInfraErr: false,
			verify: func(t *testing.T, res UserResult, infraErr error, fake *FakeUserRepository) {
				if res.Failure == nil || !errors.Is(*res.Failure, ErrInvalidDiscordID) {
					t.Errorf("expected ErrInvalidDiscordID, got %v", res.Failure)
				}
				if len(fake.Trace()) > 0 {
					t.Errorf("repo should not be called for invalid input")
				}
			},
		},
		{
			name:    "infra failure - database error on save",
			userID:  testUserID,
			guildID: testGuildID,
			setupFake: func(f *FakeUserRepository) {
				f.GetUserGlobalFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.DiscordID) (*userdb.User, error) {
					return nil, userdb.ErrNotFound
				}
				f.SaveGlobalUserFunc = func(ctx context.Context, db bun.IDB, user *userdb.User) error {
					return errors.New("db connection lost")
				}
			},
			expectInfraErr: true,
			verify: func(t *testing.T, res UserResult, infraErr error, fake *FakeUserRepository) {
				if infraErr == nil || !strings.Contains(infraErr.Error(), "db connection lost") {
					t.Errorf("expected infra error containing 'db connection lost', got %v", infraErr)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := &FakeUserRepository{}
			if tt.setupFake != nil {
				tt.setupFake(fakeRepo)
			}

			s := &UserService{
				repo:    fakeRepo,
				logger:  logger,
				metrics: metrics,
				tracer:  tracer,
			}

			res, err := s.CreateUser(ctx, tt.guildID, tt.userID, tt.tag, nil, nil)

			if tt.expectInfraErr && err == nil {
				t.Fatal("expected infrastructure error but got nil")
			}
			if !tt.expectInfraErr && err != nil {
				t.Fatalf("unexpected infrastructure error: %v", err)
			}

			if tt.verify != nil {
				tt.verify(t, res, err, fakeRepo)
			}
		})
	}
}
