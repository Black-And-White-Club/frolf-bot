package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	roundtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/round/domain/types"
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
	mockRoundDB := rounddb.NewMockRoundDB(ctrl)
	logger := slog.Default()

	type args struct {
		ctx     context.Context
		payload interface{}
	}
	tests := []struct {
		name          string
		args          args
		expectedEvent string
		expectErr     bool
		mockExpects   func()
	}{
		{
			name: "Successful round created event publishing",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundScheduledPayload{
					RoundID: "some-uuid",
				},
			},
			expectedEvent: roundevents.RoundCreated,
			expectErr:     false,
			mockExpects: func() {
				mockRoundDB.EXPECT().GetRound(gomock.Any(), gomock.Eq("some-uuid")).Return(&roundtypes.Round{
					ID: "some-uuid",
				}, nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundCreated), gomock.Any()).Times(1)
			},
		},
		{
			name: "Invalid payload",
			args: args{
				ctx:     context.Background(),
				payload: "invalid json",
			},
			expectedEvent: "",
			expectErr:     true,
			mockExpects:   func() {},
		},
		{
			name: "Database error",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundScheduledPayload{
					RoundID: "some-uuid",
				},
			},
			expectedEvent: "",
			expectErr:     true,
			mockExpects: func() {
				mockRoundDB.EXPECT().GetRound(gomock.Any(), gomock.Eq("some-uuid")).Return(nil, fmt.Errorf("database error")).Times(1)
			},
		},
		{
			name: "Publish RoundCreated event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundScheduledPayload{
					RoundID: "some-uuid",
				},
			},
			expectedEvent: "",
			expectErr:     true,
			mockExpects: func() {
				mockRoundDB.EXPECT().GetRound(gomock.Any(), gomock.Eq("some-uuid")).Return(&roundtypes.Round{
					ID: "some-uuid",
				}, nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundCreated), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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
			err := s.PublishRoundCreated(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("PublishRoundCreated() expected error, got none")
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
	mockRoundDB := rounddb.NewMockRoundDB(ctrl)
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
					RoundID:        "some-uuid",
					DiscordEventID: "discord-event-id",
				},
			},
			expectErr: false,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateDiscordEventID(gomock.Any(), gomock.Eq("some-uuid"), gomock.Eq("discord-event-id")).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDiscordEventIDUpdated), gomock.Any()).Times(1)
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
					RoundID:        "some-uuid",
					DiscordEventID: "discord-event-id",
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateDiscordEventID(gomock.Any(), gomock.Eq("some-uuid"), gomock.Eq("discord-event-id")).Return(fmt.Errorf("database error")).Times(1)
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
			err := s.UpdateDiscordEventID(tt.args.ctx, msg)
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
