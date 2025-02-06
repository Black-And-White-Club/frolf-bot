package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestRoundService_ValidateRoundDeleteRequest(t *testing.T) {
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
		mockExpects   func()
	}{
		{
			name: "Successful round delete request validation",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundDeleteRequestPayload{
					RoundID:                 "some-uuid",
					RequestingUserDiscordID: "user-123",
				},
			},
			expectedEvent: roundevents.RoundDeleteValidated,
			expectErr:     false,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDeleteValidated), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Invalid payload",
			args: args{
				ctx:     context.Background(),
				payload: "invalid json",
			},
			expectedEvent: roundevents.RoundDeleteError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDeleteError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Empty round ID",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundDeleteRequestPayload{
					RequestingUserDiscordID: "user-123",
				},
			},
			expectedEvent: roundevents.RoundDeleteError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDeleteError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Empty requesting user's Discord ID",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundDeleteRequestPayload{
					RoundID: "some-uuid",
				},
			},
			expectedEvent: roundevents.RoundDeleteError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDeleteError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish RoundDeleteValidated event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundDeleteRequestPayload{
					RoundID:                 "some-uuid",
					RequestingUserDiscordID: "user-123",
				},
			},
			expectedEvent: "",
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDeleteValidated), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			}

			// Call the service function
			err := s.ValidateRoundDeleteRequest(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("ValidateRoundDeleteRequest() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateRoundDeleteRequest() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRoundService_DeleteRound(t *testing.T) {
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
			name: "Successful round deletion",
			args: args{
				ctx: context.Background(),
				msg: message.NewMessage(watermill.NewUUID(), func() []byte {
					payload, _ := json.Marshal(roundevents.RoundDeleteAuthorizedPayload{
						RoundID: "some-uuid",
					})
					return payload
				}()),
			},
			wantErr: false,
			mockExpects: func() {
				mockRoundDB.EXPECT().DeleteRound(gomock.Any(), "some-uuid").Return(nil).Times(1)
				mockEventBus.EXPECT().CancelScheduledMessage(gomock.Any(), "some-uuid").Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDeleted), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Invalid payload",
			args: args{
				ctx: context.Background(),
				msg: message.NewMessage(watermill.NewUUID(), []byte("invalid json")),
			},
			wantErr: true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDeleteError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Failed to delete round from database",
			args: args{
				ctx: context.Background(),
				msg: message.NewMessage(watermill.NewUUID(), func() []byte {
					payload, _ := json.Marshal(roundevents.RoundDeleteAuthorizedPayload{
						RoundID: "some-uuid",
					})
					return payload
				}()),
			},
			wantErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().DeleteRound(gomock.Any(), "some-uuid").Return(fmt.Errorf("db error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDeleteError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Failed to cancel scheduled messages",
			args: args{
				ctx: context.Background(),
				msg: message.NewMessage(watermill.NewUUID(), func() []byte {
					payload, _ := json.Marshal(roundevents.RoundDeleteAuthorizedPayload{
						RoundID: "some-uuid",
					})
					return payload
				}()),
			},
			wantErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().DeleteRound(gomock.Any(), "some-uuid").Return(nil).Times(1)
				mockEventBus.EXPECT().CancelScheduledMessage(gomock.Any(), "some-uuid").Return(fmt.Errorf("cancel error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDeleteError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Failed to publish round.deleted event",
			args: args{
				ctx: context.Background(),
				msg: message.NewMessage(watermill.NewUUID(), func() []byte {
					payload, _ := json.Marshal(roundevents.RoundDeleteAuthorizedPayload{
						RoundID: "some-uuid",
					})
					return payload
				}()),
			},
			wantErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().DeleteRound(gomock.Any(), "some-uuid").Return(nil).Times(1)
				mockEventBus.EXPECT().CancelScheduledMessage(gomock.Any(), "some-uuid").Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDeleted), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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

			if err := s.DeleteRound(tt.args.ctx, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("RoundService.DeleteRound() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
