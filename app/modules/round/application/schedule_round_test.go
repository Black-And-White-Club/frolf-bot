package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"

	eventbusmocks "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	roundtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/types"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
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
		name          string
		args          args
		expectedEvent string
		expectErr     bool
		mockExpects   func(t *testing.T)
	}{
		{
			name: "Successful round events scheduling",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundStoredPayload{
					Round: roundtypes.Round{
						ID:        "some-round-id",
						Title:     "Test Round",
						StartTime: time.Now().Add(time.Hour + time.Minute), // Start time is more than an hour away
						State:     roundtypes.RoundStateUpcoming,
					},
				},
			},
			expectedEvent: roundevents.RoundScheduled,
			expectErr:     false,
			mockExpects: func(t *testing.T) {
				// Expect the one-hour reminder event
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundReminder), gomock.Any()).
					DoAndReturn(func(topic string, msg *message.Message) error {
						if topic != roundevents.RoundReminder {
							return fmt.Errorf("unexpected topic: %s", topic)
						}
						var payload roundevents.RoundReminderPayload
						if err := json.Unmarshal(msg.Payload, &payload); err != nil {
							return fmt.Errorf("failed to unmarshal payload: %w", err)
						}
						if payload.RoundID != "some-round-id" {
							return fmt.Errorf("unexpected round ID: %s", payload.RoundID)
						}
						if payload.ReminderType != "one_hour" {
							return fmt.Errorf("unexpected reminder type: %s", payload.ReminderType)
						}
						return nil
					}).
					MinTimes(0)

				// Expect the 30-minute reminder event
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundReminder), gomock.Any()).
					DoAndReturn(func(topic string, msg *message.Message) error {
						if topic != roundevents.RoundReminder {
							return fmt.Errorf("unexpected topic: %s", topic)
						}
						var payload roundevents.RoundReminderPayload
						if err := json.Unmarshal(msg.Payload, &payload); err != nil {
							return fmt.Errorf("failed to unmarshal payload: %w", err)
						}
						if payload.RoundID != "some-round-id" {
							return fmt.Errorf("unexpected round ID: %s", payload.RoundID)
						}
						if payload.ReminderType != "thirty_minutes" {
							return fmt.Errorf("unexpected reminder type: %s", payload.ReminderType)
						}
						return nil
					}).
					MinTimes(0)

				// Expect the round start event
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundStarted), gomock.Any()).
					DoAndReturn(func(topic string, msg *message.Message) error {
						if topic != roundevents.RoundStarted {
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
					MinTimes(0)

				// Expect the round scheduled event
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundScheduled), gomock.Any()).
					DoAndReturn(func(topic string, msg *message.Message) error {
						if topic != roundevents.RoundScheduled {
							return fmt.Errorf("unexpected topic: %s", topic)
						}
						var payload roundevents.RoundScheduledPayload
						if err := json.Unmarshal(msg.Payload, &payload); err != nil {
							return fmt.Errorf("failed to unmarshal payload: %w", err)
						}
						if payload.RoundID != "some-round-id" {
							return fmt.Errorf("unexpected round ID: %s", payload.RoundID)
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
			mockExpects: func(t *testing.T) {
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare a mock message with the payload
			payloadBytes, _ := json.Marshal(tt.args.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, watermill.NewUUID())

			tt.mockExpects(t)

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
