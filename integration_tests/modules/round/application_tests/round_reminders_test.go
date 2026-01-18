package roundintegrationtests

import (
	"context"
	"strings"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

// TestProcessRoundReminder is the main integration test function for the ProcessRoundReminder service method.
func TestProcessRoundReminder(t *testing.T) {
	nonexistentRoundID := sharedtypes.RoundID(uuid.New())
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) roundevents.DiscordReminderPayloadV1
		expectedError            bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult)
	}{
		{
			name: "Successful processing with accepted and tentative participants",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) roundevents.DiscordReminderPayloadV1 {
				generator := testutils.NewTestDataGenerator()

				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("creator1"),
					Title:     "Reminder Test Round",
					State:     roundtypes.RoundStateUpcoming,
				})

				description := roundtypes.Description("This is a round for reminder testing.")
				round.Description = &description
				location := roundtypes.Location("Test Location")
				round.Location = &location
				eventType := roundtypes.EventType("Practice")
				round.EventType = &eventType

				startTime := sharedtypes.StartTime(time.Now().Add(24 * time.Hour))
				round.StartTime = &startTime

				// Participants with different responses
				tag1 := sharedtypes.TagNumber(10)
				tag2 := sharedtypes.TagNumber(20)
				round.Participants = []roundtypes.Participant{
					{UserID: sharedtypes.DiscordID("user_accepted_1"), TagNumber: &tag1, Response: roundtypes.ResponseAccept},
					{UserID: sharedtypes.DiscordID("user_tentative_1"), TagNumber: &tag2, Response: roundtypes.ResponseTentative},
					{UserID: sharedtypes.DiscordID("user_declined_1"), TagNumber: nil, Response: roundtypes.ResponseDecline},
					{UserID: sharedtypes.DiscordID("user_accepted_2"), TagNumber: nil, Response: roundtypes.ResponseAccept}, // Accepted but no tag
				}
				round.EventMessageID = "test_event_message_id_123"

				err := deps.DB.CreateRound(ctx, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create round in DB for test setup: %v", err)
				}

				return roundevents.DiscordReminderPayloadV1{
					GuildID:        "test-guild",
					RoundID:        round.ID,
					RoundTitle:     round.Title,
					StartTime:      round.StartTime,
					Location:       round.Location,
					ReminderType:   "24_HOUR_REMINDER",
					EventMessageID: round.EventMessageID,
				}
			},
			expectedError: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}

				discordPayload, ok := returnedResult.Success.(*roundevents.DiscordReminderPayloadV1)
				if !ok {
					t.Errorf("Expected result to be of type *roundevents.DiscordReminderPayloadV1, got %T", returnedResult.Success)
					return
				}

				// Verify RoundID
				if discordPayload.RoundID == sharedtypes.RoundID(uuid.Nil) {
					t.Errorf("Expected RoundID to be set, got empty")
				}

				// Verify participants to be notified (accepted and tentative only)
				expectedUserIDs := map[sharedtypes.DiscordID]bool{
					"user_accepted_1":  true,
					"user_tentative_1": true,
					"user_accepted_2":  true,
				}
				if len(discordPayload.UserIDs) != len(expectedUserIDs) {
					t.Errorf("Expected %d UserIDs, got %d", len(expectedUserIDs), len(discordPayload.UserIDs))
				}
				for _, userID := range discordPayload.UserIDs {
					if !expectedUserIDs[userID] {
						t.Errorf("Unexpected UserID '%s' in notification payload", userID)
					}
				}

				// Verify other fields are passed through
				if discordPayload.RoundTitle == "" {
					t.Errorf("Expected RoundTitle to be set")
				}
				if discordPayload.StartTime == nil {
					t.Errorf("Expected StartTime to be set")
				}
				if discordPayload.Location == nil {
					t.Errorf("Expected Location to be set")
				}
				if discordPayload.ReminderType != "24_HOUR_REMINDER" {
					t.Errorf("Expected ReminderType '24_HOUR_REMINDER', got '%s'", discordPayload.ReminderType)
				}
				if discordPayload.EventMessageID == "" {
					t.Errorf("Expected EventMessageID to be set")
				}
			},
		},
		{
			name: "Successful processing with no participants to notify",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) roundevents.DiscordReminderPayloadV1 {
				generator := testutils.NewTestDataGenerator()

				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("creator2"),
					Title:     "No Participants Round",
					State:     roundtypes.RoundStateUpcoming,
				})
				startTime := sharedtypes.StartTime(time.Now().Add(48 * time.Hour))
				round.StartTime = &startTime

				// Only declined participants or no participants
				round.Participants = []roundtypes.Participant{
					{UserID: sharedtypes.DiscordID("user_declined_2"), Response: roundtypes.ResponseDecline},
					{UserID: sharedtypes.DiscordID("user_declined_3"), Response: roundtypes.ResponseDecline},
				}
				round.EventMessageID = "test_event_message_id_456"

				err := deps.DB.CreateRound(ctx, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create round in DB for test setup: %v", err)
				}

				return roundevents.DiscordReminderPayloadV1{
					GuildID:        "test-guild",
					RoundID:        round.ID,
					RoundTitle:     round.Title,
					StartTime:      round.StartTime,
					Location:       round.Location,
					ReminderType:   "1_HOUR_REMINDER",
					EventMessageID: round.EventMessageID,
				}
			},
			expectedError: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}

				processedPayload, ok := returnedResult.Success.(*roundevents.DiscordReminderPayloadV1)
				if !ok {
					t.Errorf("Expected result to be of type *roundevents.DiscordReminderPayloadV1, got %T", returnedResult.Success)
					return
				}

				if processedPayload.RoundID == sharedtypes.RoundID(uuid.Nil) {
					t.Errorf("Expected RoundID to be set in processed payload, got empty")
				}

				// Verify no UserIDs in the payload since only declined participants exist
				if len(processedPayload.UserIDs) != 0 {
					t.Errorf("Expected 0 UserIDs for declined participants only, got %d", len(processedPayload.UserIDs))
				}
			},
		},
		{
			name: "Error: Round not found in DB",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) roundevents.DiscordReminderPayloadV1 {
				// No round created, so GetParticipants will fail
				return roundevents.DiscordReminderPayloadV1{
					GuildID:        "test-guild",
					RoundID:        nonexistentRoundID,
					RoundTitle:     "Non Existent Round",
					StartTime:      testutils.StartTimePtr(time.Now().Add(24 * time.Hour)),
					Location:       testutils.RoundLocationPtr("Nowhere"),
					ReminderType:   "24_HOUR_REMINDER",
					EventMessageID: "non_existent_event_message_id",
				}
			},
			// Fix: Service returns nil error and uses Failure payload
			expectedError:            false,
			expectedErrorMessagePart: "",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}

				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundErrorPayloadV1)
				if !ok {
					t.Fatalf("Expected returnedResult.Failure to be of type *roundevents.RoundErrorPayloadV1, got %T", returnedResult.Failure)
				}

				if failurePayload.RoundID != nonexistentRoundID {
					t.Errorf("Expected failure RoundID to be '%s', got '%s'", nonexistentRoundID, failurePayload.RoundID)
				}
				expectedDBErrorMessagePart := "not found"
				if !strings.Contains(failurePayload.Error, expectedDBErrorMessagePart) && !strings.Contains(failurePayload.Error, "round") {
					t.Errorf("Expected failure payload error message to contain '%s' or 'round', got '%s'", expectedDBErrorMessagePart, failurePayload.Error)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)

			payload := tt.setupTestEnv(deps.Ctx, deps)

			// Call the actual service method being tested
			result, err := deps.Service.ProcessRoundReminder(deps.Ctx, payload)

			// Check for expected error conditions
			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				} else if tt.expectedErrorMessagePart != "" && !strings.Contains(err.Error(), tt.expectedErrorMessagePart) {
					t.Errorf("Expected error message to contain '%s', but got: '%v'", tt.expectedErrorMessagePart, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			}

			// Validate the result using the test-specific validation function
			if tt.validateResult != nil {
				tt.validateResult(t, deps.Ctx, deps, result)
			}
		})
	}
}
