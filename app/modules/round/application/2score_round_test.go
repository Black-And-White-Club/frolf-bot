package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestRoundService_CheckAllScoresSubmitted(t *testing.T) {
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
			name: "All scores submitted",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ParticipantScoreUpdatedPayload{
					RoundID:     "some-round-id",
					Participant: "some-discord-id",
					Score:       10,
				},
			},
			expectedEvent: roundevents.RoundAllScoresSubmitted,
			expectErr:     false,
			mockExpects: func() {
				score1 := 10
				score2 := 20
				mockRoundDB.EXPECT().GetParticipants(gomock.Any(), "some-round-id").Return([]roundtypes.RoundParticipant{
					{
						DiscordID: "user1",
						Score:     &score1,
					},
					{
						DiscordID: "user2",
						Score:     &score2,
					},
				}, nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundAllScoresSubmitted), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundAllScoresSubmitted {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.AllScoresSubmittedPayload
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
			name: "Not all scores submitted",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ParticipantScoreUpdatedPayload{
					RoundID:     "some-round-id",
					Participant: "some-discord-id",
					Score:       10,
				},
			},
			expectErr: false,
			mockExpects: func() {
				score1 := 10
				mockRoundDB.EXPECT().GetParticipants(gomock.Any(), "some-round-id").Return([]roundtypes.RoundParticipant{
					{
						DiscordID: "user1",
						Score:     &score1,
					},
					{
						DiscordID: "user2",
						Score:     nil, // Score not submitted
					},
				}, nil).Times(1)
			},
		},
		{
			name: "Invalid payload",
			args: args{
				ctx:     context.Background(),
				payload: "invalid json",
			},
			expectErr: true,
			mockExpects: func() {
			},
		},
		{
			name: "Database error",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ParticipantScoreUpdatedPayload{
					RoundID:     "some-round-id",
					Participant: "some-discord-id",
					Score:       10,
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().GetParticipants(gomock.Any(), "some-round-id").Return(nil, fmt.Errorf("db error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundScoreUpdateError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish AllScoresSubmitted event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ParticipantScoreUpdatedPayload{
					RoundID:     "some-round-id",
					Participant: "some-discord-id",
					Score:       10,
				},
			},
			expectErr: true,
			mockExpects: func() {
				score1 := 10
				score2 := 20
				mockRoundDB.EXPECT().GetParticipants(gomock.Any(), "some-round-id").Return([]roundtypes.RoundParticipant{
					{
						DiscordID: "user1",
						Score:     &score1,
					},
					{
						DiscordID: "user2",
						Score:     &score2,
					},
				}, nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundAllScoresSubmitted), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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
			err := s.CheckAllScoresSubmitted(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("CheckAllScoresSubmitted() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("CheckAllScoresSubmitted() unexpected error: %v", err)
				}
			}
		})
	}
}
