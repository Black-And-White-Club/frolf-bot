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
	"go.opentelemetry.io/otel/trace/noop"
)

func TestUserHandlers_HandleUserRoleUpdateRequest(t *testing.T) {
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("55555555555555555")
	testNewRole := sharedtypes.UserRoleEnum("admin")

	tests := []struct {
		name      string
		payload   *userevents.UserRoleUpdateRequestedPayloadV1
		setupFake func(*FakeUserService)
		wantErr   bool
		wantTopic string
		wantLen   int
	}{
		{
			name: "success - user role updated",
			payload: &userevents.UserRoleUpdateRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
				Role:    testNewRole,
			},
			setupFake: func(f *FakeUserService) {
				f.UpdateUserRoleInDatabaseFunc = func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum) (results.OperationResult[bool, error], error) {
					// Return success result (bool true)
					return results.SuccessResult[bool, error](true), nil
				}
			},
			wantErr:   false,
			wantTopic: userevents.UserRoleUpdatedV1,
			wantLen:   1,
		},
		{
			name: "failure - domain error (user not found)",
			payload: &userevents.UserRoleUpdateRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
				Role:    testNewRole,
			},
			setupFake: func(f *FakeUserService) {
				f.UpdateUserRoleInDatabaseFunc = func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum) (results.OperationResult[bool, error], error) {
					// Simulate domain failure (user does not exist)
					return results.FailureResult[bool, error](errors.New("user not found")), nil
				}
			},
			wantErr:   false,
			wantTopic: userevents.UserRoleUpdateFailedV1,
			wantLen:   1,
		},
		{
			name: "error - service infrastructure failure",
			payload: &userevents.UserRoleUpdateRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
				Role:    testNewRole,
			},
			setupFake: func(f *FakeUserService) {
				f.UpdateUserRoleInDatabaseFunc = func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum) (results.OperationResult[bool, error], error) {
					// Simulate database/timeout error
					return results.OperationResult[bool, error]{}, context.DeadlineExceeded
				}
			},
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize the manual fake service
			fakeService := NewFakeUserService()
			if tt.setupFake != nil {
				tt.setupFake(fakeService)
			}

			logger := loggerfrolfbot.NoOpLogger
			tracer := noop.NewTracerProvider().Tracer("test")
			metrics := &usermetrics.NoOpMetrics{}

			// Inject the fake service into the handler
			h := NewUserHandlers(fakeService, logger, tracer, nil, metrics)
			res, err := h.HandleUserRoleUpdateRequest(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUserRoleUpdateRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(res) != tt.wantLen {
				t.Errorf("HandleUserRoleUpdateRequest() got %d results, want %d", len(res), tt.wantLen)
				return
			}

			if len(res) > 0 && res[0].Topic != tt.wantTopic {
				t.Errorf("HandleUserRoleUpdateRequest() got topic %s, want %s", res[0].Topic, tt.wantTopic)
			}
		})
	}
}
