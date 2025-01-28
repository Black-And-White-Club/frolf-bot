package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	roundtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/round/domain/types"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestRoundService_StoreRound(t *testing.T) {
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
			name: "Successful round storage",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundEntityCreatedPayload{
					Round: roundtypes.Round{
						ID:        "some-uuid",
						Title:     "Valid Title",
						StartTime: time.Date(2023, 12, 25, 10, 0, 0, 0, time.UTC),
					},
				},
			},
			expectedEvent: roundevents.RoundStored,
			expectErr:     false,
			mockExpects: func() {
				mockRoundDB.EXPECT().CreateRound(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundStored), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Invalid payload",
			args: args{
				ctx:     context.Background(),
				payload: "invalid json",
			},
			expectedEvent: roundevents.RoundError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Database error",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundEntityCreatedPayload{
					Round: roundtypes.Round{
						ID:        "some-uuid",
						Title:     "Valid Title",
						StartTime: time.Date(2023, 12, 25, 10, 0, 0, 0, time.UTC),
					},
				},
			},
			expectedEvent: roundevents.RoundError,
			expectErr:     true,
			mockExpects: func() {
				mockRoundDB.EXPECT().CreateRound(gomock.Any(), gomock.Any()).Return(fmt.Errorf("db error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish RoundStored event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundEntityCreatedPayload{
					Round: roundtypes.Round{
						ID:        "some-uuid",
						Title:     "Valid Title",
						StartTime: time.Date(2023, 12, 25, 10, 0, 0, 0, time.UTC),
					},
				},
			},
			expectedEvent: "",
			expectErr:     true,
			mockExpects: func() {
				mockRoundDB.EXPECT().CreateRound(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundStored), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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
				RoundDB:   mockRoundDB,
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			}

			// Call the service function
			err := s.StoreRound(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("StoreRound() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("StoreRound() unexpected error: %v", err)
				}
			}
		})
	}
}

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
					ID:        "some-uuid",
					Title:     "Valid Title",
					StartTime: time.Date(2023, 12, 25, 10, 0, 0, 0, time.UTC),
				}, nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundCreated), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Invalid payload",
			args: args{
				ctx:     context.Background(),
				payload: "invalid json",
			},
			expectedEvent: roundevents.RoundError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Database error",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundScheduledPayload{
					RoundID: "some-uuid",
				},
			},
			expectedEvent: roundevents.RoundError,
			expectErr:     true,
			mockExpects: func() {
				mockRoundDB.EXPECT().GetRound(gomock.Any(), gomock.Eq("some-uuid")).Return(nil, fmt.Errorf("db error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundError), gomock.Any()).Return(nil).Times(1)
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
					ID:        "some-uuid",
					Title:     "Valid Title",
					StartTime: time.Date(2023, 12, 25, 10, 0, 0, 0, time.UTC),
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
				RoundDB:   mockRoundDB,
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
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
