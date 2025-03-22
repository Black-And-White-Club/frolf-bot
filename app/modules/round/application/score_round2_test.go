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
	mockRoundDB := rounddb.NewMockRoundDBInterface(ctrl)
	logger := slog.Default()

	// Create a valid EventMessageID instance
	eventMessageID := roundtypes.EventMessageID("discord-message-id")

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
					RoundID:     1,
					Participant: "some-discord-id",
					Score:       10,
				},
			},
			expectedEvent: roundevents.RoundAllScoresSubmitted,
			expectErr:     false,
			mockExpects: func() {
				score1 := 10
				score2 := 20
				mockRoundDB.EXPECT().GetParticipants(gomock.Any(), roundtypes.ID(1)).Return([]roundtypes.Participant{
					{
						UserID: "user1",
						Score:  &score1,
					},
					{
						UserID: "user2",
						Score:  &score2,
					},
				}, nil).Times(1)

				// Add expectation for GetEventMessageID
				var msgID roundtypes.EventMessageID = "discord-message-id"
				mockRoundDB.EXPECT().GetEventMessageID(gomock.Any(), roundtypes.ID(1)).Return(&msgID, nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundAllScoresSubmitted), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundAllScoresSubmitted {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.AllScoresSubmittedPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					if payload.RoundID != 1 {
						return fmt.Errorf("unexpected round ID: %v", payload.RoundID)
					}

					// Check EventMessageID with proper type comparison
					if string(*payload.EventMessageID) != string(eventMessageID) {
						return fmt.Errorf("unexpected EventMessageID: %v", payload.EventMessageID)
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
					RoundID:     1,
					Participant: "some-discord-id",
					Score:       10,
				},
			},
			expectErr: false,
			mockExpects: func() {
				score1 := 10
				mockRoundDB.EXPECT().GetParticipants(gomock.Any(), roundtypes.ID(1)).Return([]roundtypes.Participant{
					{
						UserID: "user1",
						Score:  &score1,
					},
					{
						UserID: "user2",
						Score:  nil, // Score not submitted
					},
				}, nil).Times(1)

				// Add expectation for GetEventMessageID
				var msgID roundtypes.EventMessageID = "discord-message-id"
				mockRoundDB.EXPECT().GetEventMessageID(gomock.Any(), roundtypes.ID(1)).Return(&msgID, nil).Times(1)
				// Add expectation for NotAllScoresSubmitted event
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundNotAllScoresSubmitted), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundNotAllScoresSubmitted {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.ParticipantScoreUpdatedPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					if payload.RoundID != 1 {
						return fmt.Errorf("unexpected round ID: %v", payload.RoundID)
					}

					if payload.Participant != "some-discord-id" {
						return fmt.Errorf("unexpected participant: %v", payload.Participant)
					}

					if payload.Score != 10 {
						return fmt.Errorf("unexpected score: %v", payload.Score)
					}

					// Check EventMessageID with proper type comparison
					if string(*payload.EventMessageID) != string(eventMessageID) {
						return fmt.Errorf("unexpected EventMessageID: %v", payload.EventMessageID)
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
			expectErr: true,
			mockExpects: func() {
				// No mock expectations needed as the function should fail before any DB or EventBus calls
			},
		},
		{
			name: "Database error in GetParticipants",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ParticipantScoreUpdatedPayload{
					RoundID:     1,
					Participant: "some-discord-id",
					Score:       10,
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().GetParticipants(gomock.Any(), roundtypes.ID(1)).Return(nil, fmt.Errorf("db error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundScoreUpdateError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Database error in GetEventMessageID",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ParticipantScoreUpdatedPayload{
					RoundID:     1,
					Participant: "some-discord-id",
					Score:       10,
				},
			},
			expectErr: true,
			mockExpects: func() {
				score1 := 10
				score2 := 20
				mockRoundDB.EXPECT().GetParticipants(gomock.Any(), roundtypes.ID(1)).Return([]roundtypes.Participant{
					{
						UserID: "user1",
						Score:  &score1,
					},
					{
						UserID: "user2",
						Score:  &score2,
					},
				}, nil).Times(1)

				// Mock GetEventMessageID to return error
				mockRoundDB.EXPECT().GetEventMessageID(gomock.Any(), roundtypes.ID(1)).Return(nil, fmt.Errorf("failed to get message ID")).Times(1)
			},
		},
		{
			name: "Publish AllScoresSubmitted event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ParticipantScoreUpdatedPayload{
					RoundID:     1,
					Participant: "some-discord-id",
					Score:       10,
				},
			},
			expectErr: true,
			mockExpects: func() {
				score1 := 10
				score2 := 20
				mockRoundDB.EXPECT().GetParticipants(gomock.Any(), roundtypes.ID(1)).Return([]roundtypes.Participant{
					{
						UserID: "user1",
						Score:  &score1,
					},
					{
						UserID: "user2",
						Score:  &score2,
					},
				}, nil).Times(1)

				// Add expectation for GetEventMessageID
				var msgID roundtypes.EventMessageID = "discord-message-id"
				mockRoundDB.EXPECT().GetEventMessageID(gomock.Any(), roundtypes.ID(1)).Return(&msgID, nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundAllScoresSubmitted), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
			},
		},
		{
			name: "Publish NotAllScoresSubmitted event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ParticipantScoreUpdatedPayload{
					RoundID:     1,
					Participant: "some-discord-id",
					Score:       10,
				},
			},
			expectErr: true,
			mockExpects: func() {
				score1 := 10
				mockRoundDB.EXPECT().GetParticipants(gomock.Any(), roundtypes.ID(1)).Return([]roundtypes.Participant{
					{
						UserID: "user1",
						Score:  &score1,
					},
					{
						UserID: "user2",
						Score:  nil, // Score not submitted
					},
				}, nil).Times(1)

				// Add expectation for GetEventMessageID
				var msgID roundtypes.EventMessageID = "discord-message-id"
				mockRoundDB.EXPECT().GetEventMessageID(gomock.Any(), roundtypes.ID(1)).Return(&msgID, nil).Times(1)
				// Mock publish to return error
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundNotAllScoresSubmitted), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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
