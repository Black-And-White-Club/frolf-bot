package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

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

func TestRoundService_FinalizeRound(t *testing.T) {
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
			name: "Successful round finalization",
			args: args{
				ctx: context.Background(),
				payload: roundevents.AllScoresSubmittedPayload{
					RoundID: "some-uuid",
				},
			},
			expectedEvent: roundevents.RoundFinalized,
			expectErr:     false,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateRoundState(gomock.Any(), gomock.Eq("some-uuid"), gomock.Eq(roundtypes.RoundStateFinalized)).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundFinalized), gomock.Any()).Return(nil).Times(1)
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
				payload: roundevents.AllScoresSubmittedPayload{
					RoundID: "some-uuid",
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateRoundState(gomock.Any(), gomock.Eq("some-uuid"), gomock.Eq(roundtypes.RoundStateFinalized)).Return(fmt.Errorf("db error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundFinalizationError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish RoundFinalized event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.AllScoresSubmittedPayload{
					RoundID: "some-uuid",
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateRoundState(gomock.Any(), gomock.Eq("some-uuid"), gomock.Eq(roundtypes.RoundStateFinalized)).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundFinalized), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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
			err := s.FinalizeRound(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("FinalizeRound() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("FinalizeRound() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRoundService_NotifyScoreModule(t *testing.T) {
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
			name: "Successful score module notification",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundFinalizedPayload{
					RoundID: "some-uuid",
				},
			},
			expectedEvent: roundevents.ProcessRoundScoresRequest,
			expectErr:     false,
			mockExpects: func() {
				intScore1 := 10
				intScore2 := 10
				mockRoundDB.EXPECT().GetRound(gomock.Any(), gomock.Eq("some-uuid")).Return(&roundtypes.Round{
					ID:    "some-uuid",
					Title: "Test Round",
					State: roundtypes.RoundStateFinalized,
					Participants: []roundtypes.RoundParticipant{
						{
							DiscordID: "user1",
							TagNumber: 1234,
							Score:     &intScore1,
						},
						{
							DiscordID: "user2",
							TagNumber: 0,
							Score:     &intScore2,
						},
					},
				}, nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.ProcessRoundScoresRequest), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					// Check topic first
					if topic != roundevents.ProcessRoundScoresRequest {
						return fmt.Errorf("unexpected topic: %s", topic)
					}
					// Unmarshal the payload
					var payload roundevents.ProcessRoundScoresRequestPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					// Validate payload
					if payload.RoundID != "some-uuid" {
						return fmt.Errorf("unexpected round ID: %s", payload.RoundID)
					}
					if len(payload.Scores) != 2 {
						return fmt.Errorf("unexpected number of scores: %d", len(payload.Scores))
					}
					if payload.Scores[0].DiscordID != "user1" || payload.Scores[0].TagNumber != "1234" || payload.Scores[0].Score != float64(intScore1) {
						return fmt.Errorf("unexpected score data for user1: %+v", payload.Scores[0])
					}
					if payload.Scores[1].DiscordID != "user2" || payload.Scores[1].TagNumber != "0" || payload.Scores[1].Score != float64(intScore2) {
						return fmt.Errorf("unexpected score data for user2: %+v", payload.Scores[1])
					}
					return nil
				}).Times(1)
			},
		},
		{
			name: "Successful score module notification with nil scores",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundFinalizedPayload{
					RoundID: "some-uuid",
				},
			},
			expectedEvent: roundevents.ProcessRoundScoresRequest,
			expectErr:     false,
			mockExpects: func() {
				intScore1 := 10
				// No score for intScore2
				mockRoundDB.EXPECT().GetRound(gomock.Any(), gomock.Eq("some-uuid")).Return(&roundtypes.Round{
					ID:    "some-uuid",
					Title: "Test Round",
					State: roundtypes.RoundStateFinalized,
					Participants: []roundtypes.RoundParticipant{
						{
							DiscordID: "user1",
							TagNumber: 1234,
							Score:     &intScore1,
						},
						{
							DiscordID: "user2",
							TagNumber: 0,
							Score:     nil, // No score for user2
						},
					},
				}, nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.ProcessRoundScoresRequest), gomock.Any()).DoAndReturn(func(topic string, msg *message.Message) error {
					// Check topic first
					if topic != roundevents.ProcessRoundScoresRequest {
						return fmt.Errorf("unexpected topic: %s", topic)
					}
					// Unmarshal the payload
					var payload roundevents.ProcessRoundScoresRequestPayload
					err := json.Unmarshal(msg.Payload, &payload)
					if err != nil {
						return fmt.Errorf("failed to unmarshal payload: %w", err)
					}

					// Validate payload
					if payload.RoundID != "some-uuid" {
						return fmt.Errorf("unexpected round ID: %s", payload.RoundID)
					}
					if len(payload.Scores) != 2 {
						return fmt.Errorf("unexpected number of scores: %d", len(payload.Scores))
					}
					if payload.Scores[0].DiscordID != "user1" || payload.Scores[0].TagNumber != "1234" || payload.Scores[0].Score != float64(intScore1) {
						return fmt.Errorf("unexpected score data for user1: %+v", payload.Scores[0])
					}
					// Expecting score of 0.0 for user2
					if payload.Scores[1].DiscordID != "user2" || payload.Scores[1].TagNumber != "0" || payload.Scores[1].Score != 0.0 {
						return fmt.Errorf("unexpected score data for user2: %+v", payload.Scores[1])
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
			},
		},
		{
			name: "Database error",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundFinalizedPayload{
					RoundID: "some-uuid",
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().GetRound(gomock.Any(), gomock.Eq("some-uuid")).Return(nil, fmt.Errorf("db error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.ScoreModuleNotificationError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish ProcessRoundScoresRequest event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundFinalizedPayload{
					RoundID: "some-uuid",
				},
			},
			expectErr: true,
			mockExpects: func() {
				intScore1 := 10
				mockRoundDB.EXPECT().GetRound(gomock.Any(), gomock.Eq("some-uuid")).Return(&roundtypes.Round{
					ID:    "some-uuid",
					Title: "Test Round",
					State: roundtypes.RoundStateFinalized,
					Participants: []roundtypes.RoundParticipant{
						{
							DiscordID: "user1",
							TagNumber: 1234,
							Score:     &intScore1,
						},
					},
				}, nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.ProcessRoundScoresRequest), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.ScoreModuleNotificationError), gomock.Any()).Return(nil).Times(1)
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
			err := s.NotifyScoreModule(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("NotifyScoreModule() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("NotifyScoreModule() unexpected error: %v", err)
				}
			}
		})
	}
}
