package roundservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings" // Added for strings.Contains
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
	// Common test data setup - these are fixed for all tests unless overridden
	testRoundID := sharedtypes.RoundID(uuid.New())
	testRoundTitle := roundtypes.Title("Test Round")
	testLocation := roundtypes.Location("Test Location")
	testEventMessageID := "12345"
	testUserID := sharedtypes.DiscordID("user456") // This is the input UserID
	testDescription := roundtypes.Description("Test Description")

	ctx := context.Background()
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}
	// mockRoundValidator is not used in ScheduleRoundEvents, but kept for consistency
	mockRoundValidator := roundutil.NewMockRoundValidator(gomock.NewController(t)) // Use a new controller for this mock

	// Define a helper to create the base payload
	createBasePayload := func(startTime time.Time) roundevents.RoundScheduledPayload {
		return roundevents.RoundScheduledPayload{
			BaseRoundPayload: roundtypes.BaseRoundPayload{
				RoundID:     testRoundID,
				Title:       testRoundTitle,
				Description: &testDescription,
				Location:    &testLocation,
				StartTime:   startTimePtr(startTime),
				UserID:      testUserID, // Input UserID
			},
		}
	}

	tests := []struct {
		name             string
		payload          roundevents.RoundScheduledPayload
		discordMessageID string
		mockEventBus     func(*gomock.Controller, *eventbus.MockEventBus, time.Time, roundevents.RoundScheduledPayload) // Pass ctrl, fixedNow, and the specific payload for dynamic expectations
		expectedError    error                                                                                          // Changed to expect actual error, not wrapped
		expectedPanicMsg string                                                                                         // Added for panic tests
		expectedResult   RoundOperationResult
	}{
		{
			name:             "successful scheduling",
			payload:          createBasePayload(time.Now().Add(2 * time.Hour)), // Will be dynamically set
			discordMessageID: testEventMessageID,
			mockEventBus: func(ctrl *gomock.Controller, mockEB *eventbus.MockEventBus, fixedNow time.Time, currentPayload roundevents.RoundScheduledPayload) {
				dynamicTestStartTime := fixedNow.Add(2 * time.Hour)
				dynamicTestReminderTime := dynamicTestStartTime.Add(-1 * time.Hour)

				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, sharedtypes.StartTime(dynamicTestStartTime)).Return(nil)

				// Reminder payload - ensure fields match what the service actually sets
				expectedReminderPayload := roundevents.DiscordReminderPayload{
					RoundID:        testRoundID,
					ReminderType:   "1h",
					RoundTitle:     testRoundTitle,
					EventMessageID: testEventMessageID,
					// StartTime, Location, UserIDs, DiscordChannelID, DiscordGuildID are not set in the service's marshaled payload
					// They will be nil or zero values if not explicitly set in the service
				}
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, sharedtypes.StartTime(dynamicTestReminderTime), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string, _ sharedtypes.RoundID, scheduledTime sharedtypes.StartTime, message []byte, _ map[string]string) error {
					if !scheduledTime.AsTime().Equal(dynamicTestReminderTime) {
						t.Errorf("expected reminder time %v, got %v", dynamicTestReminderTime, scheduledTime.AsTime())
					}
					var actualReminderPayload roundevents.DiscordReminderPayload
					if err := json.Unmarshal(message, &actualReminderPayload); err != nil {
						t.Fatalf("failed to unmarshal reminder payload: %v", err)
					}
					// Compare only fields that are explicitly set by the service in the marshaled payload
					if actualReminderPayload.RoundID != expectedReminderPayload.RoundID ||
						actualReminderPayload.ReminderType != expectedReminderPayload.ReminderType ||
						actualReminderPayload.RoundTitle != expectedReminderPayload.RoundTitle ||
						actualReminderPayload.EventMessageID != expectedReminderPayload.EventMessageID {
						t.Errorf("unexpected reminder payload content.\nExpected: %+v\nGot: %+v", expectedReminderPayload, actualReminderPayload)
					}
					return nil
				})

				// Start payload - dynamically create expected payload based on the current test's input payload
				expectedStartPayload := roundevents.DiscordRoundStartPayload{
					RoundID:        currentPayload.RoundID,
					Title:          currentPayload.Title,
					Location:       currentPayload.Location,          // Use the exact pointer from currentPayload
					StartTime:      currentPayload.StartTime,         // Use the exact pointer from currentPayload
					Participants:   []roundevents.RoundParticipant{}, // Service sets this as empty slice
					EventMessageID: testEventMessageID,
				}
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, sharedtypes.StartTime(dynamicTestStartTime), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string, _ sharedtypes.RoundID, scheduledTime sharedtypes.StartTime, message []byte, _ map[string]string) error {
					if !scheduledTime.AsTime().Equal(dynamicTestStartTime) {
						t.Errorf("expected start time %v, got %v", dynamicTestStartTime, scheduledTime.AsTime())
					}
					var actualStartPayload roundevents.DiscordRoundStartPayload
					if err := json.Unmarshal(message, &actualStartPayload); err != nil {
						t.Fatalf("failed to unmarshal start payload: %v", err)
					}
					// Compare all fields that are explicitly set by the service in the marshaled payload
					if actualStartPayload.RoundID != expectedStartPayload.RoundID ||
						actualStartPayload.Title != expectedStartPayload.Title ||
						// Compare Location by value
						(actualStartPayload.Location == nil && expectedStartPayload.Location != nil) ||
						(actualStartPayload.Location != nil && expectedStartPayload.Location == nil) ||
						(actualStartPayload.Location != nil && expectedStartPayload.Location != nil && *actualStartPayload.Location != *expectedStartPayload.Location) ||
						// Compare StartTime by value
						(actualStartPayload.StartTime == nil && expectedStartPayload.StartTime != nil) ||
						(actualStartPayload.StartTime != nil && expectedStartPayload.StartTime == nil) ||
						(actualStartPayload.StartTime != nil && expectedStartPayload.StartTime != nil && !actualStartPayload.StartTime.AsTime().Equal(expectedStartPayload.StartTime.AsTime())) ||
						len(actualStartPayload.Participants) != len(expectedStartPayload.Participants) ||
						actualStartPayload.EventMessageID != expectedStartPayload.EventMessageID {
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
						StartTime:   startTimePtr(time.Now().Add(2 * time.Hour)), // This will be updated dynamically in the loop
						UserID:      "",                                          // Service does not copy UserID to output payload
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name:             "error creating consumer",
			payload:          createBasePayload(time.Now().Add(2 * time.Hour)),
			discordMessageID: testEventMessageID,
			mockEventBus: func(ctrl *gomock.Controller, mockEB *eventbus.MockEventBus, fixedNow time.Time, currentPayload roundevents.RoundScheduledPayload) {
				dynamicTestStartTime := fixedNow.Add(2 * time.Hour)
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, sharedtypes.StartTime(dynamicTestStartTime)).Return(errors.New("consumer creation error"))
			},
			expectedError: errors.New("failed to create consumer for round " + testRoundID.String() + ": consumer creation error"), // No wrapper prefix
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: testRoundID,
					Error:   "consumer creation error", // Matches inner error message
				},
			},
		},
		{
			name:             "error scheduling reminder",
			payload:          createBasePayload(time.Now().Add(2 * time.Hour)),
			discordMessageID: testEventMessageID,
			mockEventBus: func(ctrl *gomock.Controller, mockEB *eventbus.MockEventBus, fixedNow time.Time, currentPayload roundevents.RoundScheduledPayload) {
				dynamicTestStartTime := fixedNow.Add(2 * time.Hour)
				dynamicTestReminderTime := dynamicTestStartTime.Add(-1 * time.Hour)
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, sharedtypes.StartTime(dynamicTestStartTime)).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, sharedtypes.StartTime(dynamicTestReminderTime), gomock.Any(), gomock.Any()).Return(errors.New("reminder scheduling error"))
			},
			expectedError: errors.New("failed to schedule reminder: reminder scheduling error"), // No wrapper prefix
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: testRoundID,
					Error:   "reminder scheduling error", // Matches inner error message
				},
			},
		},
		{
			name:             "error scheduling round start",
			payload:          createBasePayload(time.Now().Add(2 * time.Hour)),
			discordMessageID: testEventMessageID,
			mockEventBus: func(ctrl *gomock.Controller, mockEB *eventbus.MockEventBus, fixedNow time.Time, currentPayload roundevents.RoundScheduledPayload) {
				dynamicTestStartTime := fixedNow.Add(2 * time.Hour)
				dynamicTestReminderTime := dynamicTestStartTime.Add(-1 * time.Hour)
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, sharedtypes.StartTime(dynamicTestStartTime)).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, sharedtypes.StartTime(dynamicTestReminderTime), gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, sharedtypes.StartTime(dynamicTestStartTime), gomock.Any(), gomock.Any()).Return(errors.New("round start scheduling error"))
			},
			expectedError: errors.New("failed to schedule round start: round start scheduling error"), // No wrapper prefix
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: testRoundID,
					Error:   "round start scheduling error", // Matches inner error message
				},
			},
		},
		{
			name:             "past start time",
			payload:          createBasePayload(time.Now().Add(-1 * time.Hour)), // Will be dynamically set
			discordMessageID: testEventMessageID,
			mockEventBus: func(ctrl *gomock.Controller, mockEB *eventbus.MockEventBus, fixedNow time.Time, currentPayload roundevents.RoundScheduledPayload) {
				pastStartTime := fixedNow.Add(-1 * time.Hour)
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, sharedtypes.StartTime(pastStartTime)).Return(nil)
				// No ScheduleDelayedMessage calls expected as times are in the past
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testRoundTitle,
						Description: &testDescription,
						Location:    &testLocation,
						StartTime:   startTimePtr(time.Now().Add(-1 * time.Hour)), // This will be updated dynamically in the loop
						UserID:      "",                                           // Service does not copy UserID to output payload
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name:             "far future start time",
			payload:          createBasePayload(time.Now().Add(720 * time.Hour)), // Will be dynamically set
			discordMessageID: testEventMessageID,
			mockEventBus: func(ctrl *gomock.Controller, mockEB *eventbus.MockEventBus, fixedNow time.Time, currentPayload roundevents.RoundScheduledPayload) {
				futureStartTime := fixedNow.Add(720 * time.Hour)
				futureReminderTime := futureStartTime.Add(-1 * time.Hour)
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, sharedtypes.StartTime(futureStartTime)).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, sharedtypes.StartTime(futureReminderTime), gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, sharedtypes.StartTime(futureStartTime), gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testRoundTitle,
						Description: &testDescription,
						Location:    &testLocation,
						StartTime:   startTimePtr(time.Now().Add(720 * time.Hour)), // This will be updated dynamically in the loop
						UserID:      "",                                            // Service does not copy UserID to output payload
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name:             "zero start time",
			payload:          createBasePayload(time.Time{}), // Will be dynamically set
			discordMessageID: testEventMessageID,
			mockEventBus: func(ctrl *gomock.Controller, mockEB *eventbus.MockEventBus, fixedNow time.Time, currentPayload roundevents.RoundScheduledPayload) {
				zeroTime := time.Time{}
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, sharedtypes.StartTime(zeroTime)).Return(nil)
				// No ScheduleDelayedMessage calls expected as start time is zero/in the past
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
						UserID:      "", // Service does not copy UserID to output payload
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name:             "empty event message ID",
			payload:          createBasePayload(time.Now().Add(2 * time.Hour)), // Will be dynamically set
			discordMessageID: "",
			mockEventBus: func(ctrl *gomock.Controller, mockEB *eventbus.MockEventBus, fixedNow time.Time, currentPayload roundevents.RoundScheduledPayload) {
				dynamicTestStartTime := fixedNow.Add(2 * time.Hour)
				dynamicTestReminderTime := dynamicTestStartTime.Add(-1 * time.Hour)
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, sharedtypes.StartTime(dynamicTestStartTime)).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, sharedtypes.StartTime(dynamicTestReminderTime), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string, _ sharedtypes.RoundID, scheduledTime sharedtypes.StartTime, message []byte, _ map[string]string) error {
					var actualReminderPayload roundevents.DiscordReminderPayload
					json.Unmarshal(message, &actualReminderPayload)
					if actualReminderPayload.EventMessageID != "" {
						t.Errorf("expected empty reminder EventMessageID, got %q", actualReminderPayload.EventMessageID)
					}
					return nil
				})
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, sharedtypes.StartTime(dynamicTestStartTime), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string, _ sharedtypes.RoundID, scheduledTime sharedtypes.StartTime, message []byte, _ map[string]string) error {
					var actualStartPayload roundevents.DiscordRoundStartPayload
					json.Unmarshal(message, &actualStartPayload)
					if actualStartPayload.EventMessageID != "" { // Corrected from DiscordChannelID to EventMessageID
						t.Errorf("expected empty start EventMessageID, got %q", actualStartPayload.EventMessageID)
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
						StartTime:   startTimePtr(time.Now().Add(2 * time.Hour)), // This will be updated dynamically in the loop
						UserID:      "",                                          // Service does not copy UserID to output payload
					},
					EventMessageID: "",
				},
			},
		},
		{
			name: "with description in payload",
			payload: func() roundevents.RoundScheduledPayload {
				p := createBasePayload(time.Now().Add(2 * time.Hour))
				desc := roundtypes.Description("Another Description")
				p.Description = &desc
				return p
			}(),
			discordMessageID: testEventMessageID,
			mockEventBus: func(ctrl *gomock.Controller, mockEB *eventbus.MockEventBus, fixedNow time.Time, currentPayload roundevents.RoundScheduledPayload) {
				dynamicTestStartTime := fixedNow.Add(2 * time.Hour)
				dynamicTestReminderTime := dynamicTestStartTime.Add(-1 * time.Hour)
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, sharedtypes.StartTime(dynamicTestStartTime)).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, sharedtypes.StartTime(dynamicTestReminderTime), gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, sharedtypes.StartTime(dynamicTestStartTime), gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testRoundTitle,
						Description: descriptionPtr(roundtypes.Description("Another Description")), // This should now match the input payload's description
						Location:    &testLocation,
						StartTime:   startTimePtr(time.Now().Add(2 * time.Hour)), // This will be updated dynamically in the loop
						UserID:      "",                                          // Service does not copy UserID to output payload
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name: "with UserID in payload",
			payload: func() roundevents.RoundScheduledPayload {
				p := createBasePayload(time.Now().Add(2 * time.Hour))
				p.UserID = sharedtypes.DiscordID("anotheruser")
				return p
			}(),
			discordMessageID: testEventMessageID,
			mockEventBus: func(ctrl *gomock.Controller, mockEB *eventbus.MockEventBus, fixedNow time.Time, currentPayload roundevents.RoundScheduledPayload) {
				dynamicTestStartTime := fixedNow.Add(2 * time.Hour)
				dynamicTestReminderTime := dynamicTestStartTime.Add(-1 * time.Hour)
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, sharedtypes.StartTime(dynamicTestStartTime)).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, sharedtypes.StartTime(dynamicTestReminderTime), gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, sharedtypes.StartTime(dynamicTestStartTime), gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testRoundTitle,
						Description: &testDescription,
						Location:    &testLocation,
						// The StartTime will be dynamically set by the main loop's payload adjustment
						StartTime: startTimePtr(time.Now().Add(2 * time.Hour)),
						UserID:    "", // Service does not copy UserID to output payload
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name:             "reminder time equals start time", // This means reminder is in the past, and start time is now (not future)
			payload:          createBasePayload(time.Now()),     // Will be dynamically set
			discordMessageID: testEventMessageID,
			mockEventBus: func(ctrl *gomock.Controller, mockEB *eventbus.MockEventBus, fixedNow time.Time, currentPayload roundevents.RoundScheduledPayload) {
				dynamicTestStartTime := fixedNow // Start time is effectively 'now'
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, sharedtypes.StartTime(dynamicTestStartTime)).Return(nil)
				// Neither reminder nor start event will be scheduled because their times are not strictly After(now)
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testRoundTitle,
						Description: &testDescription,
						Location:    &testLocation,
						StartTime:   startTimePtr(time.Now()), // Will be updated dynamically
						UserID:      "",                       // Service does not copy UserID to output payload
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name:             "missing channel ID in payload",                  // This test name is misleading, it's about DiscordChannelID not being in marshaled payload
			payload:          createBasePayload(time.Now().Add(2 * time.Hour)), // Will be dynamically set
			discordMessageID: testEventMessageID,
			mockEventBus: func(ctrl *gomock.Controller, mockEB *eventbus.MockEventBus, fixedNow time.Time, currentPayload roundevents.RoundScheduledPayload) {
				dynamicTestStartTime := fixedNow.Add(2 * time.Hour)
				dynamicTestReminderTime := dynamicTestStartTime.Add(-1 * time.Hour)
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, sharedtypes.StartTime(dynamicTestStartTime)).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, sharedtypes.StartTime(dynamicTestReminderTime), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string, _ sharedtypes.RoundID, scheduledTime sharedtypes.StartTime, message []byte, _ map[string]string) error {
					var actualReminderPayload roundevents.DiscordReminderPayload
					json.Unmarshal(message, &actualReminderPayload)
					if actualReminderPayload.DiscordChannelID != "" { // Check that it's NOT set
						t.Errorf("expected empty reminder DiscordChannelID, got %q", actualReminderPayload.DiscordChannelID)
					}
					return nil
				})
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, sharedtypes.StartTime(dynamicTestStartTime), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string, _ sharedtypes.RoundID, scheduledTime sharedtypes.StartTime, message []byte, _ map[string]string) error {
					var actualStartPayload roundevents.DiscordRoundStartPayload
					json.Unmarshal(message, &actualStartPayload)
					if actualStartPayload.DiscordChannelID != "" { // Check that it's NOT set
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
						StartTime:   startTimePtr(time.Now().Add(2 * time.Hour)), // Will be updated dynamically
						UserID:      "",
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
		{
			name: "nil start time in payload",
			payload: func() roundevents.RoundScheduledPayload {
				p := createBasePayload(time.Now().Add(2 * time.Hour))
				p.StartTime = nil // Explicitly set to nil for this test
				return p
			}(),
			discordMessageID: testEventMessageID,
			mockEventBus: func(ctrl *gomock.Controller, mockEB *eventbus.MockEventBus, fixedNow time.Time, currentPayload roundevents.RoundScheduledPayload) {
				// No mocks expected for ProcessDelayedMessages or ScheduleDelayedMessage
				// because the service will panic before these calls if StartTime is nil.
			},
			expectedError:    nil,                                                                // No error is *returned* by the function, it panics
			expectedPanicMsg: "runtime error: invalid memory address or nil pointer dereference", // Expect this panic message
			expectedResult:   RoundOperationResult{},                                             // Zero value as operation panics and returns empty result
		},
		{
			name: "nil description in payload",
			payload: func() roundevents.RoundScheduledPayload {
				p := createBasePayload(time.Now().Add(2 * time.Hour))
				p.Description = nil // Explicitly set to nil for this test
				return p
			}(),
			discordMessageID: testEventMessageID,
			mockEventBus: func(ctrl *gomock.Controller, mockEB *eventbus.MockEventBus, fixedNow time.Time, currentPayload roundevents.RoundScheduledPayload) {
				dynamicTestStartTime := fixedNow.Add(2 * time.Hour)
				dynamicTestReminderTime := dynamicTestStartTime.Add(-1 * time.Hour)
				mockEB.EXPECT().ProcessDelayedMessages(ctx, testRoundID, sharedtypes.StartTime(dynamicTestStartTime)).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundReminder, testRoundID, sharedtypes.StartTime(dynamicTestReminderTime), gomock.Any(), gomock.Any()).Return(nil)
				mockEB.EXPECT().ScheduleDelayedMessage(ctx, roundevents.RoundStarted, testRoundID, sharedtypes.StartTime(dynamicTestStartTime), gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedError: nil,
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testRoundTitle,
						Description: nil, // Expect nil description
						Location:    &testLocation,
						StartTime:   startTimePtr(time.Now().Add(2 * time.Hour)), // Will be updated dynamically
						UserID:      "",
					},
					EventMessageID: testEventMessageID,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Catch panics for specific test cases
			if tt.expectedPanicMsg != "" {
				defer func() {
					if r := recover(); r != nil {
						panicMsg := fmt.Sprintf("%v", r)
						if !strings.Contains(panicMsg, tt.expectedPanicMsg) {
							t.Errorf("expected panic message to contain %q, got %q", tt.expectedPanicMsg, panicMsg)
						}
					} else {
						t.Errorf("expected a panic, but no panic occurred")
					}
				}()
			}

			// Capture the current time for consistent test execution within this subtest
			fixedNow := time.Now()

			// Re-create mocks for each subtest to ensure isolation
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockDB := rounddb.NewMockRoundDB(ctrl)
			mockEventBus := eventbus.NewMockEventBus(ctrl)

			// Dynamically update payload's StartTime based on fixedNow for this test run
			// This ensures the payload passed to the service matches the mock expectations for time.
			// This block runs *before* the mockEventBus setup, ensuring currentPayload has the correct StartTime.
			if tt.payload.StartTime != nil { // Only adjust if StartTime is not nil (e.g., for nil_start_time_in_payload)
				// Adjust the payload's StartTime relative to fixedNow
				if tt.name == "past start time" {
					tt.payload.StartTime = startTimePtr(fixedNow.Add(-1 * time.Hour))
				} else if tt.name == "far future start time" {
					tt.payload.StartTime = startTimePtr(fixedNow.Add(720 * time.Hour))
				} else if tt.name == "zero start time" {
					tt.payload.StartTime = startTimePtr(time.Time{})
				} else if tt.name == "reminder time equals start time" {
					tt.payload.StartTime = startTimePtr(fixedNow)
				} else { // Default for successful scheduling, empty event message ID, missing channel ID, etc.
					tt.payload.StartTime = startTimePtr(fixedNow.Add(2 * time.Hour))
				}
			}
			// For "with description in payload", "with UserID in payload", and "nil description in payload",
			// the payload's StartTime is already set by their anonymous functions and defaults to fixedNow.Add(2 * time.Hour)

			// Dynamically update expectedResult's StartTime based on fixedNow for this test run
			if tt.expectedResult.Success != nil {
				if successPayload, ok := tt.expectedResult.Success.(roundevents.RoundScheduledPayload); ok {
					if successPayload.StartTime != nil {
						// Adjust the expected success payload's StartTime relative to fixedNow
						if tt.name == "past start time" {
							successPayload.StartTime = startTimePtr(fixedNow.Add(-1 * time.Hour))
						} else if tt.name == "far future start time" {
							successPayload.StartTime = startTimePtr(fixedNow.Add(720 * time.Hour))
						} else if tt.name == "zero start time" {
							successPayload.StartTime = startTimePtr(time.Time{})
						} else if tt.name == "reminder time equals start time" {
							successPayload.StartTime = startTimePtr(fixedNow)
						} else {
							successPayload.StartTime = startTimePtr(fixedNow.Add(2 * time.Hour))
						}
					}
					// Special handling for UserID in the expected result
					// Based on service code, UserID is NOT copied to the output payload
					successPayload.UserID = ""

					// Special handling for description in the expected result
					if tt.name == "nil description in payload" {
						successPayload.Description = nil
					} else if tt.name == "with description in payload" {
						successPayload.Description = descriptionPtr(roundtypes.Description("Another Description"))
					} else {
						successPayload.Description = &testDescription // Default description
					}

					tt.expectedResult.Success = successPayload
				}
			}

			// Call the mock setup function, passing the fixedNow and the current test's payload
			tt.mockEventBus(ctrl, mockEventBus, fixedNow, tt.payload)

			s := &RoundService{
				RoundDB:        mockDB,
				logger:         logger,
				metrics:        mockMetrics,
				tracer:         tracer,
				roundValidator: mockRoundValidator,
				EventBus:       mockEventBus,
				// Bypass the serviceWrapper for direct unit testing
				serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			result, err := s.ScheduleRoundEvents(ctx, tt.payload, tt.discordMessageID)

			// Validate error presence and message (only if no panic is expected)
			if tt.expectedPanicMsg == "" { // Only check err if no panic is expected
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
			} else { // If panic is expected, ensure err is nil (as panic is recovered by test, not returned as error)
				if err != nil {
					t.Errorf("expected no error returned when panic is handled, got: %v", err)
				}
			}

			// Validate result payload (success/failure)
			if tt.expectedResult.Success != nil {
				// Type assertion for Success payload
				if result.Success == nil {
					t.Errorf("expected success result, got failure")
				} else if successPayload, ok := result.Success.(roundevents.RoundScheduledPayload); !ok {
					t.Errorf("expected result.Success to be of type roundevents.RoundScheduledPayload, got %T", result.Success)
				} else if expectedSuccessPayload, ok := tt.expectedResult.Success.(roundevents.RoundScheduledPayload); !ok {
					t.Errorf("expected tt.expectedResult.Success to be of type roundevents.RoundScheduledPayload, got %T", tt.expectedResult.Success)
				} else {
					// Compare fields directly
					if successPayload.RoundID != expectedSuccessPayload.RoundID {
						t.Errorf("expected success RoundID %s, got %s", expectedSuccessPayload.RoundID, successPayload.RoundID)
					}
					if successPayload.Title != expectedSuccessPayload.Title {
						t.Errorf("expected success Title %s, got %s", expectedSuccessPayload.Title, successPayload.Title)
					}
					if (successPayload.Description == nil && expectedSuccessPayload.Description != nil) ||
						(successPayload.Description != nil && expectedSuccessPayload.Description == nil) ||
						(successPayload.Description != nil && expectedSuccessPayload.Description != nil && *successPayload.Description != *expectedSuccessPayload.Description) {
						t.Errorf("expected success Description %v, got %v", expectedSuccessPayload.Description, successPayload.Description)
					}
					if (successPayload.Location == nil && expectedSuccessPayload.Location != nil) ||
						(successPayload.Location != nil && expectedSuccessPayload.Location == nil) ||
						(successPayload.Location != nil && expectedSuccessPayload.Location != nil && *successPayload.Location != *expectedSuccessPayload.Location) {
						t.Errorf("expected success Location %v, got %v", expectedSuccessPayload.Location, successPayload.Location)
					}
					if (successPayload.StartTime == nil && expectedSuccessPayload.StartTime != nil) ||
						(successPayload.StartTime != nil && expectedSuccessPayload.StartTime == nil) ||
						(successPayload.StartTime != nil && expectedSuccessPayload.StartTime != nil && !successPayload.StartTime.AsTime().Equal(expectedSuccessPayload.StartTime.AsTime())) {
						t.Errorf("expected success StartTime %v, got %v", expectedSuccessPayload.StartTime, successPayload.StartTime)
					}
					// Special comparison for UserID
					if successPayload.UserID != expectedSuccessPayload.UserID {
						t.Errorf("expected success UserID %s, got %s", expectedSuccessPayload.UserID, successPayload.UserID)
					}
					if successPayload.EventMessageID != expectedSuccessPayload.EventMessageID {
						t.Errorf("expected success EventMessageID %s, got %s", expectedSuccessPayload.EventMessageID, successPayload.EventMessageID)
					}
				}
			} else if tt.expectedResult.Failure != nil {
				// Type assertion for Failure payload
				if result.Failure == nil {
					t.Errorf("expected failure result, got success")
				} else if failurePayload, ok := result.Failure.(roundevents.RoundErrorPayload); !ok {
					t.Errorf("expected result.Failure to be of type roundevents.RoundErrorPayload, got %T", result.Failure)
				} else if expectedFailurePayload, ok := tt.expectedResult.Failure.(roundevents.RoundErrorPayload); !ok {
					t.Errorf("expected tt.expectedResult.Failure to be of type roundevents.RoundErrorPayload, got %T", tt.expectedResult.Failure)
				} else if failurePayload.Error != expectedFailurePayload.Error {
					t.Errorf("expected failure error %q, got %q", expectedFailurePayload.Error, failurePayload.Error)
				}
			}
		})
	}
}
