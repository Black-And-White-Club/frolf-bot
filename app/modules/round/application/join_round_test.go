package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestRoundService_ValidateParticipantJoinRequest(t *testing.T) {
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
			name: "Successful participant join request validation",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ParticipantJoinRequestPayload{
					RoundID: 1,
					UserID:  "some-discord-id",
				},
			},
			expectedEvent: roundevents.RoundParticipantJoinValidated,
			expectErr:     false,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundParticipantJoinValidated), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Invalid payload",
			args: args{
				ctx:     context.Background(),
				payload: "invalid json",
			},
			expectedEvent: roundevents.RoundParticipantJoinError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundParticipantJoinError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Empty round ID",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ParticipantJoinRequestPayload{
					UserID: "some-discord-id",
				},
			},
			expectedEvent: roundevents.RoundParticipantJoinError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundParticipantJoinError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Empty participant Discord ID",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ParticipantJoinRequestPayload{
					RoundID: 1,
				},
			},
			expectedEvent: roundevents.RoundParticipantJoinError,
			expectErr:     true,
			mockExpects: func() {
			},
		},
		{
			name: "Publish RoundParticipantJoinValidated event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ParticipantJoinRequestPayload{
					RoundID: 1,
					UserID:  "some-discord-id",
				},
			},
			expectedEvent: "",
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundParticipantJoinValidated), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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
			err := s.ValidateParticipantJoinRequest(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("ValidateParticipantJoinRequest() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateParticipantJoinRequest() unexpected error: %v", err)
				}
			}
		})
	}
}
