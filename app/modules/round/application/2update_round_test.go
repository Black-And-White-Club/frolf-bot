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

func TestRoundService_StoreRoundUpdate(t *testing.T) {
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
			name: "Successful round update storage",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundEntityUpdatedPayload{
					Round: roundtypes.Round{
						ID:        "some-round-id",
						Title:     "Updated Title",
						Location:  "Updated Location",
						EventType: func() *string { s := "Updated Type"; return &s }(),
						StartTime: func() *time.Time { t := time.Now(); return &t }(),
						State:     roundtypes.RoundStateUpcoming,
					},
				},
			},
			expectedEvent: roundevents.RoundUpdated,
			expectErr:     false,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateRound(gomock.Any(), "some-round-id", gomock.Any()).DoAndReturn(
					func(ctx context.Context, roundID string, round *roundtypes.Round) error {
						if round.Title != "Updated Title" {
							return fmt.Errorf("unexpected title: %s", round.Title)
						}
						if round.Location != "Updated Location" {
							return fmt.Errorf("unexpected location: %s", round.Location)
						}
						if *round.EventType != "Updated Type" {
							return fmt.Errorf("unexpected event type: %s", *round.EventType)
						}
						return nil
					},
				).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUpdated), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundUpdated {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.RoundUpdatedPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					if payload.RoundID != "some-round-id" {
						return fmt.Errorf("unexpected round ID: %s", payload.RoundID)
					}

					return nil
				}).Times(1)
			},
		},
		{
			name: "Invalid payload",
			args: args{
				ctx:     context.Background(),
				payload: "invalid json",
			},
			expectedEvent: roundevents.RoundUpdateError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUpdateError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Database error",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundEntityUpdatedPayload{
					Round: roundtypes.Round{
						ID:        "some-round-id",
						Title:     "Updated Title",
						Location:  "Updated Location",
						EventType: func() *string { s := "Updated Type"; return &s }(),
						StartTime: func() *time.Time { t := time.Now(); return &t }(),
						State:     roundtypes.RoundStateUpcoming,
					},
				},
			},
			expectedEvent: roundevents.RoundUpdateError,
			expectErr:     true,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateRound(gomock.Any(), "some-round-id", gomock.Any()).Return(fmt.Errorf("db error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUpdateError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish RoundUpdated event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundEntityUpdatedPayload{
					Round: roundtypes.Round{
						ID:        "some-round-id",
						Title:     "Updated Title",
						Location:  "Updated Location",
						EventType: func() *string { s := "Updated Type"; return &s }(),
						StartTime: func() *time.Time { t := time.Now(); return &t }(),
						State:     roundtypes.RoundStateUpcoming,
					},
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateRound(gomock.Any(), "some-round-id", gomock.Any()).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundUpdated), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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
			err := s.StoreRoundUpdate(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("StoreRoundUpdate() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("StoreRoundUpdate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRoundService_UpdateScheduledRoundEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDB(ctrl)
	logger := slog.Default()

	type args struct {
		ctx context.Context
		msg *message.Message
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		mockExpects func()
	}{
		{
			name: "Successful update of scheduled round events",
			args: args{
				ctx: context.Background(),
				msg: message.NewMessage(watermill.NewUUID(), func() []byte {
					payload, _ := json.Marshal(roundevents.RoundUpdatedPayload{
						RoundID: "some-round-id",
					})
					return payload
				}()),
			},
			wantErr: false,
			mockExpects: func() {
				mockEventBus.EXPECT().CancelScheduledMessage(gomock.Any(), "some-round-id").Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundScheduleUpdate), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Invalid payload",
			args: args{
				ctx: context.Background(),
				msg: message.NewMessage(watermill.NewUUID(), []byte("invalid json")),
			},
			wantErr:     true,
			mockExpects: func() {},
		},
		{
			name: "Failed to cancel existing scheduled events",
			args: args{
				ctx: context.Background(),
				msg: message.NewMessage(watermill.NewUUID(), func() []byte {
					payload, _ := json.Marshal(roundevents.RoundUpdatedPayload{
						RoundID: "some-round-id",
					})
					return payload
				}()),
			},
			wantErr: true,
			mockExpects: func() {
				mockEventBus.EXPECT().CancelScheduledMessage(gomock.Any(), "some-round-id").Return(fmt.Errorf("cancel error")).Times(1)
			},
		},
		{
			name: "Failed to publish round.schedule.update event",
			args: args{
				ctx: context.Background(),
				msg: message.NewMessage(watermill.NewUUID(), func() []byte {
					payload, _ := json.Marshal(roundevents.RoundUpdatedPayload{
						RoundID: "some-round-id",
					})
					return payload
				}()),
			},
			wantErr: true,
			mockExpects: func() {
				mockEventBus.EXPECT().CancelScheduledMessage(gomock.Any(), "some-round-id").Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundScheduleUpdate), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockExpects()

			s := &RoundService{
				RoundDB:  mockRoundDB,
				EventBus: mockEventBus,
				logger:   logger,
			}

			if err := s.UpdateScheduledRoundEvents(tt.args.ctx, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("RoundService.UpdateScheduledRoundEvents() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
