package roundservice

import (
	"context"
	"errors"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	queuemocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/queue/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

// startTimePtr is a helper function to convert time.Time to *sharedtypes.StartTime.
func startTimePtr(t time.Time) *sharedtypes.StartTime {
	st := sharedtypes.StartTime(t)
	return &st
}

func TestRoundService_ScheduleRoundEvents(t *testing.T) {
	ctx := context.Background()
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name             string
		startTimeOffset  time.Duration // Offset from current time
		discordMessageID string
		mockQueueSetup   func(*queuemocks.MockQueueService, sharedtypes.RoundID, time.Time)
		expectedError    error
		wantSuccess      bool
		wantFailure      bool
	}{
		{
			name:             "successful scheduling",
			startTimeOffset:  2 * time.Hour,
			discordMessageID: "12345",
			mockQueueSetup: func(mockQueue *queuemocks.MockQueueService, roundID sharedtypes.RoundID, fixedNow time.Time) {
				// Expect cancellation of existing jobs
				mockQueue.EXPECT().CancelRoundJobs(gomock.Any(), roundID).Return(nil)

				// Expect reminder scheduling - use gomock.Any() for time since implementation uses time.Now()
				mockQueue.EXPECT().ScheduleRoundReminder(gomock.Any(), gomock.Any(), roundID, gomock.Any(), gomock.Any()).Return(nil)

				// Expect round start scheduling - use gomock.Any() for time since implementation uses time.Now()
				mockQueue.EXPECT().ScheduleRoundStart(gomock.Any(), gomock.Any(), roundID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
			wantSuccess:   true,
			wantFailure:   false,
		},
		{
			name:             "error cancelling jobs",
			startTimeOffset:  2 * time.Hour,
			discordMessageID: "12345",
			mockQueueSetup: func(mockQueue *queuemocks.MockQueueService, roundID sharedtypes.RoundID, fixedNow time.Time) {
				// Expect cancellation to fail
				mockQueue.EXPECT().CancelRoundJobs(gomock.Any(), roundID).Return(errors.New("job cancellation error"))
			},
			expectedError: nil, // Error is handled in Failure
			wantSuccess:   false,
			wantFailure:   true,
		},
		{
			name:             "error scheduling reminder",
			startTimeOffset:  2 * time.Hour,
			discordMessageID: "12345",
			mockQueueSetup: func(mockQueue *queuemocks.MockQueueService, roundID sharedtypes.RoundID, fixedNow time.Time) {
				// Expect cancellation to succeed
				mockQueue.EXPECT().CancelRoundJobs(gomock.Any(), roundID).Return(nil)

				// Expect reminder scheduling to fail - use gomock.Any() for time
				mockQueue.EXPECT().ScheduleRoundReminder(gomock.Any(), gomock.Any(), roundID, gomock.Any(), gomock.Any()).Return(errors.New("reminder scheduling error"))
			},
			expectedError: nil, // Error is handled in Failure
			wantSuccess:   false,
			wantFailure:   true,
		},
		{
			name:             "error scheduling round start",
			startTimeOffset:  2 * time.Hour,
			discordMessageID: "12345",
			mockQueueSetup: func(mockQueue *queuemocks.MockQueueService, roundID sharedtypes.RoundID, fixedNow time.Time) {
				// Expect cancellation to succeed
				mockQueue.EXPECT().CancelRoundJobs(gomock.Any(), roundID).Return(nil)

				// Expect reminder scheduling to succeed - use gomock.Any() for time
				mockQueue.EXPECT().ScheduleRoundReminder(gomock.Any(), gomock.Any(), roundID, gomock.Any(), gomock.Any()).Return(nil)

				// Expect round start scheduling to fail - use gomock.Any() for time
				mockQueue.EXPECT().ScheduleRoundStart(gomock.Any(), gomock.Any(), roundID, gomock.Any(), gomock.Any()).Return(errors.New("round start scheduling error"))
			},
			expectedError: nil, // Error is handled in Failure
			wantSuccess:   false,
			wantFailure:   true,
		},
		{
			name:             "past start time",
			startTimeOffset:  -1 * time.Hour,
			discordMessageID: "12345",
			mockQueueSetup: func(mockQueue *queuemocks.MockQueueService, roundID sharedtypes.RoundID, fixedNow time.Time) {
				// Expect cancellation to succeed
				mockQueue.EXPECT().CancelRoundJobs(gomock.Any(), roundID).Return(nil)
				// No scheduling calls expected as times are in the past
			},
			expectedError: nil,
			wantSuccess:   true,
			wantFailure:   false,
		},
		{
			name:             "far future start time",
			startTimeOffset:  720 * time.Hour,
			discordMessageID: "12345",
			mockQueueSetup: func(mockQueue *queuemocks.MockQueueService, roundID sharedtypes.RoundID, fixedNow time.Time) {
				// Expect cancellation to succeed
				mockQueue.EXPECT().CancelRoundJobs(gomock.Any(), roundID).Return(nil)

				// Expect reminder scheduling - use gomock.Any() for time
				mockQueue.EXPECT().ScheduleRoundReminder(gomock.Any(), gomock.Any(), roundID, gomock.Any(), gomock.Any()).Return(nil)

				// Expect round start scheduling - use gomock.Any() for time
				mockQueue.EXPECT().ScheduleRoundStart(gomock.Any(), gomock.Any(), roundID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
			wantSuccess:   true,
			wantFailure:   false,
		},
		{
			name:             "empty event message ID",
			startTimeOffset:  2 * time.Hour,
			discordMessageID: "",
			mockQueueSetup: func(mockQueue *queuemocks.MockQueueService, roundID sharedtypes.RoundID, fixedNow time.Time) {
				// Expect cancellation to succeed
				mockQueue.EXPECT().CancelRoundJobs(gomock.Any(), roundID).Return(nil)

				// Expect reminder scheduling with empty EventMessageID - use gomock.Any() for time
				mockQueue.EXPECT().ScheduleRoundReminder(gomock.Any(), gomock.Any(), roundID, gomock.Any(), gomock.Any()).Return(nil)

				// Expect round start scheduling - use gomock.Any() for time
				mockQueue.EXPECT().ScheduleRoundStart(gomock.Any(), gomock.Any(), roundID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
			wantSuccess:   true,
			wantFailure:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQueue := queuemocks.NewMockQueueService(ctrl)

			// Use current time instead of fixed historical time
			now := time.Now().UTC()

			// Create test data with dynamic round ID for this specific test
			testRoundID := sharedtypes.RoundID(uuid.New())
			testRoundTitle := roundtypes.Title("Test Round")
			testLocation := roundtypes.Location("Test Location")
			testDescription := roundtypes.Description("Test Description")

			// Create payload with the dynamic start time relative to current time
			payload := roundevents.RoundScheduledPayloadV1{
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID:     testRoundID,
					Title:       testRoundTitle,
					Description: &testDescription,
					Location:    &testLocation,
					StartTime:   startTimePtr(now.Add(tt.startTimeOffset)),
				},
			}

			// Setup mocks with the specific round ID for this test
			tt.mockQueueSetup(mockQueue, testRoundID, now)

			s := &RoundService{
				logger:       logger,
				metrics:      mockMetrics,
				tracer:       tracer,
				queueService: mockQueue,
			}

			guildID := sharedtypes.GuildID("test-guild")
			result, err := s.ScheduleRoundEvents(ctx, guildID, payload, tt.discordMessageID)

			// Validate error
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}

			// Validate result type
			if tt.wantSuccess && result.Success == nil {
				t.Errorf("expected success result, got failure: %v", result.Failure)
			}
			if tt.wantFailure && result.Failure == nil {
				t.Errorf("expected failure result, got success: %v", result.Success)
			}

			// Basic validation for success payloads
			if result.Success != nil {
				if successPayload, ok := result.Success.(*roundevents.RoundScheduledPayloadV1); !ok {
					t.Errorf("expected result.Success to be of type *roundevents.RoundScheduledPayloadV1, got %T", result.Success)
				} else {
					if successPayload.RoundID != testRoundID {
						t.Errorf("expected success RoundID %s, got %s", testRoundID, successPayload.RoundID)
					}
					if successPayload.Title != testRoundTitle {
						t.Errorf("expected success Title %s, got %s", testRoundTitle, successPayload.Title)
					}
					if successPayload.EventMessageID != tt.discordMessageID {
						t.Errorf("expected success EventMessageID %s, got %s", tt.discordMessageID, successPayload.EventMessageID)
					}
				}
			}

			// Basic validation for failure payloads
			if result.Failure != nil {
				if failurePayload, ok := result.Failure.(*roundevents.RoundErrorPayloadV1); !ok {
					t.Errorf("expected result.Failure to be of type *roundevents.RoundErrorPayloadV1, got %T", result.Failure)
				} else {
					if failurePayload.RoundID != testRoundID {
						t.Errorf("expected failure RoundID %s, got %s", testRoundID, failurePayload.RoundID)
					}
				}
			}
		})
	}
}
