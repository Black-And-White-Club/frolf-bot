package scoreservice

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	events "github.com/Black-And-White-Club/tcr-bot/app/modules/score/domain/events"
	scoredbtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/score/infrastructure/repositories"
	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

// --- Tests ---

func TestNewScoreService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockScoreDB := scoredb.NewMockScoreDB(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testCases := []struct {
		name     string
		eventBus shared.EventBus
		scoreDB  *scoredb.MockScoreDB
		logger   *slog.Logger
		want     *ScoreService
	}{
		{
			name:     "Success",
			eventBus: mockEventBus,
			scoreDB:  mockScoreDB,
			logger:   logger,
			want: &ScoreService{
				EventBus: mockEventBus,
				ScoreDB:  mockScoreDB,
				logger:   logger,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := NewScoreService(tc.eventBus, tc.scoreDB, tc.logger)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("NewScoreService() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestScoreService_ProcessRoundScores(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testCases := []struct {
		name          string
		event         events.ScoresReceivedEvent
		mockSetup     func(mockScoreDB *scoredb.MockScoreDB, mockEventBus *eventbusmock.MockEventBus)
		expectedError string
	}{
		{
			name: "Success",
			event: events.ScoresReceivedEvent{
				RoundID: "round123",
				Scores: []events.Score{
					{DiscordID: "user1", TagNumber: "1", Score: -2},
					{DiscordID: "user2", TagNumber: "2", Score: 1},
				},
			},
			mockSetup: func(mockScoreDB *scoredb.MockScoreDB, mockEventBus *eventbusmock.MockEventBus) {
				ctx := context.Background()

				// Prepare sorted scores
				expectedScores := []scoredbtypes.Score{
					{DiscordID: "user1", TagNumber: 1, Score: -2},
					{DiscordID: "user2", TagNumber: 2, Score: 1},
				}

				// Expect LogScores to be called with sorted scores
				mockScoreDB.EXPECT().LogScores(ctx, "round123", gomock.Eq(expectedScores), "auto").Return(nil)

				// Prepare LeaderboardUpdateEvent
				leaderboardUpdate := events.LeaderboardUpdateEvent{
					RoundID: "round123",
					Scores: []events.Score{
						{DiscordID: "user1", TagNumber: "1", Score: -2},
						{DiscordID: "user2", TagNumber: "2", Score: 1},
					},
				}

				// Expect Publish to be called with correct message
				mockEventBus.EXPECT().Publish(ctx, events.LeaderboardStreamName, gomock.AssignableToTypeOf(&message.Message{})).DoAndReturn(
					func(_ context.Context, _ string, msg *message.Message) error {
						// Normalize payloads before comparing
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
		// ... (other test cases)
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

			tc.mockSetup(mockScoreDB, mockEventBus)

			err := s.ProcessRoundScores(context.Background(), tc.event)
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
