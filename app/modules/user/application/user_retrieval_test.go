package userservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	results "github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestUserService_GetUser(t *testing.T) {
	ctx := context.Background()
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("98765432109876543")

	logger := loggerfrolfbot.NoOpLogger
	metrics := &usermetrics.NoOpMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name           string
		userID         sharedtypes.DiscordID
		setupFake      func(*FakeUserRepository)
		expectInfraErr bool
		// UPDATED: Now expects the domain result alias
		verify func(t *testing.T, res UserWithMembershipResult, infraErr error)
	}{
		{
			name:   "success",
			userID: testUserID,
			setupFake: func(f *FakeUserRepository) {
				f.GetUserByUserIDFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.DiscordID, gID sharedtypes.GuildID) (*userdb.UserWithMembership, error) {
					return &userdb.UserWithMembership{
						User: &userdb.User{ID: 1, UserID: id},
						// Assuming db structure is dbUser.Role based on your Map function
						Role: sharedtypes.UserRoleAdmin,
					}, nil
				}
			},
			verify: func(t *testing.T, res UserWithMembershipResult, infraErr error) {
				if infraErr != nil {
					t.Fatalf("unexpected infra error: %v", infraErr)
				}
				// UPDATED: Accessing the clean domain struct fields
				if res.Success == nil || (*res.Success).UserData.UserID != testUserID {
					t.Errorf("expected success for user %s, got %v", testUserID, res.Success)
				}
				if (*res.Success).UserData.Role != sharedtypes.UserRoleAdmin {
					t.Errorf("expected role Admin, got %v", (*res.Success).UserData.Role)
				}
			},
		},
		{
			name:   "not found (domain failure)",
			userID: testUserID,
			setupFake: func(f *FakeUserRepository) {
				f.GetUserByUserIDFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.DiscordID, gID sharedtypes.GuildID) (*userdb.UserWithMembership, error) {
					return nil, userdb.ErrNotFound
				}
			},
			verify: func(t *testing.T, res UserWithMembershipResult, infraErr error) {
				if infraErr != nil {
					t.Fatalf("unexpected infra error: %v", infraErr)
				}
				if res.Failure == nil || !errors.Is(*res.Failure, ErrUserNotFound) {
					t.Errorf("expected ErrUserNotFound in failure, got %v", res.Failure)
				}
			},
		},
		{
			name:   "database error (infra failure)",
			userID: testUserID,
			setupFake: func(f *FakeUserRepository) {
				f.GetUserByUserIDFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.DiscordID, gID sharedtypes.GuildID) (*userdb.UserWithMembership, error) {
					return nil, errors.New("connection failed")
				}
			},
			expectInfraErr: true,
			verify: func(t *testing.T, res UserWithMembershipResult, infraErr error) {
				if infraErr == nil || !strings.Contains(infraErr.Error(), "connection failed") {
					t.Errorf("expected connection failed error, got %v", infraErr)
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

			res, err := s.GetUser(ctx, testGuildID, tt.userID)

			if tt.expectInfraErr && err == nil {
				t.Fatal("expected infra error but got nil")
			}
			if !tt.expectInfraErr && err != nil {
				t.Fatalf("unexpected infra error: %v", err)
			}

			if tt.verify != nil {
				tt.verify(t, res, err)
			}
		})
	}
}
func TestUserService_GetUserRole(t *testing.T) {
	ctx := context.Background()
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("98765432109876543")

	logger := loggerfrolfbot.NoOpLogger
	metrics := &usermetrics.NoOpMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name           string
		setupFake      func(*FakeUserRepository)
		expectInfraErr bool
		verify         func(t *testing.T, res results.OperationResult[sharedtypes.UserRoleEnum, error], infraErr error)
	}{
		{
			name: "success",
			setupFake: func(f *FakeUserRepository) {
				f.GetUserRoleFunc = func(ctx context.Context, db bun.IDB, uID sharedtypes.DiscordID, gID sharedtypes.GuildID) (sharedtypes.UserRoleEnum, error) {
					return sharedtypes.UserRoleAdmin, nil
				}
			},
			verify: func(t *testing.T, res results.OperationResult[sharedtypes.UserRoleEnum, error], infraErr error) {
				if res.Success == nil || *res.Success != sharedtypes.UserRoleAdmin {
					t.Errorf("expected role Admin, got %v", res.Success)
				}
			},
		},
		{
			name: "invalid role (domain failure)",
			setupFake: func(f *FakeUserRepository) {
				f.GetUserRoleFunc = func(ctx context.Context, db bun.IDB, uID sharedtypes.DiscordID, gID sharedtypes.GuildID) (sharedtypes.UserRoleEnum, error) {
					return sharedtypes.UserRoleEnum("garbage"), nil
				}
			},
			verify: func(t *testing.T, res results.OperationResult[sharedtypes.UserRoleEnum, error], infraErr error) {
				if res.Failure == nil || !strings.Contains((*res.Failure).Error(), "invalid role") {
					t.Errorf("expected invalid role error in failure, got %v", res.Failure)
				}
			},
		},
		{
			name: "database error (infra failure)",
			setupFake: func(f *FakeUserRepository) {
				f.GetUserRoleFunc = func(ctx context.Context, db bun.IDB, uID sharedtypes.DiscordID, gID sharedtypes.GuildID) (sharedtypes.UserRoleEnum, error) {
					return "", errors.New("db error")
				}
			},
			expectInfraErr: true,
			verify: func(t *testing.T, res results.OperationResult[sharedtypes.UserRoleEnum, error], infraErr error) {
				if infraErr == nil {
					t.Fatal("expected infra error")
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

			res, err := s.GetUserRole(ctx, testGuildID, testUserID)

			if tt.expectInfraErr && err == nil {
				t.Fatal("expected infra error")
			}

			if tt.verify != nil {
				tt.verify(t, res, err)
			}
		})
	}
}
