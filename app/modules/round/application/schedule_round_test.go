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
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestRoundService_ScheduleRoundEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.Default()

	type args struct {
		ctx     context.Context
		payload interface{}
	}
	tests := []struct {
		name        string
		args        args
		expectErr   bool
		mockExpects func(*eventbusmocks.MockEventBus)
	}{
		{
			name: "Successful round events scheduling",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundStoredPayload{
					Round: roundtypes.Round{
						ID:        "some-round-id",
						Title:     "Test Round",
						Location:  "Test Location",
						StartTime: time.Now().Add(2 * time.Hour), // Start time is more than an hour away
						State:     roundtypes.RoundStateUpcoming,
					},
				},
			},
			expectErr: false,
			mockExpects: func(mockEventBus *eventbusmocks.MockEventBus) {
				// Expect the one-hour reminder event
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.DelayedMessagesSubject), gomock.Any()).
					DoAndReturn(func(topic string, msg *message.Message) error {
						if topic != roundevents.DelayedMessagesSubject {
							return fmt.Errorf("unexpected topic: %s", topic)
						}
						var payload roundevents.RoundReminderPayload
						if err := json.Unmarshal(msg.Payload, &payload); err != nil {
							return fmt.Errorf("failed to unmarshal payload: %w", err)
						}
						if payload.RoundID != "some-round-id" {
							return fmt.Errorf("unexpected round ID: %s", payload.RoundID)
						}
						if payload.ReminderType != "1h" {
							return fmt.Errorf("unexpected reminder type: %s", payload.ReminderType)
						}
						return nil
					}).
					Times(1)

				// Expect the round start event
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.DelayedMessagesSubject), gomock.Any()).
					DoAndReturn(func(topic string, msg *message.Message) error {
						if topic != roundevents.DelayedMessagesSubject {
							return fmt.Errorf("unexpected topic: %s", topic)
						}
						var payload roundevents.RoundStartedPayload
						if err := json.Unmarshal(msg.Payload, &payload); err != nil {
							return fmt.Errorf("failed to unmarshal payload: %w", err)
						}
						if payload.RoundID != "some-round-id" {
							return fmt.Errorf("unexpected round ID: %s", payload.RoundID)
						}
						return nil
					}).
					Times(1)

				// Expect the round scheduled event
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundScheduled), gomock.Any()).
					DoAndReturn(func(topic string, msg *message.Message) error {
						if topic != roundevents.RoundScheduled {
							return fmt.Errorf("unexpected topic: %s", topic)
						}
						var payload roundevents.RoundStoredPayload
						if err := json.Unmarshal(msg.Payload, &payload); err != nil {
							return fmt.Errorf("failed to unmarshal payload: %w", err)
						}
						if payload.Round.ID != "some-round-id" {
							return fmt.Errorf("unexpected round ID: %s", payload.Round.ID)
						}
						return nil
					}).
					Times(1)
			},
		},
		{
			name: "Invalid payload",
			args: args{
				ctx:     context.Background(),
				payload: "invalid json",
			},
			expectErr: true,
			mockExpects: func(mockEventBus *eventbusmocks.MockEventBus) {
				// No expectations for invalid payload
			},
		},
		{
			name: "Failed to publish reminder",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundStoredPayload{
					Round: roundtypes.Round{
						ID:        "some-round-id",
						Title:     "Test Round",
						Location:  "Test Location",
						StartTime: time.Now().Add(2 * time.Hour),
						State:     roundtypes.RoundStateUpcoming,
					},
				},
			},
			expectErr: true,
			mockExpects: func(mockEventBus *eventbusmocks.MockEventBus) {
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.DelayedMessagesSubject), gomock.Any()).
					Return(fmt.Errorf("publish error")).
					Times(1)
			},
		},
		{
			name: "Failed to publish round start",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundStoredPayload{
					Round: roundtypes.Round{
						ID:        "some-round-id",
						Title:     "Test Round",
						Location:  "Test Location",
						StartTime: time.Now().Add(2 * time.Hour),
						State:     roundtypes.RoundStateUpcoming,
					},
				},
			},
			expectErr: true,
			mockExpects: func(mockEventBus *eventbusmocks.MockEventBus) {
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.DelayedMessagesSubject), gomock.Any()).
					Return(nil).
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.DelayedMessagesSubject), gomock.Any()).
					Return(fmt.Errorf("publish error")).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare a mock message with the payload
			payloadBytes, _ := json.Marshal(tt.args.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, watermill.NewUUID())

			tt.mockExpects(mockEventBus)

			s := &RoundService{
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			}

			// Call the service function
			err := s.ScheduleRoundEvents(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("ScheduleRoundEvents() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("ScheduleRoundEvents() unexpected error: %v", err)
				}
			}
		})
	}
}
