package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestRoundService_PublishRoundCreated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDBInterface(ctrl)
	logger := slog.Default()

	s := &RoundService{
		EventBus: mockEventBus,
		RoundDB:  mockRoundDB,
		logger:   logger,
	}

	// Create a valid RoundScheduledPayload
	validScheduledPayload := roundevents.RoundScheduledPayload{
		BaseRoundPayload: roundtypes.BaseRoundPayload{
			RoundID:     validRoundID,
			Title:       validTitle,
			Description: &validDescription,
			Location:    &validLocation,
			StartTime:   roundtypes.StartTimePtr(validStartTime),
			UserID:      validUserID,
		},
		EventMessageID: nil,
	}

	tests := []struct {
		name          string
		payload       any
		mockSetup     func()
		expectedEvent string
		shouldPublish bool
		wantErr       bool
		errMsg        string
	}{
		{
			name:    "Successful publish",
			payload: validScheduledPayload,
			mockSetup: func() {
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundCreated), gomock.Any()).
					Times(1).
					Return(nil)
			},
			expectedEvent: roundevents.RoundCreated,
			shouldPublish: true,
			wantErr:       false,
		},
		{
			name:          "Invalid Payload",
			payload:       "invalid",
			mockSetup:     func() {},
			expectedEvent: "",
			shouldPublish: false,
			wantErr:       true,
			errMsg:        "invalid payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, _ := json.Marshal(tt.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)

			if tt.mockSetup != nil {
				tt.mockSetup()
			}

			err := s.PublishRoundCreated(context.Background(), msg)

			if tt.wantErr {
				if err == nil {
					t.Error("PublishRoundCreated() expected error, got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("PublishRoundCreated() error = %v, wantErrMsg containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("PublishRoundCreated() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRoundService_UpdateDiscordEventID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDBInterface(ctrl)
	logger := slog.Default()

	type args struct {
		ctx     context.Context
		payload interface{}
	}
	tests := []struct {
		name        string
		args        args
		expectErr   bool
		mockExpects func()
	}{
		{
			name: "Successful DiscordEventID update",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundEventCreatedPayload{
					RoundID:        1,
					EventMessageID: "discord-event-id",
				},
			},
			expectErr: false,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateEventMessageID(gomock.Any(), gomock.Eq(roundtypes.ID(1)), gomock.Eq("discord-event-id")).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundEventMessageIDUpdated), gomock.Any()).Times(1)
			},
		},
		{
			name: "Invalid payload",
			args: args{
				ctx:     context.Background(),
				payload: "invalid json",
			},
			expectErr:   true,
			mockExpects: func() {},
		},
		{
			name: "Database error",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundEventCreatedPayload{
					RoundID:        1,
					EventMessageID: "discord-event-id",
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateEventMessageID(gomock.Any(), gomock.Eq(roundtypes.ID(1)), gomock.Eq("discord-event-id")).Return(fmt.Errorf("database error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare a mock message with the payload
			payloadBytes, _ := json.Marshal(tt.args.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, watermill.NewUUID())

			tt.mockExpects()

			s := &RoundService{
				EventBus: mockEventBus,
				RoundDB:  mockRoundDB,
				logger:   logger,
			}

			// Call the service function
			err := s.UpdateEventMessageID(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("UpdateDiscordEventID() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("UpdateDiscordEventID() unexpected error: %v", err)
				}
			}
		})
	}
}
