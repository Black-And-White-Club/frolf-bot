package userhandlers

import (
	"context"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	results "github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	usermocks "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleUserRoleUpdateRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("55555555555555555")
	testNewRole := sharedtypes.UserRoleEnum("admin")

	mockUserService := usermocks.NewMockService(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	tests := []struct {
		name      string
		payload   *userevents.UserRoleUpdateRequestedPayloadV1
		mockSetup func()
		wantLen   int
		wantTopic string
		wantErr   bool
	}{
		{
			name: "Successfully handle UserRoleUpdateRequest",
			payload: &userevents.UserRoleUpdateRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
				Role:    testNewRole,
			},
			mockSetup: func() {
				updateResultPayload := &userevents.UserRoleUpdateResultPayloadV1{
					GuildID: testGuildID,
					Success: true,
					UserID:  testUserID,
					Role:    testNewRole,
					Reason:  "",
				}

				mockUserService.EXPECT().UpdateUserRoleInDatabase(gomock.Any(), testGuildID, testUserID, testNewRole).Return(
					results.OperationResult{
						Success: updateResultPayload,
						Failure: nil,
					},
					nil,
				)
			},
			wantLen:   1,
			wantTopic: userevents.UserRoleUpdatedV1,
			wantErr:   false,
		},
		{
			name: "Failed to update user role",
			payload: &userevents.UserRoleUpdateRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
				Role:    testNewRole,
			},
			mockSetup: func() {
				failurePayload := &userevents.UserRoleUpdateResultPayloadV1{
					GuildID: testGuildID,
					Success: false,
					UserID:  testUserID,
					Role:    testNewRole,
					Reason:  "user not found",
				}

				mockUserService.EXPECT().UpdateUserRoleInDatabase(gomock.Any(), testGuildID, testUserID, testNewRole).Return(
					results.OperationResult{
						Success: nil,
						Failure: failurePayload,
					},
					nil,
				)
			},
			wantLen:   1,
			wantTopic: userevents.UserRoleUpdateFailedV1,
			wantErr:   false,
		},
		{
			name: "Service failure in UpdateUserRoleInDatabase",
			payload: &userevents.UserRoleUpdateRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
				Role:    testNewRole,
			},
			mockSetup: func() {
				mockUserService.EXPECT().UpdateUserRoleInDatabase(gomock.Any(), testGuildID, testUserID, testNewRole).Return(
					results.OperationResult{},
					context.DeadlineExceeded,
				)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := NewUserHandlers(mockUserService, logger, tracer, nil, metrics)

			results, err := h.HandleUserRoleUpdateRequest(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUserRoleUpdateRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(results) != tt.wantLen {
					t.Errorf("HandleUserRoleUpdateRequest() got %d results, want %d", len(results), tt.wantLen)
					return
				}
				if results[0].Topic != tt.wantTopic {
					t.Errorf("HandleUserRoleUpdateRequest() got topic %s, want %s", results[0].Topic, tt.wantTopic)
				}
			}
		})
	}
}
