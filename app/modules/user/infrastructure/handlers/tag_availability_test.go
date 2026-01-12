package userhandlers

import (
	"context"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	usermocks "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleTagAvailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("98765432109876543")
	testTagNumber := sharedtypes.TagNumber(1)

	mockUserService := usermocks.NewMockService(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	tests := []struct {
		name      string
		payload   *userevents.TagAvailablePayloadV1
		mockSetup func()
		wantLen   int
		wantTopic string
		wantErr   bool
	}{
		{
			name: "Successfully handle TagAvailable event",
			payload: &userevents.TagAvailablePayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: testTagNumber,
			},
			mockSetup: func() {
				mockUserService.EXPECT().CreateUser(gomock.Any(), testGuildID, testUserID, gomock.Eq(&testTagNumber), nil, nil).Return(
					userservice.UserOperationResult{
						Success: &userevents.UserCreatedPayloadV1{GuildID: testGuildID, UserID: testUserID, TagNumber: &testTagNumber},
						Failure: nil,
						Error:   nil,
					},
					nil,
				)
			},
			wantLen:   1,
			wantTopic: userevents.UserCreatedV1,
			wantErr:   false,
		},
		{
			name: "Fail to create user",
			payload: &userevents.TagAvailablePayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: testTagNumber,
			},
			mockSetup: func() {
				mockUserService.EXPECT().CreateUser(gomock.Any(), testGuildID, testUserID, gomock.Eq(&testTagNumber), nil, nil).Return(
					userservice.UserOperationResult{
						Success: nil,
						Failure: &userevents.UserCreationFailedPayloadV1{
							GuildID:   testGuildID,
							UserID:    testUserID,
							TagNumber: &testTagNumber,
							Reason:    "failed",
						},
						Error: nil,
					},
					nil,
				)
			},
			wantLen:   1,
			wantTopic: userevents.UserCreationFailedV1,
			wantErr:   false,
		},
		{
			name: "Service failure in CreateUser",
			payload: &userevents.TagAvailablePayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: testTagNumber,
			},
			mockSetup: func() {
				mockUserService.EXPECT().CreateUser(gomock.Any(), testGuildID, testUserID, gomock.Eq(&testTagNumber), nil, nil).Return(
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

			results, err := h.HandleTagAvailable(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagAvailable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(results) != tt.wantLen {
					t.Errorf("HandleTagAvailable() got %d results, want %d", len(results), tt.wantLen)
					return
				}
				if results[0].Topic != tt.wantTopic {
					t.Errorf("HandleTagAvailable() got topic %s, want %s", results[0].Topic, tt.wantTopic)
				}
			}
		})
	}
}

func TestUserHandlers_HandleTagUnavailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUserID := sharedtypes.DiscordID("98765432109876543")
	testGuildID := sharedtypes.GuildID("98765432109876543")
	testTagNumber := sharedtypes.TagNumber(2)

	mockUserService := usermocks.NewMockService(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	tests := []struct {
		name      string
		payload   *userevents.TagUnavailablePayloadV1
		mockSetup func()
		wantLen   int
		wantTopic string
		wantErr   bool
	}{
		{
			name: "Successfully handle TagUnavailable event",
			payload: &userevents.TagUnavailablePayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: testTagNumber,
				Reason:    "tag not available",
			},
			mockSetup: func() {
				// No service call needed
			},
			wantLen:   1,
			wantTopic: userevents.UserCreationFailedV1,
			wantErr:   false,
		},
		{
			name: "Handle empty reason",
			payload: &userevents.TagUnavailablePayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: testTagNumber,
				Reason:    "",
			},
			mockSetup: func() {
				// No service call needed
			},
			wantLen:   1,
			wantTopic: userevents.UserCreationFailedV1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := NewUserHandlers(mockUserService, logger, tracer, nil, metrics)

			results, err := h.HandleTagUnavailable(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagUnavailable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(results) != tt.wantLen {
					t.Errorf("HandleTagUnavailable() got %d results, want %d", len(results), tt.wantLen)
					return
				}
				if results[0].Topic != tt.wantTopic {
					t.Errorf("HandleTagUnavailable() got topic %s, want %s", results[0].Topic, tt.wantTopic)
				}
			}
		})
	}
}
