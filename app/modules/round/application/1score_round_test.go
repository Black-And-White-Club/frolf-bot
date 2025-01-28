package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	eventbusmocks "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestRoundService_ValidateScoreUpdateRequest(t *testing.T) {
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
			name: "Successful score update request validation",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ScoreUpdateRequestPayload{
					RoundID:     "some-round-id",
					Participant: "some-discord-id",
					Score:       func() *int { i := 10; return &i }(),
				},
			},
			expectedEvent: roundevents.RoundScoreUpdateValidated,
			expectErr:     false,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundScoreUpdateValidated), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundScoreUpdateValidated {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.ScoreUpdateValidatedPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					if payload.ScoreUpdateRequestPayload.RoundID != "some-round-id" {
						return fmt.Errorf("unexpected round ID: %s", payload.ScoreUpdateRequestPayload.RoundID)
					}

					if payload.ScoreUpdateRequestPayload.Participant != "some-discord-id" {
						return fmt.Errorf("unexpected participant ID: %s", payload.ScoreUpdateRequestPayload.Participant)
					}

					if *payload.ScoreUpdateRequestPayload.Score != 10 {
						return fmt.Errorf("unexpected score: %d", *payload.ScoreUpdateRequestPayload.Score)
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
			expectedEvent: roundevents.RoundScoreUpdateError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundScoreUpdateError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Empty round ID",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ScoreUpdateRequestPayload{
					Participant: "some-discord-id",
					Score:       func() *int { i := 10; return &i }(),
				},
			},
			expectedEvent: roundevents.RoundScoreUpdateError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundScoreUpdateError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Empty participant ID",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ScoreUpdateRequestPayload{
					RoundID: "some-round-id",
					Score:   func() *int { i := 10; return &i }(),
				},
			},
			expectedEvent: roundevents.RoundScoreUpdateError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundScoreUpdateError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Nil score",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ScoreUpdateRequestPayload{
					RoundID:     "some-round-id",
					Participant: "some-discord-id",
					Score:       nil,
				},
			},
			expectedEvent: roundevents.RoundScoreUpdateError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundScoreUpdateError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish RoundScoreUpdateValidated event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ScoreUpdateRequestPayload{
					RoundID:     "some-round-id",
					Participant: "some-discord-id",
					Score:       func() *int { i := 10; return &i }(),
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundScoreUpdateValidated), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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
			err := s.ValidateScoreUpdateRequest(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("ValidateScoreUpdateRequest() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateScoreUpdateRequest() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRoundService_UpdateParticipantScore(t *testing.T) {
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
			name: "Successful participant score update",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ScoreUpdateValidatedPayload{
					ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
						RoundID:     "some-round-id",
						Participant: "some-discord-id",
						Score:       func() *int { i := 10; return &i }(),
					},
				},
			},
			expectedEvent: roundevents.RoundParticipantScoreUpdated,
			expectErr:     false,
			mockExpects: func() {
				score := 10
				mockRoundDB.EXPECT().UpdateParticipantScore(gomock.Any(), "some-round-id", "some-discord-id", score).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundParticipantScoreUpdated), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					if topic != roundevents.RoundParticipantScoreUpdated {
						return fmt.Errorf("unexpected topic: %s", topic)
					}

					var payload roundevents.ParticipantScoreUpdatedPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					if payload.RoundID != "some-round-id" {
						return fmt.Errorf("unexpected round ID: %s", payload.RoundID)
					}

					if payload.Participant != "some-discord-id" {
						return fmt.Errorf("unexpected participant ID: %s", payload.Participant)
					}

					if payload.Score != 10 {
						return fmt.Errorf("unexpected score: %d", payload.Score)
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
			expectedEvent: roundevents.RoundScoreUpdateError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundScoreUpdateError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Database error",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ScoreUpdateValidatedPayload{
					ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
						RoundID:     "some-round-id",
						Participant: "some-discord-id",
						Score:       func() *int { i := 10; return &i }(),
					},
				},
			},
			expectedEvent: roundevents.RoundScoreUpdateError,
			expectErr:     true,
			mockExpects: func() {
				score := 10
				mockRoundDB.EXPECT().UpdateParticipantScore(gomock.Any(), "some-round-id", "some-discord-id", score).Return(fmt.Errorf("db error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundScoreUpdateError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish RoundParticipantScoreUpdated event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.ScoreUpdateValidatedPayload{
					ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
						RoundID:     "some-round-id",
						Participant: "some-discord-id",
						Score:       func() *int { i := 10; return &i }(),
					},
				},
			},
			expectErr: true,
			mockExpects: func() {
				score := 10
				mockRoundDB.EXPECT().UpdateParticipantScore(gomock.Any(), "some-round-id", "some-discord-id", score).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundParticipantScoreUpdated), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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
			err := s.UpdateParticipantScore(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("UpdateParticipantScore() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("UpdateParticipantScore() unexpected error: %v", err)
				}
			}
		})
	}
}
