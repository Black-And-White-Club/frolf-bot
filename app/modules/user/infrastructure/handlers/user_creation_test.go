package userhandlers

import (
	"context"
	"testing"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	usermocks "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleUserSignupRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("33333333333333333")
	testTagNumber := sharedtypes.TagNumber(1)

	mockUserService := usermocks.NewMockService(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	tests := []struct {
		name         string
		payload      *userevents.UserSignupRequestedPayloadV1
		mockSetup    func()
		wantLen      int
		wantTopic    string
		wantErr      bool
	}{
		{
			name: "Successful signup with tag",
			payload: &userevents.UserSignupRequestedPayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: &testTagNumber,
			},
			mockSetup: func() {
				// No service call when tag is present
			},
			wantLen:   1,
			wantTopic: sharedevents.TagAvailabilityCheckRequestedV1,
			wantErr:   false,
		},
		{
			name: "Successful signup without tag",
			payload: &userevents.UserSignupRequestedPayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: nil,
			},
			mockSetup: func() {
				mockUserService.EXPECT().CreateUser(gomock.Any(), testGuildID, testUserID, nil, nil, nil).
					Return(userservice.UserOperationResult{
						Success: &userevents.UserCreatedPayloadV1{UserID: testUserID},
						Failure: nil,
						Error:   nil,
					}, nil)
			},
			wantLen:   1,
			wantTopic: userevents.UserCreatedV1,
			wantErr:   false,
		},
		{
			name: "Failed signup",
			payload: &userevents.UserSignupRequestedPayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: nil,
			},
			mockSetup: func() {
				mockUserService.EXPECT().CreateUser(gomock.Any(), testGuildID, testUserID, nil, nil, nil).
					Return(userservice.UserOperationResult{
						Success: nil,
						Failure: &userevents.UserCreationFailedPayloadV1{UserID: testUserID, Reason: "failed"},
						Error:   nil,
					}, nil)
			},
			wantLen:   1,
			wantTopic: userevents.UserCreationFailedV1,
			wantErr:   false,
		},
		{
			name: "Error from CreateUser",
			payload: &userevents.UserSignupRequestedPayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: nil,
			},
			mockSetup: func() {
				mockUserService.EXPECT().CreateUser(gomock.Any(), testGuildID, testUserID, nil, nil, nil).
					Return(userservice.UserOperationResult{}, nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			h := NewUserHandlers(mockUserService, logger, tracer, nil, metrics)

			results, err := h.HandleUserSignupRequest(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUserSignupRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(results) != tt.wantLen {
					t.Errorf("HandleUserSignupRequest() got %d results, want %d", len(results), tt.wantLen)
					return
				}
				if results[0].Topic != tt.wantTopic {
					t.Errorf("HandleUserSignupRequest() got topic %s, want %s", results[0].Topic, tt.wantTopic)
				}
			}
		})
	}
}
