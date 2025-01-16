package scoresubscribers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"testing"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	scoreevents "github.com/Black-And-White-Club/tcr-bot/app/modules/score/domain/events"
	scorehandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/score/infrastructure/handlers/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestNewSubscribers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockHandlers := scorehandlers.NewMockHandlers(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	want := &ScoreSubscribers{
		eventBus: mockEventBus,
		handlers: mockHandlers,
		logger:   logger,
	}

	got := NewSubscribers(mockEventBus, mockHandlers, logger)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewSubscribers() = %v, want %v", got, want)
	}
}

func TestScoreSubscribers_SubscribeToScoreEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockHandlers := scorehandlers.NewMockHandlers(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name          string
		setupMocks    func()
		expectedError string
	}{
		{
			name: "Success",
			setupMocks: func() {
				mockEventBus.EXPECT().Subscribe(gomock.Any(), scoreevents.ScoreStreamName, scoreevents.ScoresReceivedEventSubject, gomock.Any()).Return(nil)
				mockEventBus.EXPECT().Subscribe(gomock.Any(), scoreevents.ScoreStreamName, scoreevents.ScoreCorrectedEventSubject, gomock.Any()).Return(nil)
				mockEventBus.EXPECT().Subscribe(gomock.Any(), scoreevents.ScoreStreamName, scoreevents.ProcessedScoresEventSubject, gomock.Any()).Return(nil)
			},
			expectedError: "",
		},
		{
			name: "Failure on ScoresReceivedEvent subscription",
			setupMocks: func() {
				mockEventBus.EXPECT().Subscribe(gomock.Any(), scoreevents.ScoreStreamName, scoreevents.ScoresReceivedEventSubject, gomock.Any()).Return(fmt.Errorf("subscription error"))
			},
			expectedError: "failed to subscribe to ScoresReceivedEvent: subscription error",
		},
		{
			name: "Failure on ScoreCorrectedEvent subscription",
			setupMocks: func() {
				mockEventBus.EXPECT().Subscribe(gomock.Any(), scoreevents.ScoreStreamName, scoreevents.ScoresReceivedEventSubject, gomock.Any()).Return(nil)
				mockEventBus.EXPECT().Subscribe(gomock.Any(), scoreevents.ScoreStreamName, scoreevents.ScoreCorrectedEventSubject, gomock.Any()).Return(fmt.Errorf("subscription error"))
			},
			expectedError: "failed to subscribe to ScoreCorrectedEvent: subscription error",
		},
		{
			name: "Failure on ProcessedScoresEvent subscription",
			setupMocks: func() {
				mockEventBus.EXPECT().Subscribe(gomock.Any(), scoreevents.ScoreStreamName, scoreevents.ScoresReceivedEventSubject, gomock.Any()).Return(nil)
				mockEventBus.EXPECT().Subscribe(gomock.Any(), scoreevents.ScoreStreamName, scoreevents.ScoreCorrectedEventSubject, gomock.Any()).Return(nil)
				mockEventBus.EXPECT().Subscribe(gomock.Any(), scoreevents.ScoreStreamName, scoreevents.ProcessedScoresEventSubject, gomock.Any()).Return(fmt.Errorf("subscription error"))
			},
			expectedError: "failed to subscribe to ProcessedScoresEvent: subscription error",
		},
		{
			name: "ScoresReceivedEvent handler error",
			setupMocks: func() {
				// Mock the handler to return an error *before* setting up Subscribe expectation
				mockHandlers.EXPECT().HandleScoresReceived(gomock.Any(), gomock.Any()).Return(fmt.Errorf("handler error"))

				mockEventBus.EXPECT().Subscribe(gomock.Any(), scoreevents.ScoreStreamName, scoreevents.ScoresReceivedEventSubject, gomock.Any()).DoAndReturn(
					func(ctx context.Context, stream string, subject string, handler func(ctx context.Context, msg *message.Message) error) error {
						// Simulate receiving a message and calling the handler
						msg := message.NewMessage(watermill.NewUUID(), []byte(`{"round_id":"round123","scores":[{"discord_id":"user1","tag_number":"1","score":-3},{"discord_id":"user2","tag_number":"2","score":1}]}`))
						msg.SetContext(ctx)
						// Call the handler function with a test message
						return handler(ctx, msg) // The mocked handler will be called, returning an error
					},
				)
				mockEventBus.EXPECT().Subscribe(gomock.Any(), scoreevents.ScoreStreamName, scoreevents.ScoreCorrectedEventSubject, gomock.Any()).Return(nil)
				mockEventBus.EXPECT().Subscribe(gomock.Any(), scoreevents.ScoreStreamName, scoreevents.ProcessedScoresEventSubject, gomock.Any()).Return(nil)
			},
			expectedError: "",
		},
		{
			name: "ScoreCorrectedEvent handler error",
			setupMocks: func() {
				// Mock the handler to return an error *before* setting up Subscribe expectation
				mockHandlers.EXPECT().HandleScoreCorrected(gomock.Any(), gomock.Any()).Return(fmt.Errorf("handler error"))

				mockEventBus.EXPECT().Subscribe(gomock.Any(), scoreevents.ScoreStreamName, scoreevents.ScoresReceivedEventSubject, gomock.Any()).Return(nil)
				mockEventBus.EXPECT().Subscribe(gomock.Any(), scoreevents.ScoreStreamName, scoreevents.ScoreCorrectedEventSubject, gomock.Any()).DoAndReturn(
					func(ctx context.Context, stream string, subject string, handler func(ctx context.Context, msg *message.Message) error) error {
						// Simulate receiving a message and calling the handler
						msg := message.NewMessage(watermill.NewUUID(), []byte(`{"round_id":"round123","discord_id":"user1","tag_number":"1","new_score":-3}`))
						msg.SetContext(ctx)
						// Call the handler function with a test message
						return handler(ctx, msg) // The mocked handler will be called, returning an error
					},
				)
				mockEventBus.EXPECT().Subscribe(gomock.Any(), scoreevents.ScoreStreamName, scoreevents.ProcessedScoresEventSubject, gomock.Any()).Return(nil)
			},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks before each test case
			mockEventBus = eventbusmock.NewMockEventBus(ctrl)
			mockHandlers = scorehandlers.NewMockHandlers(ctrl)

			// Setup mocks for this specific test case
			tt.setupMocks()

			s := &ScoreSubscribers{
				eventBus: mockEventBus,
				handlers: mockHandlers,
				logger:   logger,
			}

			err := s.SubscribeToScoreEvents(context.Background())
			if tt.expectedError == "" {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if err.Error() != tt.expectedError {
					t.Errorf("Expected error: %v, got: %v", tt.expectedError, err.Error())
				}
			}
		})
	}
}
