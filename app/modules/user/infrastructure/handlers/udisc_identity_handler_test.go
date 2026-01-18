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

func TestUserHandlers_HandleUpdateUDiscIdentityRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("33333333333333333")
	testUsername := "testuser"
	testName := "Test User"

	mockUserService := usermocks.NewMockService(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	tests := []struct {
		name      string
		payload   *userevents.UpdateUDiscIdentityRequestedPayloadV1
		mockSetup func()
		wantLen   int
		wantTopic string
		wantErr   bool
	}{
		{
			name: "Successful update",
			payload: &userevents.UpdateUDiscIdentityRequestedPayloadV1{
				GuildID:  testGuildID,
				UserID:   testUserID,
				Username: &testUsername,
				Name:     &testName,
			},
			mockSetup: func() {
				mockUserService.EXPECT().UpdateUDiscIdentity(gomock.Any(), testGuildID, testUserID, &testUsername, &testName).
					Return(results.OperationResult{
						Success: &userevents.UDiscIdentityUpdatedPayloadV1{
							UserID: testUserID,
						},
					}, nil)
			},
			wantLen:   1,
			wantTopic: userevents.UDiscIdentityUpdatedV1,
			wantErr:   false,
		},
		{
			name: "Update failed (business logic failure)",
			payload: &userevents.UpdateUDiscIdentityRequestedPayloadV1{
				GuildID:  testGuildID,
				UserID:   testUserID,
				Username: &testUsername,
				Name:     &testName,
			},
			mockSetup: func() {
				mockUserService.EXPECT().UpdateUDiscIdentity(gomock.Any(), testGuildID, testUserID, &testUsername, &testName).
					Return(results.OperationResult{
						Failure: &userevents.UDiscIdentityUpdateFailedPayloadV1{
							GuildID: testGuildID,
							UserID:  testUserID,
							Reason:  "some error",
						},
					}, nil)
			},
			wantLen:   1,
			wantTopic: userevents.UDiscIdentityUpdateFailedV1,
			wantErr:   false,
		},
		{
			name: "Unexpected result (nil success and failure)",
			payload: &userevents.UpdateUDiscIdentityRequestedPayloadV1{
				GuildID:  testGuildID,
				UserID:   testUserID,
				Username: &testUsername,
				Name:     &testName,
			},
			mockSetup: func() {
				mockUserService.EXPECT().UpdateUDiscIdentity(gomock.Any(), testGuildID, testUserID, &testUsername, &testName).
					Return(results.OperationResult{}, nil)
			},
			wantLen: 0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			h := NewUserHandlers(mockUserService, logger, tracer, nil, metrics)
			results, err := h.HandleUpdateUDiscIdentityRequest(context.Background(), tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUpdateUDiscIdentityRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(results) != tt.wantLen {
					t.Errorf("HandleUpdateUDiscIdentityRequest() got %d results, want %d", len(results), tt.wantLen)
					return
				}
				if tt.wantLen > 0 && results[0].Topic != tt.wantTopic {
					t.Errorf("HandleUpdateUDiscIdentityRequest() got topic %s, want %s", results[0].Topic, tt.wantTopic)
				}
			}
		})
	}
}
