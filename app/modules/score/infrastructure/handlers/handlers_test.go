package scorehandlers

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
	"github.com/Black-And-White-Club/tcr-bot/app/modules/score/application/mocks"
	events "github.com/Black-And-White-Club/tcr-bot/app/modules/score/domain/events"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestNewScoreHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCases := []struct {
		name     string
		service  *mocks.MockService         // Mock service
		eventBus *eventbusmock.MockEventBus // Mock event bus
		logger   *slog.Logger
		want     *ScoreHandlers
	}{
		{
			name:     "Success",
			service:  mocks.NewMockService(ctrl),
			eventBus: eventbusmock.NewMockEventBus(ctrl),
			logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			want: &ScoreHandlers{
				ScoreService: mocks.NewMockService(ctrl),
				EventBus:     eventbusmock.NewMockEventBus(ctrl),
				logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := NewScoreHandlers(tc.service, tc.eventBus, tc.logger)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("NewScoreHandlers() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestScoreHandlers_HandleScoresReceived(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockScoreService := mocks.NewMockService(ctrl)
	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name          string
		msg           *message.Message
		mockSetup     func(*mocks.MockService, *eventbusmock.MockEventBus, *testing.T)
		expectedError string
	}{
		{
			name: "Success",
			msg: &message.Message{
				UUID:    "test-message-id",
				Payload: []byte(`{"round_id":"round123","scores":[{"discord_id":"user1","tag_number":"1","score":-3},{"discord_id":"user2","tag_number":"2","score":1}]}`),
			},
			mockSetup: func(mockScoreService *mocks.MockService, mockEventBus *eventbusmock.MockEventBus, t *testing.T) {
				event := events.ScoresReceivedEvent{
					RoundID: "round123",
					Scores: []events.Score{
						{DiscordID: "user1", TagNumber: "1", Score: -3},
						{DiscordID: "user2", TagNumber: "2", Score: 1},
					},
				}

				mockScoreService.EXPECT().ProcessRoundScores(gomock.Any(), event).Return(nil)

				mockEventBus.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, subject string, msg *message.Message) error {
						// 1. Validate the subject:
						if subject != events.ProcessedScoresEventSubject {
							return fmt.Errorf("expected subject %s, got %s", events.ProcessedScoresEventSubject, subject)
						}

						// 2. Unmarshal the payload:
						var processedEvent events.ProcessedScoresEvent
						if err := json.Unmarshal(msg.Payload, &processedEvent); err != nil {
							return fmt.Errorf("failed to unmarshal ProcessedScoresEvent: %w", err)
						}

						// 3. Validate the unmarshalled event's content:
						if processedEvent.RoundID != "round123" {
							t.Errorf("Expected RoundID %s, got %s", "round123", processedEvent.RoundID)
						}
						if !processedEvent.Success {
							t.Errorf("Expected Success to be true, got false")
						}
						if processedEvent.Error != "" {
							t.Errorf("Expected Error to be empty, got %s", processedEvent.Error)
						}
						if len(processedEvent.Scores) != 2 {
							t.Errorf("Expected Scores length to be 2, got %d", len(processedEvent.Scores))
						}

						return nil
					},
				).Return(nil)
			},
			expectedError: "",
		},
		{
			name: "Unmarshal error",
			msg: &message.Message{
				Payload: []byte(`invalid json`),
			},
			mockSetup:     func(mockScoreService *mocks.MockService, mockEventBus *eventbusmock.MockEventBus, t *testing.T) {},
			expectedError: "failed to unmarshal ScoresReceivedEvent",
		},
		{
			name: "ProcessRoundScores error",
			msg: &message.Message{
				UUID:    "test-message-id",
				Payload: []byte(`{"round_id":"round123","scores":[{"discord_id":"user1","tag_number":"1","score":-3}]}`),
			},
			mockSetup: func(mockScoreService *mocks.MockService, mockEventBus *eventbusmock.MockEventBus, t *testing.T) {
				event := events.ScoresReceivedEvent{
					RoundID: "round123",
					Scores: []events.Score{
						{DiscordID: "user1", TagNumber: "1", Score: -3},
					},
				}

				mockScoreService.EXPECT().ProcessRoundScores(gomock.Any(), event).Return(fmt.Errorf("service error"))

				mockEventBus.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, subject string, msg *message.Message) error {
						// 1. Validate the subject:
						if subject != events.ProcessedScoresEventSubject {
							return fmt.Errorf("expected subject %s, got %s", events.ProcessedScoresEventSubject, subject)
						}

						// 2. Unmarshal the payload:
						var processedEvent events.ProcessedScoresEvent
						if err := json.Unmarshal(msg.Payload, &processedEvent); err != nil {
							return fmt.Errorf("failed to unmarshal ProcessedScoresEvent: %w", err)
						}

						// 3. Validate the unmarshalled event's content:
						if processedEvent.RoundID != "round123" {
							t.Errorf("Expected RoundID %s, got %s", "round123", processedEvent.RoundID)
						}
						if processedEvent.Success {
							t.Errorf("Expected Success to be false, got true")
						}
						if processedEvent.Error != "service error" {
							t.Errorf("Expected Error to be 'service error', got %s", processedEvent.Error)
						}
						if len(processedEvent.Scores) != 1 {
							t.Errorf("Expected Scores length to be 1, got %d", len(processedEvent.Scores))
						}

						return nil
					},
				).Return(nil)
			},
			expectedError: "service error",
		},
		{
			name: "Publish error after ProcessRoundScores error",
			msg: &message.Message{
				UUID:    "test-message-id",
				Payload: []byte(`{"round_id":"round123","scores":[{"discord_id":"user1","tag_number":"1","score":-3}]}`),
			},
			mockSetup: func(mockScoreService *mocks.MockService, mockEventBus *eventbusmock.MockEventBus, t *testing.T) {
				event := events.ScoresReceivedEvent{
					RoundID: "round123",
					Scores: []events.Score{
						{DiscordID: "user1", TagNumber: "1", Score: -3},
					},
				}

				mockScoreService.EXPECT().ProcessRoundScores(gomock.Any(), event).Return(fmt.Errorf("service error"))

				// Expecting Publish to be called and return an error
				mockEventBus.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("publish error"))
			},
			expectedError: "publish error",
		},
		{
			name: "Publish error after success",
			msg: &message.Message{
				UUID:    "test-message-id",
				Payload: []byte(`{"round_id":"round123","scores":[{"discord_id":"user1","tag_number":"1","score":-3}]}`),
			},
			mockSetup: func(mockScoreService *mocks.MockService, mockEventBus *eventbusmock.MockEventBus, t *testing.T) {
				event := events.ScoresReceivedEvent{
					RoundID: "round123",
					Scores: []events.Score{
						{DiscordID: "user1", TagNumber: "1", Score: -3},
					},
				}

				mockScoreService.EXPECT().ProcessRoundScores(gomock.Any(), event).Return(nil)

				// Expecting Publish to be called and return an error
				mockEventBus.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("publish error"))
			},
			expectedError: "publish error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ScoreHandlers{
				ScoreService: mockScoreService,
				EventBus:     mockEventBus,
				logger:       logger,
			}

			tt.mockSetup(mockScoreService, mockEventBus, t)

			err := h.HandleScoresReceived(context.Background(), tt.msg)
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Error mismatch.\nExpected: %s\nGot: %s", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestScoreHandlers_HandleScoreCorrected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockScoreService := mocks.NewMockService(ctrl)
	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testCases := []struct {
		name          string
		msg           *message.Message
		mockSetup     func(*mocks.MockService, *eventbusmock.MockEventBus)
		expectedError string
	}{
		{
			name: "Success",
			msg: &message.Message{
				Payload: []byte(`{"round_id":"round123","discord_id":"user1","tag_number":"1","new_score":-3}`),
			},
			mockSetup: func(mockScoreService *mocks.MockService, mockEventBus *eventbusmock.MockEventBus) {
				ctx := context.Background()
				event := events.ScoreCorrectedEvent{
					RoundID:   "round123",
					DiscordID: "user1",
					TagNumber: "1",
					NewScore:  -3,
				}

				mockScoreService.EXPECT().CorrectScore(ctx, event).Return(nil)

				// Expect Publish to be called with a success event
				mockEventBus.EXPECT().Publish(gomock.Any(), events.ScoreStreamName, gomock.Any()).DoAndReturn(
					func(_ context.Context, _ string, msg *message.Message) error {
						var event events.ScoreCorrectedEvent
						if err := json.Unmarshal(msg.Payload, &event); err != nil {
							return fmt.Errorf("failed to unmarshal ScoreCorrectedEvent: %w", err)
						}
						// Add assertions here to check the content of the success event if needed
						return nil
					},
				).Return(nil)
			},
			expectedError: "",
		},
		{
			name: "Error unmarshaling event",
			msg: &message.Message{
				Payload: []byte(`invalid json`),
			},
			mockSetup:     func(mockScoreService *mocks.MockService, mockEventBus *eventbusmock.MockEventBus) {},
			expectedError: "failed to unmarshal ScoreCorrectedEvent",
		},
		{
			name: "Error correcting score",
			msg: &message.Message{
				Payload: []byte(`{"round_id":"round123","discord_id":"user1","tag_number":"1","new_score":-3}`),
			},
			mockSetup: func(mockScoreService *mocks.MockService, mockEventBus *eventbusmock.MockEventBus) {
				ctx := context.Background()
				event := events.ScoreCorrectedEvent{
					RoundID:   "round123",
					DiscordID: "user1",
					TagNumber: "1",
					NewScore:  -3,
				}

				mockScoreService.EXPECT().CorrectScore(ctx, event).Return(fmt.Errorf("service error"))

				// Expect Publish to be called with an error event
				mockEventBus.EXPECT().Publish(gomock.Any(), events.ScoreStreamName, gomock.Any()).DoAndReturn(
					func(_ context.Context, _ string, msg *message.Message) error {
						var event events.ScoreCorrectedEvent
						if err := json.Unmarshal(msg.Payload, &event); err != nil {
							return fmt.Errorf("failed to unmarshal ScoreCorrectedEvent: %w", err)
						}
						// Add assertions here to check the content of the error event if needed
						return nil
					},
				).Return(nil)
			},
			expectedError: "service error",
		},
		// You can add more test cases here for other error scenarios
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := &ScoreHandlers{
				ScoreService: mockScoreService,
				EventBus:     mockEventBus,
				logger:       logger,
			}

			tc.mockSetup(mockScoreService, mockEventBus)

			err := h.HandleScoreCorrected(context.Background(), tc.msg)
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
