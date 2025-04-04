package roundservice

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/round"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/mocks"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestRoundService_ScheduleRoundEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRoundID := sharedtypes.RoundID(uuid.New())
	testRoundTitle := roundtypes.Title("Test Round")
	testLocation := roundtypes.Location("Test Location")
	testStartTime := time.Now().Add(2 * time.Hour)
	testReminderTime := testStartTime.Add(-1 * time.Hour)
	testEventMessageID := testRoundID

	ctx := context.Background()
	mockDB := rounddb.NewMockRoundDB(ctrl)
	mockLogger := &lokifrolfbot.NoOpLogger{}
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockTracer := tempofrolfbot.NewNoOpTracer()
	mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)
	mockEventBus := eventbus.NewMockEventBus(ctrl)

	testPayload := roundevents.RoundStoredPayload{
		Round: roundtypes.Round{
			ID:             testRoundID,
			Title:          testRoundTitle,
			Location:       &testLocation,
			EventMessageID: testEventMessageID,
		},
	}

	tests := []struct {
		name          string
		payload       roundevents.RoundStoredPayload
		startTime     sharedtypes.StartTime
		mockEventBus  func(*eventbus.MockEventBus)
		expectedError error
	}{
		{
			name:      "successful scheduling",
			payload:   testPayload,
			startTime: sharedtypes.StartTime(testStartTime),
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, sharedtypes.StartTime(testReminderTime), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, sharedtypes.StartTime(testStartTime), gomock.Any()).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:      "error creating consumer",
			payload:   testPayload,
			startTime: sharedtypes.StartTime(testStartTime),
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, gomock.Any()).Return(errors.New("consumer creation error"))
			},
			expectedError: fmt.Errorf("failed to create consumer for round %s: consumer creation error", testRoundID),
		},
		{
			name:      "error scheduling reminder",
			payload:   testPayload,
			startTime: sharedtypes.StartTime(testStartTime),
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, gomock.Any(), gomock.Any()).Return(errors.New("reminder scheduling error"))
			},
			expectedError: fmt.Errorf("failed to schedule reminder: reminder scheduling error"),
		},
		{
			name:      "error scheduling round start",
			payload:   testPayload,
			startTime: sharedtypes.StartTime(testStartTime),
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, gomock.Any(), gomock.Any()).Return(errors.New("round start scheduling error"))
			},
			expectedError: fmt.Errorf("failed to schedule round start: round start scheduling error"),
		},
		{
			name:      "past start time",
			payload:   testPayload,
			startTime: sharedtypes.StartTime(time.Now().Add(-1 * time.Hour)),
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "missing round ID",
			payload: roundevents.RoundStoredPayload{
				Round: roundtypes.Round{
					ID:             sharedtypes.RoundID(uuid.Nil),
					Title:          testRoundTitle,
					Location:       &testLocation,
					EventMessageID: testEventMessageID,
				},
			},
			startTime: sharedtypes.StartTime(testStartTime),
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, sharedtypes.RoundID(uuid.Nil), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, sharedtypes.RoundID(uuid.Nil), gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, sharedtypes.RoundID(uuid.Nil), gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "missing title",
			payload: roundevents.RoundStoredPayload{
				Round: roundtypes.Round{
					ID:             testRoundID,
					Title:          "",
					Location:       &testLocation,
					EventMessageID: testEventMessageID,
				},
			},
			startTime: sharedtypes.StartTime(testStartTime),
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "missing location",
			payload: roundevents.RoundStoredPayload{
				Round: roundtypes.Round{
					ID:             testRoundID,
					Title:          testRoundTitle,
					Location:       nil,
					EventMessageID: testEventMessageID,
				},
			},
			startTime: sharedtypes.StartTime(testStartTime),
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:      "far future start time",
			payload:   testPayload,
			startTime: sharedtypes.StartTime(time.Now().Add(720 * time.Hour)),
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:      "zero start time",
			payload:   testPayload,
			startTime: sharedtypes.StartTime(time.Time{}),
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "nil event message ID",
			payload: func() roundevents.RoundStoredPayload {
				p := testPayload
				p.Round.EventMessageID = sharedtypes.RoundID(uuid.Nil)
				return p
			}(),
			startTime: sharedtypes.StartTime(testStartTime),
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "with description in payload",
			payload: func() roundevents.RoundStoredPayload {
				p := testPayload
				description := roundtypes.Description("Test Description")
				p.Round.Description = &description
				return p
			}(),
			startTime: sharedtypes.StartTime(testStartTime),
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "with createdBy in payload",
			payload: func() roundevents.RoundStoredPayload {
				p := testPayload
				p.Round.CreatedBy = sharedtypes.DiscordID("user123")
				return p
			}(),
			startTime: sharedtypes.StartTime(testStartTime),
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:      "reminder time equals start time",
			payload:   testPayload,
			startTime: sharedtypes.StartTime(time.Now()),
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockEventBus(mockEventBus)

			s := &RoundService{
				RoundDB:        mockDB,
				logger:         mockLogger,
				metrics:        mockMetrics,
				tracer:         mockTracer,
				roundValidator: mockRoundValidator,
				EventBus:       mockEventBus,
				serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func() (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc()
				},
			}

			_, err := s.ScheduleRoundEvents(context.Background(), tt.payload, tt.startTime)
			if (err != nil) && (tt.expectedError == nil || err.Error() != tt.expectedError.Error()) {
				t.Fatalf("expected error %v, got %v", tt.expectedError, err)
			}

			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}
