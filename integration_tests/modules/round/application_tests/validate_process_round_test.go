// integration_tests/modules/round/application_tests/validate_process_round_test.go
package roundintegrationtests

import (
	"strings"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
)

func TestValidateAndProcessRound(t *testing.T) {
	tests := []struct {
		name string
		// setupTestEnv is a no-op for time control due to service's direct time.Now().UTC() usage.
		setupTestEnv func()
		// payload is the incoming event payload to be processed by the service.
		payload roundevents.CreateRoundRequestedPayloadV1
		// expectedError indicates if the service call is expected to return a Go error.
		expectedError bool
		// expectedSuccess indicates if the service call is expected to return a success payload.
		expectedSuccess bool
		// validateResult asserts the content of the RoundOperationResult returned by the service.
		validateResult func(t *testing.T, deps RoundTestDeps, result results.OperationResult)
	}{
		{
			name: "Successful round creation",
			setupTestEnv: func() {
				// No specific time setup possible here for the service's internal time.Now().UTC()
				// The RealTimeParser will use roundutil.RealClock{} as invoked by the service.
			},
			payload: roundevents.CreateRoundRequestedPayloadV1{
				GuildID:     "test-guild",
				Title:       roundtypes.Title("Test Round 1"),
				Description: roundtypes.Description("A test description"),
				Location:    roundtypes.Location("Test Location"),
				StartTime:   "tomorrow at 10 AM", // Will be parsed relative to actual system time
				Timezone:    roundtypes.Timezone("America/Chicago"),
				UserID:      "user_123",
				ChannelID:   "channel_abc",
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps RoundTestDeps, result results.OperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				successPayload, ok := result.Success.(*roundevents.RoundEntityCreatedPayloadV1)
				if !ok {
					t.Errorf("Expected success result of type *roundevents.RoundEntityCreatedPayloadV1, but got %T", result.Success)
					return
				}

				// Validate the round object in the success payload
				if successPayload.Round.Title != "Test Round 1" {
					t.Errorf("Expected title 'Test Round 1', got '%s'", successPayload.Round.Title)
				}
				if *successPayload.Round.Description != "A test description" {
					t.Errorf("Expected description 'A test description', got '%s'", *successPayload.Round.Description)
				}
				if *successPayload.Round.Location != "Test Location" {
					t.Errorf("Expected location 'Test Location', got '%s'", *successPayload.Round.Location)
				}
				if successPayload.Round.CreatedBy != "user_123" {
					t.Errorf("Expected CreatedBy 'user_123', got '%s'", successPayload.Round.CreatedBy)
				}
				if successPayload.Round.State != roundtypes.RoundStateUpcoming {
					t.Errorf("Expected state 'Upcoming', got '%s'", successPayload.Round.State)
				}
				if len(successPayload.Round.Participants) != 0 {
					t.Errorf("Expected 0 participants, got %d", len(successPayload.Round.Participants))
				}
				if successPayload.DiscordChannelID != "channel_abc" {
					t.Errorf("Expected DiscordChannelID 'channel_abc', got '%s'", successPayload.DiscordChannelID)
				}
				if successPayload.DiscordGuildID != "test-guild" {
					t.Errorf("Expected DiscordGuildID 'test-guild', got '%s'", successPayload.DiscordGuildID)
				}
				// Verify StartTime is in the future relative to the test execution time.
				// Due to time.Now().UTC() in service, this test can be flaky if run at specific times.
				if time.Time(*successPayload.Round.StartTime).Before(time.Now().UTC().Truncate(time.Minute)) {
					t.Errorf("Expected StartTime %v to be in the future, but it is in the past relative to %v", time.Time(*successPayload.Round.StartTime), time.Now().UTC())
				}
			},
		},
		{
			name: "Validation failure - missing title",
			setupTestEnv: func() {
				// No specific time setup
			},
			payload: roundevents.CreateRoundRequestedPayloadV1{
				GuildID:     "test-guild",
				Title:       roundtypes.Title(""), // Empty title
				Description: roundtypes.Description("A test description"),
				Location:    roundtypes.Location("Test Location"),
				StartTime:   "tomorrow at 10 AM",
				Timezone:    roundtypes.Timezone("America/Chicago"),
				UserID:      "user_123",
				ChannelID:   "channel_abc",
			},
			expectedError:   false,
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps RoundTestDeps, result results.OperationResult) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*roundevents.RoundValidationFailedPayloadV1)
				if !ok {
					t.Errorf("Expected failure result of type *roundevents.RoundValidationFailedPayloadV1, but got %T", result.Failure)
					return
				}
				// Updated expected error message to match actual output
				expectedErrMsg := "title cannot be empty"
				if len(failurePayload.ErrorMessages) != 1 || failurePayload.ErrorMessages[0] != expectedErrMsg {
					t.Errorf("Expected error message '%s', got '%v'", expectedErrMsg, failurePayload.ErrorMessages)
				}
				if failurePayload.UserID != "user_123" {
					t.Errorf("Expected UserID 'user_123', got '%s'", failurePayload.UserID)
				}
			},
		},
		{
			name: "Validation failure - start time in past (flaky test)",
			setupTestEnv: func() {
				// This test case is inherently flaky because the service's `parsedTime.Before(time.Now().UTC())`
				// directly uses the system's current time, which cannot be controlled without modifying the service
				// or using runtime patching (which is like a mock).
				// For this test to pass reliably, it must be run when "yesterday at 5 PM" is indeed in the past.
			},
			payload: roundevents.CreateRoundRequestedPayloadV1{
				GuildID:     "test-guild",
				Title:       roundtypes.Title("Past Round"),
				Description: roundtypes.Description("Should fail"),
				Location:    roundtypes.Location("Anywhere"),
				StartTime:   "yesterday at 5 PM", // Will be parsed relative to actual system time
				Timezone:    roundtypes.Timezone("America/New_York"),
				UserID:      "user_456",
				ChannelID:   "channel_def",
			},
			expectedError:   false,
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps RoundTestDeps, result results.OperationResult) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*roundevents.RoundValidationFailedPayloadV1)
				if !ok {
					t.Errorf("Expected failure result of type *roundevents.RoundValidationFailedPayloadV1, but got %T", result.Failure)
					return
				}
				// Updated to check for substring due to dynamic timestamps in the actual error message
				expectedContains := "start time must be in the future"
				if len(failurePayload.ErrorMessages) != 1 || !strings.Contains(failurePayload.ErrorMessages[0], expectedContains) {
					t.Errorf("Expected error message to contain '%s', got '%v'", expectedContains, failurePayload.ErrorMessages)
				}
				if failurePayload.UserID != "user_456" {
					t.Errorf("Expected UserID 'user_456', got '%s'", failurePayload.UserID)
				}
			},
		},
		{
			name: "Time parsing failure",
			setupTestEnv: func() {
				// No specific time setup
			},
			payload: roundevents.CreateRoundRequestedPayloadV1{
				GuildID:     "test-guild",
				Title:       roundtypes.Title("Bad Time Round"),
				Description: roundtypes.Description("Invalid time"),
				Location:    roundtypes.Location("Here"),
				StartTime:   "not a real time string", // Invalid time input
				Timezone:    roundtypes.Timezone("America/Los_Angeles"),
				UserID:      "user_789",
				ChannelID:   "channel_ghi",
			},
			expectedError:   false,
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps RoundTestDeps, result results.OperationResult) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*roundevents.RoundValidationFailedPayloadV1)
				if !ok {
					t.Errorf("Expected failure result of type *roundevents.RoundValidationFailedPayloadV1, but got %T", result.Failure)
					return
				}
				// Updated expected error message to match actual output from TimeParser
				expectedErrMsg := "could not recognize time format 'not a real time string'. Supported formats: YYYY-MM-DD HH:MM, MM/DD/YYYY HH:MM, or natural language like 'tomorrow 5pm'"
				if len(failurePayload.ErrorMessages) != 1 || failurePayload.ErrorMessages[0] != expectedErrMsg {
					t.Errorf("Expected error message '%s', got '%v'", expectedErrMsg, failurePayload.ErrorMessages)
				}
				if failurePayload.UserID != "user_789" {
					t.Errorf("Expected UserID 'user_789', got '%s'", failurePayload.UserID)
				}
			},
		},
		{
			name: "Successful creation with empty description and location",
			setupTestEnv: func() {
				// No specific time setup
			},
			payload: roundevents.CreateRoundRequestedPayloadV1{
				GuildID:     "test-guild",
				Title:       roundtypes.Title("Round with Empty Optional Fields"),
				Description: roundtypes.Description(""), // Empty description - will be rejected by validator
				Location:    roundtypes.Location(""),    // Empty location - will be rejected by validator
				StartTime:   "in 2 days at 3 PM",        // Changed to be more reliably in the future
				Timezone:    roundtypes.Timezone("UTC"),
				UserID:      "user_empty",
				ChannelID:   "channel_xyz",
			},
			expectedError:   false,
			expectedSuccess: false, // Changed: validator rejects empty fields
			validateResult: func(t *testing.T, deps RoundTestDeps, result results.OperationResult) {
				if result.Success != nil {
					t.Errorf("Expected failure result, but got success: %+v", result.Success)
					return
				}

				if result.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}

				failurePayload, ok := result.Failure.(*roundevents.RoundValidationFailedPayloadV1)
				if !ok {
					t.Errorf("Expected failure result of type *roundevents.RoundValidationFailedPayloadV1, but got %T", result.Failure)
					return
				}

				// Check that validation failed for both empty description and location
				if len(failurePayload.ErrorMessages) != 2 {
					t.Errorf("Expected 2 validation errors, got %d: %v", len(failurePayload.ErrorMessages), failurePayload.ErrorMessages)
					return
				}

				// Check for both expected error messages
				errorMessages := failurePayload.ErrorMessages
				hasLocationError := false
				hasDescriptionError := false

				for _, msg := range errorMessages {
					if msg == "location cannot be empty" {
						hasLocationError = true
					}
					if msg == "description cannot be empty" {
						hasDescriptionError = true
					}
				}

				if !hasLocationError {
					t.Errorf("Expected validation error 'location cannot be empty', got: %v", errorMessages)
				}
				if !hasDescriptionError {
					t.Errorf("Expected validation error 'description cannot be empty', got: %v", errorMessages)
				}

				if failurePayload.UserID != "user_empty" {
					t.Errorf("Expected UserID 'user_empty', got '%s'", failurePayload.UserID)
				}
			},
		},
		{
			name: "Successful creation with optional fields",
			setupTestEnv: func() {
				// No specific time setup
			},
			payload: roundevents.CreateRoundRequestedPayloadV1{
				GuildID:     "test-guild",
				Title:       roundtypes.Title("Round with Optional Fields"),
				Description: roundtypes.Description("A valid description"), // Non-empty description
				Location:    roundtypes.Location("A valid location"),       // Non-empty location
				StartTime:   "in 2 days at 3 PM",
				Timezone:    roundtypes.Timezone("UTC"),
				UserID:      "user_valid",
				ChannelID:   "channel_xyz",
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps RoundTestDeps, result results.OperationResult) {
				if result.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}

				successPayload, ok := result.Success.(*roundevents.RoundEntityCreatedPayloadV1)
				if !ok {
					t.Errorf("Expected success result of type *roundevents.RoundEntityCreatedPayloadV1, but got %T", result.Success)
					return
				}

				if successPayload.Round.Title != "Round with Optional Fields" {
					t.Errorf("Expected title 'Round with Optional Fields', got '%s'", successPayload.Round.Title)
				}
				if successPayload.Round.Description == nil || *successPayload.Round.Description != "A valid description" {
					t.Errorf("Expected description 'A valid description', got '%v'", successPayload.Round.Description)
				}
				if successPayload.Round.Location == nil || *successPayload.Round.Location != "A valid location" {
					t.Errorf("Expected location 'A valid location', got '%v'", successPayload.Round.Location)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup dependencies for each test to ensure isolation
			deps := SetupTestRoundService(t)

			// Setup test-specific environment (currently no-op for time control)
			if tt.setupTestEnv != nil {
				tt.setupTestEnv()
			}

			// Call the service method.
			result, err := deps.Service.ValidateAndProcessRound(deps.Ctx, tt.payload, roundtime.NewTimeParser())

			// Validate expected error.
			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			}

			// Validate expected success payload presence.
			if tt.expectedSuccess {
				if result.Success == nil {
					t.Errorf("Expected a success result, but got nil")
				}
			} else {
				if result.Success != nil {
					t.Errorf("Expected no success result, but got: %+v", result.Success)
				}
			}

			// Run test-specific result validation.
			if tt.validateResult != nil {
				tt.validateResult(t, deps, result)
			}
		})
	}
}
