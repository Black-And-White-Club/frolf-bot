package userservice

import (
	"context"
	"errors" // Import fmt for error wrapping
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

func TestUserService_UpdateUserRoleInDatabase(t *testing.T) {
	ctx := context.Background()
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("98765432109876543")
	testRole := sharedtypes.UserRoleAdmin

	logger := loggerfrolfbot.NoOpLogger
	metrics := &usermetrics.NoOpMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name           string
		userID         sharedtypes.DiscordID
		newRole        sharedtypes.UserRoleEnum
		setupFake      func(*FakeUserRepository)
		expectInfraErr bool
		verify         func(t *testing.T, res results.OperationResult[bool, error], infraErr error)
	}{
		{
			name:    "success",
			userID:  testUserID,
			newRole: testRole,
			setupFake: func(f *FakeUserRepository) {
				f.UpdateUserRoleFunc = func(ctx context.Context, db bun.IDB, uID sharedtypes.DiscordID, gID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error {
					return nil
				}
			},
			verify: func(t *testing.T, res results.OperationResult[bool, error], infraErr error) {
				if infraErr != nil {
					t.Fatalf("unexpected infra error: %v", infraErr)
				}
				if res.Success == nil || !*res.Success {
					t.Errorf("expected success result true, got %v", res.Success)
				}
			},
		},
		{
			name:    "domain failure - invalid role",
			userID:  testUserID,
			newRole: sharedtypes.UserRoleEnum("invalid"),
			verify: func(t *testing.T, res results.OperationResult[bool, error], infraErr error) {
				if res.Failure == nil || !strings.Contains((*res.Failure).Error(), "invalid role") {
					t.Errorf("expected invalid role failure, got %v", res.Failure)
				}
			},
		},
		{
			name:    "domain failure - user not found",
			userID:  testUserID,
			newRole: testRole,
			setupFake: func(f *FakeUserRepository) {
				f.UpdateUserRoleFunc = func(ctx context.Context, db bun.IDB, uID sharedtypes.DiscordID, gID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error {
					return userdb.ErrNotFound
				}
			},
			verify: func(t *testing.T, res results.OperationResult[bool, error], infraErr error) {
				if res.Failure == nil || !errors.Is(*res.Failure, ErrUserNotFound) {
					t.Errorf("expected ErrUserNotFound in failure result, got %v", res.Failure)
				}
			},
		},
		{
			name:    "infra failure - database error",
			userID:  testUserID,
			newRole: testRole,
			setupFake: func(f *FakeUserRepository) {
				f.UpdateUserRoleFunc = func(ctx context.Context, db bun.IDB, uID sharedtypes.DiscordID, gID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error {
					return errors.New("db connection timeout")
				}
			},
			expectInfraErr: true,
			verify: func(t *testing.T, res results.OperationResult[bool, error], infraErr error) {
				if infraErr == nil || !strings.Contains(infraErr.Error(), "db connection timeout") {
					t.Errorf("expected connection timeout error, got %v", infraErr)
				}
			},
		},
		{
			name:   "domain failure - empty discord ID",
			userID: "",
			verify: func(t *testing.T, res results.OperationResult[bool, error], infraErr error) {
				if res.Failure == nil || !errors.Is(*res.Failure, ErrInvalidDiscordID) {
					t.Errorf("expected ErrInvalidDiscordID, got %v", res.Failure)
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

			res, err := s.UpdateUserRoleInDatabase(ctx, testGuildID, tt.userID, tt.newRole)

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
