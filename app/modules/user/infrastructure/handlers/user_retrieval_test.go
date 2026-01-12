package userhandlers

import (
	"context"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	usermocks "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleGetUserRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testGuildID := sharedtypes.GuildID("55555555555555555")
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testUserData := &usertypes.UserData{
		UserID: testUserID,
		Role:   "member",
	}

	mockUserService := usermocks.NewMockService(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	tests := []struct {
		name      string
		payload   *userevents.GetUserRequestedPayloadV1
		mockSetup func()
		wantLen   int
		wantTopic string
		wantErr   bool
	}{
		{
			name: "Successfully handle GetUser Request",
			payload: &userevents.GetUserRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
			},
			mockSetup: func() {
				mockUserService.EXPECT().GetUser(gomock.Any(), testGuildID, testUserID).Return(
					userservice.UserOperationResult{
						Success: &userevents.GetUserResponsePayloadV1{User: testUserData},
						Failure: nil,
						Error:   nil,
					},
					nil,
				)
			},
			wantLen:   1,
			wantTopic: userevents.GetUserResponseV1,
			wantErr:   false,
		},
		{
			name: "User not found",
			payload: &userevents.GetUserRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
			},
			mockSetup: func() {
				mockUserService.EXPECT().GetUser(gomock.Any(), testGuildID, testUserID).Return(
					userservice.UserOperationResult{
						Success: nil,
						Failure: &userevents.GetUserFailedPayloadV1{
							UserID: testUserID,
							Reason: "user not found",
						},
						Error: nil,
					},
					nil,
				)
			},
			wantLen:   1,
			wantTopic: userevents.GetUserFailedV1,
			wantErr:   false,
		},
		{
			name: "Service failure in GetUser",
			payload: &userevents.GetUserRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
			},
			mockSetup: func() {
				mockUserService.EXPECT().GetUser(gomock.Any(), testGuildID, testUserID).Return(
					userservice.UserOperationResult{},
					nil,
				)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := NewUserHandlers(mockUserService, logger, tracer, nil, metrics)

			results, err := h.HandleGetUserRequest(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetUserRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(results) != tt.wantLen {
					t.Errorf("HandleGetUserRequest() got %d results, want %d", len(results), tt.wantLen)
					return
				}
				if results[0].Topic != tt.wantTopic {
					t.Errorf("HandleGetUserRequest() got topic %s, want %s", results[0].Topic, tt.wantTopic)
				}
			}
		})
	}
}

func TestUserHandlers_HandleGetUserRoleRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testGuildID := sharedtypes.GuildID("55555555555555555")
	testUserID := sharedtypes.DiscordID("12345678901234567")

	mockUserService := usermocks.NewMockService(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	tests := []struct {
		name      string
		payload   *userevents.GetUserRoleRequestedPayloadV1
		mockSetup func()
		wantLen   int
		wantTopic string
		wantErr   bool
	}{
		{
			name: "Successfully handle GetUserRoleRequest",
			payload: &userevents.GetUserRoleRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
			},
			mockSetup: func() {
				mockUserService.EXPECT().GetUserRole(gomock.Any(), testGuildID, testUserID).Return(
					userservice.UserOperationResult{
						Success: &userevents.GetUserRoleResponsePayloadV1{UserID: testUserID, Role: "admin"},
						Failure: nil,
						Error:   nil,
					},
					nil,
				)
			},
			wantLen:   1,
			wantTopic: userevents.GetUserRoleResponseV1,
			wantErr:   false,
		},
		{
			name: "User role not found",
			payload: &userevents.GetUserRoleRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
			},
			mockSetup: func() {
				mockUserService.EXPECT().GetUserRole(gomock.Any(), testGuildID, testUserID).Return(
					userservice.UserOperationResult{
						Success: nil,
						Failure: &userevents.GetUserRoleFailedPayloadV1{
							UserID: testUserID,
							Reason: "user role not found",
						},
						Error: nil,
					},
					nil,
				)
			},
			wantLen:   1,
			wantTopic: userevents.GetUserRoleFailedV1,
			wantErr:   false,
		},
		{
			name: "Service failure in GetUserRole",
			payload: &userevents.GetUserRoleRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
			},
			mockSetup: func() {
				mockUserService.EXPECT().GetUserRole(gomock.Any(), testGuildID, testUserID).Return(
					userservice.UserOperationResult{},
					nil,
				)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := NewUserHandlers(mockUserService, logger, tracer, nil, metrics)

			results, err := h.HandleGetUserRoleRequest(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetUserRoleRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(results) != tt.wantLen {
					t.Errorf("HandleGetUserRoleRequest() got %d results, want %d", len(results), tt.wantLen)
					return
				}
				if results[0].Topic != tt.wantTopic {
					t.Errorf("HandleGetUserRoleRequest() got topic %s, want %s", results[0].Topic, tt.wantTopic)
				}
			}
		})
	}
}
