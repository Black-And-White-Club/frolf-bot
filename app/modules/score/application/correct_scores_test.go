package scoreservice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	events "github.com/Black-And-White-Club/tcr-bot/app/modules/score/domain/events"
	scoredbtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/score/infrastructure/repositories"
	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestScoreService_CorrectScore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testCases := []struct {
		name          string
		event         events.ScoreCorrectedEvent
		mockSetup     func(*ScoreService, *scoredb.MockScoreDB, *eventbusmock.MockEventBus)
		expectedError string
	}{
		{
			name: "Success",
			event: events.ScoreCorrectedEvent{
				RoundID:   "round123",
				DiscordID: "user1",
				TagNumber: "1",
				NewScore:  -3,
			},
			mockSetup: func(s *ScoreService, mockScoreDB *scoredb.MockScoreDB, mockEventBus *eventbusmock.MockEventBus) {
				ctx := context.Background()

				mockScoreDB.EXPECT().UpdateOrAddScore(ctx, &scoredbtypes.Score{
					DiscordID: "user1",
					RoundID:   "round123",
					TagNumber: 1,
					Score:     -3,
				}).Return(nil)

				mockScoreDB.EXPECT().GetScoresForRound(ctx, "round123").Return([]scoredbtypes.Score{
					{DiscordID: "user1", TagNumber: 1, Score: -3, RoundID: "round123"},
				}, nil)

				mockScoreDB.EXPECT().LogScores(ctx, "round123", []scoredbtypes.Score{
					{DiscordID: "user1", TagNumber: 1, Score: -3, RoundID: "round123"},
				}, "manual").Return(nil)

				leaderboardUpdate := events.LeaderboardUpdateEvent{
					RoundID: "round123",
					Scores: []events.Score{
						{DiscordID: "user1", TagNumber: "1", Score: -3},
					},
				}

				mockEventBus.EXPECT().Publish(ctx, events.LeaderboardStreamName, gomock.AssignableToTypeOf(&message.Message{})).DoAndReturn(
					func(_ context.Context, _ string, msg *message.Message) error {
						normalizedExpected, _ := json.Marshal(leaderboardUpdate)
						normalizedActual, _ := json.Marshal(json.RawMessage(msg.Payload))

						if msg.Metadata.Get("subject") != events.LeaderboardUpdateEventSubject {
							t.Errorf("Expected subject: %s, got: %s", events.LeaderboardUpdateEventSubject, msg.Metadata.Get("subject"))
						}
						if !reflect.DeepEqual(normalizedActual, normalizedExpected) {
							t.Errorf("Payload mismatch.\nExpected: %s\nGot: %s", string(normalizedExpected), string(normalizedActual))
						}
						return nil
					},
				)
			},
			expectedError: "",
		},
		{
			name: "Error converting tag number",
			event: events.ScoreCorrectedEvent{
				TagNumber: "invalid",
			},
			mockSetup:     func(s *ScoreService, mockScoreDB *scoredb.MockScoreDB, mockEventBus *eventbusmock.MockEventBus) {},
			expectedError: "invalid tag number",
		},
		{
			name: "Error updating score",
			event: events.ScoreCorrectedEvent{
				RoundID:   "round123",
				DiscordID: "user1",
				TagNumber: "1",
				NewScore:  -3,
			},
			mockSetup: func(s *ScoreService, mockScoreDB *scoredb.MockScoreDB, mockEventBus *eventbusmock.MockEventBus) {
				ctx := context.Background()
				mockScoreDB.EXPECT().UpdateOrAddScore(ctx, gomock.Any()).Return(fmt.Errorf("database error"))
			},
			expectedError: "database error",
		},
		{
			name: "Error getting scores for round",
			event: events.ScoreCorrectedEvent{
				RoundID:   "round123",
				DiscordID: "user1",
				TagNumber: "1",
				NewScore:  -3,
			},
			mockSetup: func(s *ScoreService, mockScoreDB *scoredb.MockScoreDB, mockEventBus *eventbusmock.MockEventBus) {
				ctx := context.Background()
				mockScoreDB.EXPECT().UpdateOrAddScore(ctx, gomock.Any()).Return(nil)
				mockScoreDB.EXPECT().GetScoresForRound(ctx, "round123").Return(nil, fmt.Errorf("database error"))
			},
			expectedError: "database error",
		},
		{
			name: "Error logging updated scores",
			event: events.ScoreCorrectedEvent{
				RoundID:   "round123",
				DiscordID: "user1",
				TagNumber: "1",
				NewScore:  -3,
			},
			mockSetup: func(s *ScoreService, mockScoreDB *scoredb.MockScoreDB, mockEventBus *eventbusmock.MockEventBus) {
				ctx := context.Background()
				mockScoreDB.EXPECT().UpdateOrAddScore(ctx, gomock.Any()).Return(nil)
				mockScoreDB.EXPECT().GetScoresForRound(ctx, "round123").Return([]scoredbtypes.Score{
					{DiscordID: "user1", TagNumber: 1, Score: -3, RoundID: "round123"},
				}, nil)
				mockScoreDB.EXPECT().LogScores(ctx, "round123", gomock.Any(), "manual").Return(fmt.Errorf("database error"))
			},
			expectedError: "database error",
		},
		{
			name: "Error publishing event",
			event: events.ScoreCorrectedEvent{
				RoundID:   "round123",
				DiscordID: "user1",
				TagNumber: "1",
				NewScore:  -3,
			},
			mockSetup: func(s *ScoreService, mockScoreDB *scoredb.MockScoreDB, mockEventBus *eventbusmock.MockEventBus) {
				ctx := context.Background()
				mockScoreDB.EXPECT().UpdateOrAddScore(ctx, gomock.Any()).Return(nil)
				mockScoreDB.EXPECT().GetScoresForRound(ctx, "round123").Return([]scoredbtypes.Score{
					{DiscordID: "user1", TagNumber: 1, Score: -3, RoundID: "round123"},
				}, nil)
				mockScoreDB.EXPECT().LogScores(ctx, "round123", gomock.Any(), "manual").Return(nil)
				mockEventBus.EXPECT().Publish(ctx, events.LeaderboardStreamName, gomock.Any()).Return(fmt.Errorf("event bus error"))
			},
			expectedError: "event bus error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockScoreDB := scoredb.NewMockScoreDB(ctrl)
			mockEventBus := eventbusmock.NewMockEventBus(ctrl)

			s := &ScoreService{
				ScoreDB:  mockScoreDB,
				EventBus: mockEventBus,
				logger:   logger,
			}

			tc.mockSetup(s, mockScoreDB, mockEventBus)

			err := s.CorrectScore(context.Background(), tc.event)
			if tc.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("Error mismatch.\nExpected: %s\nGot: %s", tc.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
