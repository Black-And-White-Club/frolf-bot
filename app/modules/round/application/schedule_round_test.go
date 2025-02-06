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
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestRoundService_ScheduleRoundEvents(t *testing.T) {
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
			name: "Successful scheduling of round events",
			args: args{
				ctx: context.Background(),
				msg: message.NewMessage(watermill.NewUUID(), func() []byte {
					payload, _ := json.Marshal(roundevents.RoundStoredPayload{
						Round: roundtypes.Round{
							ID:        "some-uuid",
							Title:     "Test Round",
							Location:  "Test Location",
							StartTime: time.Now().Add(2 * time.Hour),
						},
					})
					return payload
				}()),
			},
			wantErr: false,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(roundevents.DelayedMessagesSubject, gomock.Any()).Return(nil).Times(2)
				mockEventBus.EXPECT().Publish(roundevents.RoundScheduled, gomock.Any()).Return(nil).Times(1)
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
			name: "Failed to schedule 1-hour reminder",
			args: args{
				ctx: context.Background(),
				msg: message.NewMessage(watermill.NewUUID(), func() []byte {
					payload, _ := json.Marshal(roundevents.RoundStoredPayload{
						Round: roundtypes.Round{
							ID:        "some-uuid",
							Title:     "Test Round",
							Location:  "Test Location",
							StartTime: time.Now().Add(2 * time.Hour),
						},
					})
					return payload
				}()),
			},
			wantErr: true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(roundevents.DelayedMessagesSubject, gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
			},
		},
		{
			name: "Failed to schedule round start",
			args: args{
				ctx: context.Background(),
				msg: message.NewMessage(watermill.NewUUID(), func() []byte {
					payload, _ := json.Marshal(roundevents.RoundStoredPayload{
						Round: roundtypes.Round{
							ID:        "some-uuid",
							Title:     "Test Round",
							Location:  "Test Location",
							StartTime: time.Now().Add(2 * time.Hour),
						},
					})
					return payload
				}()),
			},
			wantErr: true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(roundevents.DelayedMessagesSubject, gomock.Any()).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(roundevents.DelayedMessagesSubject, gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
			},
		},
		{
			name: "Failed to publish final event",
			args: args{
				ctx: context.Background(),
				msg: message.NewMessage(watermill.NewUUID(), func() []byte {
					payload, _ := json.Marshal(roundevents.RoundStoredPayload{
						Round: roundtypes.Round{
							ID:        "some-uuid",
							Title:     "Test Round",
							Location:  "Test Location",
							StartTime: time.Now().Add(2 * time.Hour),
						},
					})
					return payload
				}()),
			},
			wantErr: true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(roundevents.DelayedMessagesSubject, gomock.Any()).Return(nil).Times(2)
				mockEventBus.EXPECT().Publish(roundevents.RoundScheduled, gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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

			if err := s.ScheduleRoundEvents(tt.args.ctx, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("RoundService.ScheduleRoundEvents() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
