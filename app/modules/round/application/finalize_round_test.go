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

func TestRoundService_FinalizeRound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDBInterface(ctrl)
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
					RoundID: 1,
				},
			},
			expectedEvent: roundevents.DiscordRoundFinalized,
			expectErr:     false,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateRoundState(gomock.Any(), gomock.Eq(roundtypes.ID(1)), gomock.Eq(roundtypes.RoundStateFinalized)).Return(nil).Times(1)
				// Expect both events in the order they're published
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundFinalized), gomock.Any()).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.DiscordRoundFinalized), gomock.Any()).Return(nil).Times(1)
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
				// No mocks needed here as the function should fail before any mocked method is called
			},
		},
		{
			name: "Database error",
			args: args{
				ctx: context.Background(),
				payload: roundevents.AllScoresSubmittedPayload{
					RoundID: 1,
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateRoundState(gomock.Any(), gomock.Eq(roundtypes.ID(1)), gomock.Eq(roundtypes.RoundStateFinalized)).Return(fmt.Errorf("db error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundFinalizationError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish RoundFinalized event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.AllScoresSubmittedPayload{
					RoundID: 1,
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateRoundState(gomock.Any(), gomock.Eq(roundtypes.ID(1)), gomock.Eq(roundtypes.RoundStateFinalized)).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundFinalized), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
				// No expectation for DiscordRoundFinalized since the function should return after the first failure
			},
		},
		{
			name: "Publish DiscordRoundFinalized event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.AllScoresSubmittedPayload{
					RoundID: 1,
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().UpdateRoundState(gomock.Any(), gomock.Eq(roundtypes.ID(1)), gomock.Eq(roundtypes.RoundStateFinalized)).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundFinalized), gomock.Any()).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.DiscordRoundFinalized), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
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
	// Helper function to create a pointer to an int
	intPtr := func(i int) *int {
		return &i
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDBInterface(ctrl)
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
					RoundID: 1,
				},
			},
			expectedEvent: roundevents.ProcessRoundScoresRequest,
			expectErr:     false,
			mockExpects: func() {
				intScore1 := 10
				intScore2 := 10
				mockRoundDB.EXPECT().GetRound(gomock.Any(), gomock.Eq(roundtypes.ID(1))).Return(&roundtypes.Round{
					ID:    1,
					Title: "Test Round",
					State: roundtypes.RoundStateFinalized,
					Participants: []roundtypes.Participant{
						{
							UserID:    "user1",
							TagNumber: intPtr(1234),
							Score:     &intScore1,
						},
						{
							UserID:    "user2",
							TagNumber: intPtr(0),
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
					if payload.RoundID != 1 {
						return fmt.Errorf("unexpected round ID: %v", payload.RoundID)
					}
					if len(payload.Scores) != 2 {
						return fmt.Errorf("unexpected number of scores: %d", len(payload.Scores))
					}
					//Compare the integer values, not the pointer addresses.
					if payload.Scores[0].UserID != "user1" || *payload.Scores[0].TagNumber != 1234 || payload.Scores[0].Score != 10 {
						return fmt.Errorf("unexpected score data for user1: %+v", payload.Scores[0])
					}
					if payload.Scores[1].UserID != "user2" || *payload.Scores[1].TagNumber != 0 || payload.Scores[1].Score != 10 {
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
					RoundID: 1,
				},
			},
			expectedEvent: roundevents.ProcessRoundScoresRequest,
			expectErr:     false,
			mockExpects: func() {
				intScore1 := 10
				// No score for intScore2
				mockRoundDB.EXPECT().GetRound(gomock.Any(), gomock.Eq(roundtypes.ID(1))).Return(&roundtypes.Round{
					ID:    1,
					Title: "Test Round",
					State: roundtypes.RoundStateFinalized,
					Participants: []roundtypes.Participant{
						{
							UserID:    "user1",
							TagNumber: intPtr(1234),
							Score:     &intScore1,
						},
						{
							UserID:    "user2",
							TagNumber: intPtr(0),
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
					if payload.RoundID != 1 {
						return fmt.Errorf("unexpected round ID: %v", payload.RoundID)
					}
					if len(payload.Scores) != 2 {
						return fmt.Errorf("unexpected number of scores: %d", len(payload.Scores))
					}
					//Compare the integer values, not the pointer addresses.
					if payload.Scores[0].UserID != "user1" || *payload.Scores[0].TagNumber != 1234 || payload.Scores[0].Score != 10 {
						return fmt.Errorf("unexpected score data for user1: %+v", payload.Scores[0])
					}
					// Expecting score of 0 for user2
					if payload.Scores[1].UserID != "user2" || *payload.Scores[1].TagNumber != 0 || payload.Scores[1].Score != 0 {
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
					RoundID: 1,
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().GetRound(gomock.Any(), gomock.Eq(roundtypes.ID(1))).Return(nil, fmt.Errorf("db error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.ScoreModuleNotificationError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish ProcessRoundScoresRequest event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundFinalizedPayload{
					RoundID: 1,
				},
			},
			expectErr: true,
			mockExpects: func() {
				intScore1 := 10
				mockRoundDB.EXPECT().GetRound(gomock.Any(), gomock.Eq(roundtypes.ID(1))).Return(&roundtypes.Round{
					ID:    1,
					Title: "Test Round",
					State: roundtypes.RoundStateFinalized,
					Participants: []roundtypes.Participant{
						{
							UserID:    "user1",
							TagNumber: intPtr(1234),
							Score:     &intScore1,
						},
					},
				}, nil).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.ProcessRoundScoresRequest), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.ScoreModuleNotificationError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "GetRound returns error",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundFinalizedPayload{
					RoundID: 1,
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().GetRound(gomock.Any(), gomock.Eq(roundtypes.ID(1))).Return(nil, fmt.Errorf("database error"))
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.ScoreModuleNotificationError), gomock.Any()).Return(nil)
			},
		},
		{
			name: "Publish score.module.notification.error fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundFinalizedPayload{
					RoundID: 1,
				},
			},
			expectErr: true,
			mockExpects: func() {
				mockRoundDB.EXPECT().GetRound(gomock.Any(), gomock.Eq(roundtypes.ID(1))).Return(nil, fmt.Errorf("database error"))
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.ScoreModuleNotificationError), gomock.Any()).Return(fmt.Errorf("publish error"))
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
