package userhandlers

import (
	"context"
	"errors"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestUserHandlers_HandleGetUserRequest(t *testing.T) {
	testGuildID := sharedtypes.GuildID("55555555555555555")
	testUserID := sharedtypes.DiscordID("12345678901234567")

	tests := []struct {
		name      string
		payload   *userevents.GetUserRequestedPayloadV1
		setupFake func(*FakeUserService)
		wantLen   int
		wantTopic string
		wantErr   bool
	}{
		{
			name: "success - user found",
			payload: &userevents.GetUserRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
			},
			setupFake: func(f *FakeUserService) {
				f.GetUserFunc = func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (userservice.UserWithMembershipResult, error) {
					return results.SuccessResult[*userservice.UserWithMembership, error](&userservice.UserWithMembership{}), nil
				}
			},
			wantLen:   1,
			wantTopic: userevents.GetUserResponseV1,
			wantErr:   false,
		},
		{
			name: "failure - user not found",
			payload: &userevents.GetUserRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
			},
			setupFake: func(f *FakeUserService) {
				f.GetUserFunc = func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (userservice.UserWithMembershipResult, error) {
					return results.FailureResult[*userservice.UserWithMembership, error](errors.New("user not found")), nil
				}
			},
			wantLen:   1,
			wantTopic: userevents.GetUserFailedV1,
			wantErr:   false,
		},
		{
			name: "error - service error",
			payload: &userevents.GetUserRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
			},
			setupFake: func(f *FakeUserService) {
				f.GetUserFunc = func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (userservice.UserWithMembershipResult, error) {
					return userservice.UserWithMembershipResult{}, context.DeadlineExceeded
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := NewFakeUserService()
			if tt.setupFake != nil {
				tt.setupFake(fakeService)
			}

			h := NewUserHandlers(fakeService, loggerfrolfbot.NoOpLogger, noop.NewTracerProvider().Tracer("test"), nil, &usermetrics.NoOpMetrics{})
			res, err := h.HandleGetUserRequest(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(res) != tt.wantLen {
					t.Errorf("got %d results, want %d", len(res), tt.wantLen)
				}
				if res[0].Topic != tt.wantTopic {
					t.Errorf("got topic %s, want %s", res[0].Topic, tt.wantTopic)
				}
			}
		})
	}
}

func TestUserHandlers_HandleGetUserRoleRequest(t *testing.T) {
	testGuildID := sharedtypes.GuildID("55555555555555555")
	testUserID := sharedtypes.DiscordID("12345678901234567")

	tests := []struct {
		name      string
		payload   *userevents.GetUserRoleRequestedPayloadV1
		setupFake func(*FakeUserService)
		wantLen   int
		wantTopic string
		wantErr   bool
	}{
		{
			name: "success - role found",
			payload: &userevents.GetUserRoleRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
			},
			setupFake: func(f *FakeUserService) {
				f.GetUserRoleFunc = func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult[sharedtypes.UserRoleEnum, error], error) {
					return results.SuccessResult[sharedtypes.UserRoleEnum, error]("admin"), nil
				}
			},
			wantLen:   1,
			wantTopic: userevents.GetUserRoleResponseV1,
			wantErr:   false,
		},
		{
			name: "failure - role not found",
			payload: &userevents.GetUserRoleRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
			},
			setupFake: func(f *FakeUserService) {
				f.GetUserRoleFunc = func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult[sharedtypes.UserRoleEnum, error], error) {
					return results.FailureResult[sharedtypes.UserRoleEnum, error](errors.New("role not found")), nil
				}
			},
			wantLen:   1,
			wantTopic: userevents.GetUserRoleFailedV1,
			wantErr:   false,
		},
		{
			name: "error - service error",
			payload: &userevents.GetUserRoleRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
			},
			setupFake: func(f *FakeUserService) {
				f.GetUserRoleFunc = func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult[sharedtypes.UserRoleEnum, error], error) {
					return results.OperationResult[sharedtypes.UserRoleEnum, error]{}, context.DeadlineExceeded
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := NewFakeUserService()
			if tt.setupFake != nil {
				tt.setupFake(fakeService)
			}

			h := NewUserHandlers(fakeService, loggerfrolfbot.NoOpLogger, noop.NewTracerProvider().Tracer("test"), nil, &usermetrics.NoOpMetrics{})
			res, err := h.HandleGetUserRoleRequest(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(res) != tt.wantLen {
					t.Errorf("got %d results, want %d", len(res), tt.wantLen)
				}
				if res[0].Topic != tt.wantTopic {
					t.Errorf("got topic %s, want %s", res[0].Topic, tt.wantTopic)
				}
			}
		})
	}
}
