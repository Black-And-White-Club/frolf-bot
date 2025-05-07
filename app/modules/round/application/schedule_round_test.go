package roundservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

// startTimePtr is a helper function to convert time.Time to *sharedtypes.StartTime.
func startTimePtr(t time.Time) *sharedtypes.StartTime {
	st := sharedtypes.StartTime(t)
	return &st
}

// descriptionPtr is a helper function to get a pointer to a roundtypes.Description.
func descriptionPtr(d roundtypes.Description) *roundtypes.Description {
	return &d
}

func TestRoundService_ScheduleRoundEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Common test data setup
	testRoundID := sharedtypes.RoundID(uuid.New())
	testRoundTitle := roundtypes.Title("Test Round")
	testLocation := roundtypes.Location("Test Location")
	testStartTime := time.Now().Add(2 * time.Hour)
	testReminderTime := testStartTime.Add(-1 * time.Hour)
	testEventMessageID := "12345"
	testChannelID := "channel123"
	testUserID := sharedtypes.DiscordID("user456")
	testDescription := roundtypes.Description("Test Description")

	ctx := context.Background()
	mockDB := rounddb.NewMockRoundDB(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)

	// Define a helper to create the base payload
	createBasePayload := func(startTime time.Time) roundevents.RoundScheduledPayload {
		return roundevents.RoundScheduledPayload{
			BaseRoundPayload: roundtypes.BaseRoundPayload{
				RoundID:     testRoundID,
				Title:       testRoundTitle,
				Description: &testDescription,
				Location:    &testLocation,
				StartTime:   startTimePtr(startTime),
				UserID:      testUserID,
			},
		}
	}

	tests := []struct {
		name             string
		payload          roundevents.RoundScheduledPayload
		discordMessageID string
		mockEventBus     func(*eventbus.MockEventBus)
		expectedError    error
		expectedResult   RoundOperationResult
	}{
		{
			name:             "successful scheduling",
			payload:          createBasePayload(testStartTime),
			discordMessageID: testEventMessageID,
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, testStartTime).Return(nil)

				expectedReminderPayload := roundevents.DiscordReminderPayload{
					RoundID:          testRoundID,
					ReminderType:     "1h",
					RoundTitle:       testRoundTitle,
					StartTime:        startTimePtr(testStartTime),
					Location:         &testLocation,
					UserIDs:          []sharedtypes.DiscordID{},
					DiscordChannelID: testChannelID,
					DiscordGuildID:   "",
					EventMessageID:   testEventMessageID,
				}
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, testReminderTime, gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string, _ sharedtypes.RoundID, scheduledTime time.Time, message []byte) error {
					if !scheduledTime.Equal(testReminderTime) {
						t.Errorf("expected reminder time %v, got %v", testReminderTime, scheduledTime)
					}
					var actualReminderPayload roundevents.DiscordReminderPayload
					if err := json.Unmarshal(message, &actualReminderPayload); err != nil {
						t.Fatalf("failed to unmarshal reminder payload: %v", err)
					}
					// Deep compare relevant fields
					if actualReminderPayload.RoundID != expectedReminderPayload.RoundID ||
						actualReminderPayload.ReminderType != expectedReminderPayload.ReminderType ||
						actualReminderPayload.RoundTitle != expectedReminderPayload.RoundTitle ||
						(actualReminderPayload.StartTime == nil && expectedReminderPayload.StartTime != nil) ||
						(actualReminderPayload.StartTime != nil && expectedReminderPayload.StartTime == nil) ||
						(actualReminderPayload.StartTime != nil && expectedReminderPayload.StartTime != nil && *actualReminderPayload.StartTime == (*expectedReminderPayload.StartTime)) ||
						(actualReminderPayload.Location == nil && expectedReminderPayload.Location != nil) ||
						(actualReminderPayload.Location != nil && expectedReminderPayload.Location == nil) ||
						(actualReminderPayload.Location != nil && expectedReminderPayload.Location != nil && *actualReminderPayload.Location != *expectedReminderPayload.Location) ||
						len(actualReminderPayload.UserIDs) != len(expectedReminderPayload.UserIDs) ||
						actualReminderPayload.DiscordChannelID != expectedReminderPayload.DiscordChannelID ||
						actualReminderPayload.DiscordGuildID != expectedReminderPayload.DiscordGuildID ||
						actualReminderPayload.EventMessageID != expectedReminderPayload.EventMessageID {
						t.Errorf("unexpected reminder payload content.\nExpected: %+v\nGot: %+v", expectedReminderPayload, actualReminderPayload)
					}
					return nil
				})

				expectedStartPayload := roundevents.DiscordRoundStartPayload{
					RoundID:          testRoundID,
					Title:            testRoundTitle,
					Location:         &testLocation,
					StartTime:        startTimePtr(testStartTime),
					Participants:     []roundevents.RoundParticipant{},
					DiscordChannelID: testChannelID,
					DiscordGuildID:   "",
					EventMessageID:   testEventMessageID,
				}
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, testStartTime, gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string, _ sharedtypes.RoundID, scheduledTime time.Time, message []byte) error {
					if !scheduledTime.Equal(testStartTime) {
						t.Errorf("expected start time %v, got %v", testStartTime, scheduledTime)
					}
					var actualStartPayload roundevents.DiscordRoundStartPayload
					if err := json.Unmarshal(message, &actualStartPayload); err != nil {
						t.Fatalf("failed to unmarshal start payload: %v", err)
					}
					// Deep compare relevant fields
					if actualStartPayload.RoundID != expectedStartPayload.RoundID ||
						actualStartPayload.Title != expectedStartPayload.Title ||
						(actualStartPayload.Location == nil && expectedStartPayload.Location != nil) ||
						(actualStartPayload.Location != nil && expectedStartPayload.Location == nil) ||
						(actualStartPayload.Location != nil && expectedStartPayload.Location != nil && *actualStartPayload.Location != *expectedStartPayload.Location) ||
						(actualStartPayload.StartTime == nil && expectedStartPayload.StartTime != nil) ||
						(actualStartPayload.StartTime != nil && expectedStartPayload.StartTime == nil) ||
						(actualStartPayload.StartTime != nil && expectedStartPayload.StartTime != nil && *actualStartPayload.StartTime == (*expectedStartPayload.StartTime)) ||
						len(actualStartPayload.Participants) != len(expectedStartPayload.Participants) ||
						actualStartPayload.DiscordChannelID != expectedStartPayload.DiscordChannelID ||
						actualStartPayload.DiscordGuildID != expectedStartPayload.DiscordGuildID ||
						actualStartPayload.EventMessageID != expectedStartPayload.EventMessageID {
						t.Errorf("unexpected start payload content.\nExpected: %+v\nGot: %+v", expectedStartPayload, actualStartPayload)
					}
					return nil
				})
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testRoundTitle,
						Description: &testDescription,
						Location:    &testLocation,
						StartTime:   startTimePtr(testStartTime),
						UserID:      testUserID,
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name:             "error creating consumer",
			payload:          createBasePayload(testStartTime),
			discordMessageID: testEventMessageID,
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, testStartTime).Return(errors.New("consumer creation error"))
			},
			expectedError: fmt.Errorf("failed to create consumer for round %s: consumer creation error", testRoundID),
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: testRoundID,
					Error:   "failed to create consumer for round " + testRoundID.String() + ": consumer creation error",
				},
			},
		},
		{
			name:             "error scheduling reminder",
			payload:          createBasePayload(testStartTime),
			discordMessageID: testEventMessageID,
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, testStartTime).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, testReminderTime, gomock.Any(), gomock.Any()).Return(errors.New("reminder scheduling error"))
			},
			expectedError: fmt.Errorf("failed to schedule reminder: reminder scheduling error"),
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: testRoundID,
					Error:   "failed to schedule reminder: reminder scheduling error",
				},
			},
		},
		{
			name:             "error scheduling round start",
			payload:          createBasePayload(testStartTime),
			discordMessageID: testEventMessageID,
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, testStartTime).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, testReminderTime, gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, testStartTime, gomock.Any(), gomock.Any()).Return(errors.New("round start scheduling error"))
			},
			expectedError: fmt.Errorf("failed to schedule round start: round start scheduling error"),
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: testRoundID,
					Error:   "failed to schedule round start: round start scheduling error",
				},
			},
		},
		{
			name:             "past start time",
			payload:          createBasePayload(time.Now().Add(-1 * time.Hour)),
			discordMessageID: testEventMessageID,
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				pastStartTime := time.Now().Add(-1 * time.Hour)
				pastReminderTime := pastStartTime.Add(-1 * time.Hour)
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, pastStartTime).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, pastReminderTime, gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, pastStartTime, gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testRoundTitle,
						Description: &testDescription,
						Location:    &testLocation,
						StartTime:   startTimePtr(time.Now().Add(-1 * time.Hour)),
						UserID:      testUserID,
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name:             "missing round ID in payload",
			payload:          createBasePayload(testStartTime),
			discordMessageID: testEventMessageID,
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				payloadWithNilID := createBasePayload(testStartTime)
				payloadWithNilID.RoundID = sharedtypes.RoundID(uuid.Nil)

				mockEB.EXPECT().ProcessDelayedMessages(ctx, sharedtypes.RoundID(uuid.Nil), testStartTime).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, sharedtypes.RoundID(uuid.Nil), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, sharedtypes.RoundID(uuid.Nil), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     sharedtypes.RoundID(uuid.Nil),
						Title:       testRoundTitle,
						Description: &testDescription,
						Location:    &testLocation,
						StartTime:   startTimePtr(testStartTime),
						UserID:      testUserID,
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name:             "missing title in payload",
			payload:          createBasePayload(testStartTime),
			discordMessageID: testEventMessageID,
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				payloadWithEmptyTitle := createBasePayload(testStartTime)
				payloadWithEmptyTitle.Title = ""

				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, testStartTime).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       "",
						Description: &testDescription,
						Location:    &testLocation,
						StartTime:   startTimePtr(testStartTime),
						UserID:      testUserID,
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name:             "missing location in payload",
			payload:          createBasePayload(testStartTime),
			discordMessageID: testEventMessageID,
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				payloadWithNilLocation := createBasePayload(testStartTime)
				payloadWithNilLocation.Location = nil

				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, testStartTime).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testRoundTitle,
						Description: &testDescription,
						Location:    nil,
						StartTime:   startTimePtr(testStartTime),
						UserID:      testUserID,
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name:             "far future start time",
			payload:          createBasePayload(time.Now().Add(720 * time.Hour)),
			discordMessageID: testEventMessageID,
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				futureStartTime := time.Now().Add(720 * time.Hour)
				futureReminderTime := futureStartTime.Add(-1 * time.Hour)
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, futureStartTime).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, futureReminderTime, gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, futureStartTime, gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testRoundTitle,
						Description: &testDescription,
						Location:    &testLocation,
						StartTime:   startTimePtr(time.Now().Add(720 * time.Hour)),
						UserID:      testUserID,
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name:             "zero start time",
			payload:          createBasePayload(time.Time{}),
			discordMessageID: testEventMessageID,
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				zeroTime := time.Time{}
				zeroReminderTime := zeroTime.Add(-1 * time.Hour)
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, zeroTime).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, zeroReminderTime, gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, zeroTime, gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testRoundTitle,
						Description: &testDescription,
						Location:    &testLocation,
						StartTime:   startTimePtr(time.Time{}),
						UserID:      testUserID,
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name:             "empty event message ID",
			payload:          createBasePayload(testStartTime),
			discordMessageID: "",
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, testStartTime).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string, _ sharedtypes.RoundID, scheduledTime time.Time, message []byte) error {
					var actualReminderPayload roundevents.DiscordReminderPayload
					json.Unmarshal(message, &actualReminderPayload)
					if actualReminderPayload.EventMessageID != "" {
						t.Errorf("expected empty reminder EventMessageID, got %q", actualReminderPayload.EventMessageID)
					}
					return nil
				})
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string, _ sharedtypes.RoundID, scheduledTime time.Time, message []byte) error {
					var actualStartPayload roundevents.DiscordRoundStartPayload
					json.Unmarshal(message, &actualStartPayload)
					if actualStartPayload.DiscordChannelID != "" {
						t.Errorf("expected empty start DiscordChannelID, got %q", actualStartPayload.DiscordChannelID)
					}
					return nil
				})
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testRoundTitle,
						Description: &testDescription,
						Location:    &testLocation,
						StartTime:   startTimePtr(testStartTime),
						UserID:      testUserID,
					},
					EventMessageID: "",
				},
			},
		},
		{
			name:             "with description in payload",
			payload:          createBasePayload(testStartTime),
			discordMessageID: testEventMessageID,
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				payloadWithDescription := createBasePayload(testStartTime)
				description := roundtypes.Description("Another Description")
				payloadWithDescription.Description = &description

				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, testStartTime).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testRoundTitle,
						Description: descriptionPtr(roundtypes.Description("Another Description")),
						Location:    &testLocation,
						StartTime:   startTimePtr(testStartTime),
						UserID:      testUserID,
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name:             "with UserID in payload",
			payload:          createBasePayload(testStartTime),
			discordMessageID: testEventMessageID,
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				payloadWithUserID := createBasePayload(testStartTime)
				payloadWithUserID.UserID = sharedtypes.DiscordID("anotheruser")

				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, testStartTime).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testRoundTitle,
						Description: &testDescription,
						Location:    &testLocation,
						StartTime:   startTimePtr(testStartTime),
						UserID:      sharedtypes.DiscordID("anotheruser"),
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name:             "reminder time equals start time",
			payload:          createBasePayload(time.Now()),
			discordMessageID: testEventMessageID,
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				now := time.Now()
				reminderTime := now.Add(-1 * time.Hour)
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, now).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, reminderTime, gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, now, gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testRoundTitle,
						Description: &testDescription,
						Location:    &testLocation,
						StartTime:   startTimePtr(time.Now()),
						UserID:      testUserID,
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name:             "missing channel ID in payload",
			payload:          createBasePayload(testStartTime),
			discordMessageID: testEventMessageID,
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, testStartTime).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string, _ sharedtypes.RoundID, scheduledTime time.Time, message []byte) error {
					var actualReminderPayload roundevents.DiscordReminderPayload
					json.Unmarshal(message, &actualReminderPayload)
					if actualReminderPayload.DiscordChannelID != "" {
						t.Errorf("expected empty reminder DiscordChannelID, got %q", actualReminderPayload.DiscordChannelID)
					}
					return nil
				})
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string, _ sharedtypes.RoundID, scheduledTime time.Time, message []byte) error {
					var actualStartPayload roundevents.DiscordRoundStartPayload
					json.Unmarshal(message, &actualStartPayload)
					if actualStartPayload.DiscordChannelID != "" {
						t.Errorf("expected empty start DiscordChannelID, got %q", actualStartPayload.DiscordChannelID)
					}
					return nil
				})
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testRoundTitle,
						Description: &testDescription,
						Location:    &testLocation,
						StartTime:   startTimePtr(testStartTime),
						UserID:      testUserID,
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name:             "nil start time in payload",
			payload:          createBasePayload(testStartTime),
			discordMessageID: testEventMessageID,
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				payloadWithNilStartTime := createBasePayload(testStartTime)
				payloadWithNilStartTime.StartTime = nil

				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, time.Time{}).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string, _ sharedtypes.RoundID, scheduledTime time.Time, message []byte) error {
					var actualReminderPayload roundevents.DiscordReminderPayload
					if err := json.Unmarshal(message, &actualReminderPayload); err != nil {
						t.Fatalf("failed to unmarshal reminder payload: %v", err)
					}
					if actualReminderPayload.StartTime != nil {
						t.Errorf("expected nil reminder StartTime, got %v", actualReminderPayload.StartTime)
					}
					return nil
				})
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, time.Time{}, gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string, _ sharedtypes.RoundID, scheduledTime time.Time, message []byte) error {
					if !scheduledTime.Equal(time.Time{}) {
						t.Errorf("expected start time %v, got %v", time.Time{}, scheduledTime)
					}
					var actualStartPayload roundevents.DiscordRoundStartPayload
					if err := json.Unmarshal(message, &actualStartPayload); err != nil {
						t.Fatalf("failed to unmarshal start payload: %v", err)
					}
					if actualStartPayload.StartTime != nil {
						t.Errorf("expected nil start StartTime, got %v", actualStartPayload.StartTime)
					}
					return nil
				})
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testRoundTitle,
						Description: &testDescription,
						Location:    &testLocation,
						StartTime:   nil,
						UserID:      testUserID,
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name:             "nil description in payload",
			payload:          createBasePayload(testStartTime),
			discordMessageID: testEventMessageID,
			mockEventBus: func(mockEB *eventbus.MockEventBus) {
				payloadWithNilDescription := createBasePayload(testStartTime)
				payloadWithNilDescription.Description = nil

				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, testStartTime).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testRoundTitle,
						Description: nil,
						Location:    &testLocation,
						StartTime:   startTimePtr(testStartTime),
						UserID:      testUserID,
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEventBus := eventbus.NewMockEventBus(ctrl)
			tt.mockEventBus(mockEventBus)

			s := &RoundService{
				RoundDB:        mockDB,
				logger:         logger,
				metrics:        mockMetrics,
				tracer:         tracer,
				roundValidator: mockRoundValidator,
				EventBus:       mockEventBus,
				serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			result, err := s.ScheduleRoundEvents(context.Background(), tt.payload, tt.discordMessageID)

			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			if tt.expectedError == nil {
				expectedSuccess, ok := tt.expectedResult.Success.(roundevents.RoundScheduledPayload)
				if !ok {
					t.Fatalf("expected result Success is not of type RoundScheduledPayload")
				}
				actualSuccess, ok := result.Success.(roundevents.RoundScheduledPayload)
				if !ok {
					t.Fatalf("actual result Success is not of type RoundScheduledPayload")
				}

				if expectedSuccess.RoundID != actualSuccess.RoundID {
					t.Errorf("expected result RoundID %v, got %v", expectedSuccess.RoundID, actualSuccess.RoundID)
				}
				if expectedSuccess.Title != actualSuccess.Title {
					t.Errorf("expected result Title %q, got %q", expectedSuccess.Title, actualSuccess.Title)
				}
				if expectedSuccess.Location == nil && actualSuccess.Location != nil ||
					expectedSuccess.Location != nil && actualSuccess.Location == nil ||
					(expectedSuccess.Location != nil && actualSuccess.Location != nil && *expectedSuccess.Location != *actualSuccess.Location) {
					t.Errorf("expected result Location %v, got %v", expectedSuccess.Location, actualSuccess.Location)
				}
				if (expectedSuccess.StartTime == nil && actualSuccess.StartTime != nil) ||
					(expectedSuccess.StartTime != nil && actualSuccess.StartTime == nil) ||
					(expectedSuccess.StartTime != nil && actualSuccess.StartTime != nil && *expectedSuccess.StartTime == (*actualSuccess.StartTime)) {
					t.Errorf("expected result StartTime %v, got %v", expectedSuccess.StartTime, actualSuccess.StartTime)
				}
				if expectedSuccess.EventMessageID != actualSuccess.EventMessageID {
					t.Errorf("expected result EventMessageID %q, got %q", expectedSuccess.EventMessageID, actualSuccess.EventMessageID)
				}
				if (expectedSuccess.Description == nil && actualSuccess.Description != nil) ||
					(expectedSuccess.Description != nil && actualSuccess.Description == nil) ||
					(expectedSuccess.Description != nil && actualSuccess.Description != nil && *expectedSuccess.Description != *actualSuccess.Description) {
					t.Errorf("expected result Description %v, got %v", expectedSuccess.Description, actualSuccess.Description)
				}
				if expectedSuccess.UserID != actualSuccess.UserID {
					t.Errorf("expected result UserID %q, got %q", expectedSuccess.UserID, actualSuccess.UserID)
				}

			} else {
				expectedFailure, ok := tt.expectedResult.Failure.(roundevents.RoundErrorPayload)
				if !ok && tt.expectedResult.Failure != nil {
					t.Fatalf("expected result Failure is not of type RoundErrorPayload")
				}
				actualFailure, ok := result.Failure.(roundevents.RoundErrorPayload)
				if !ok && result.Failure != nil {
					t.Fatalf("actual result Failure is not of type RoundErrorPayload")
				}

				if tt.expectedResult.Failure != nil {
					if expectedFailure.RoundID != actualFailure.RoundID {
						t.Errorf("expected failure RoundID %v, got %v", expectedFailure.RoundID, actualFailure.RoundID)
					}
					if expectedFailure.Error != actualFailure.Error {
						t.Errorf("expected failure Error %q, got %q", expectedFailure.Error, actualFailure.Error)
					}
				} else if result.Failure != nil {
					t.Errorf("expected no failure result, got %v", result.Failure)
				}
			}
		})
	}
}
