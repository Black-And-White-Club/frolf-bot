package roundhandler_integration_tests

import (
	"context"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// createValidRoundDeleteRequestPayload creates a valid RoundDeleteRequestPayload for testing
func createValidRoundDeleteRequestPayload(roundID sharedtypes.RoundID, requestingUserID sharedtypes.DiscordID) roundevents.RoundDeleteRequestPayload {
	return roundevents.RoundDeleteRequestPayload{
		RoundID:              roundID,
		RequestingUserUserID: requestingUserID,
	}
}

// createValidRoundDeleteAuthorizedPayload creates a valid RoundDeleteAuthorizedPayload for testing
func createValidRoundDeleteAuthorizedPayload(roundID sharedtypes.RoundID) roundevents.RoundDeleteAuthorizedPayload {
	return roundevents.RoundDeleteAuthorizedPayload{
		RoundID: roundID,
	}
}

// TestHandleRoundDeleteRequest tests the handler logic for RoundDeleteRequest
// createExistingRoundForDeletion creates and stores a round that can be deleted
func createExistingRoundForDeletion(t *testing.T, helper *testutils.RoundTestHelper, userID sharedtypes.DiscordID, db bun.IDB) sharedtypes.RoundID {
	t.Helper()

	// Use the passed DB instance instead of creating new deps
	return helper.CreateRoundInDB(t, db, userID)
}

// TestHandleRoundDeleteRequest tests the handler logic for RoundDeleteRequest
func TestHandleRoundDeleteRequest(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	user := generator.GenerateUsers(1)[0]
	userID := sharedtypes.DiscordID(user.UserID)
	const testTimeout = 5 * time.Second

	testCases := []struct {
		name             string
		setupAndRun      func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) // ✅ Pass deps
		expectAuthorized bool
		expectError      bool
		expectNoMessages bool
	}{
		{
			name:             "Success - Valid Delete Request",
			expectAuthorized: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) { // ✅ Accept deps
				roundID := createExistingRoundForDeletion(t, helper, userID, deps.DB) // ✅ Pass DB
				payload := createValidRoundDeleteRequestPayload(roundID, userID)
				helper.PublishRoundDeleteRequest(t, context.Background(), payload)
			},
		},
		{
			name:             "Failure - Nil Round ID",
			expectNoMessages: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				payload := createValidRoundDeleteRequestPayload(sharedtypes.RoundID(uuid.Nil), userID)
				helper.PublishRoundDeleteRequest(t, context.Background(), payload)
			},
		},
		{
			name:        "Failure - Non-Existent Round ID",
			expectError: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				payload := createValidRoundDeleteRequestPayload(nonExistentRoundID, userID)
				helper.PublishRoundDeleteRequest(t, context.Background(), payload)
			},
		},
		{
			name:        "Failure - Unauthorized User",
			expectError: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := createExistingRoundForDeletion(t, helper, userID, deps.DB) // ✅ Pass DB
				differentUser := generator.GenerateUsers(1)[0]
				differentUserID := sharedtypes.DiscordID(differentUser.UserID)
				payload := createValidRoundDeleteRequestPayload(roundID, differentUserID)
				helper.PublishRoundDeleteRequest(t, context.Background(), payload)
			},
		},
		{
			name:             "Failure - Invalid JSON Message",
			expectNoMessages: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				helper.PublishInvalidJSON(t, context.Background(), roundevents.RoundDeleteRequest)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestRoundHandler(t) // ✅ Create deps once per test case
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)
			helper.ClearMessages()

			tc.setupAndRun(t, helper, &deps) // ✅ Pass deps to the test function

			// ... rest of test validation logic stays the same
			if tc.expectAuthorized {
				if !helper.WaitForRoundDeleteAuthorized(1, testTimeout) {
					t.Error("Timed out waiting for authorized message")
				}
			} else if tc.expectError {
				if !helper.WaitForRoundDeleteError(1, testTimeout) {
					t.Error("Timed out waiting for error message")
				}
			} else if tc.expectNoMessages {
				time.Sleep(500 * time.Millisecond)
			}

			authorizedMsgs := helper.GetRoundDeleteAuthorizedMessages()
			errorMsgs := helper.GetRoundDeleteErrorMessages()

			if tc.expectAuthorized {
				if len(authorizedMsgs) == 0 {
					t.Error("Expected authorized message, got none")
				}
				if len(errorMsgs) > 0 {
					t.Errorf("Expected no error messages, got %d", len(errorMsgs))
				}
			} else if tc.expectError {
				if len(errorMsgs) == 0 {
					t.Error("Expected error message, got none")
				}
				if len(authorizedMsgs) > 0 {
					t.Errorf("Expected no authorized messages, got %d", len(authorizedMsgs))
				}
			} else if tc.expectNoMessages {
				if len(authorizedMsgs) > 0 {
					t.Errorf("Expected no authorized messages, got %d", len(authorizedMsgs))
				}
				if len(errorMsgs) > 0 {
					t.Errorf("Expected no error messages, got %d", len(errorMsgs))
				}
			}
		})
	}
}

// TestHandleRoundDeleteAuthorized tests the authorized delete handler
func TestHandleRoundDeleteAuthorized(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	user := generator.GenerateUsers(1)[0]
	userID := sharedtypes.DiscordID(user.UserID)
	const testTimeout = 5 * time.Second // Define a common timeout for waiting

	testCases := []struct {
		name             string
		setupAndRun      func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) // ✅ Pass deps
		expectDeleted    bool
		expectError      bool
		expectNoMessages bool
	}{
		{
			name:          "Success - Delete Existing Round",
			expectDeleted: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) { // ✅ Accept deps
				roundID := createExistingRoundForDeletion(t, helper, userID, deps.DB) // ✅ Pass DB
				payload := createValidRoundDeleteAuthorizedPayload(roundID)
				helper.PublishRoundDeleteAuthorized(t, context.Background(), payload)
			},
		},
		{
			name:        "Failure - Non-Existent Round ID",
			expectError: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				payload := createValidRoundDeleteAuthorizedPayload(nonExistentRoundID)
				helper.PublishRoundDeleteAuthorized(t, context.Background(), payload)
			},
		},
		{
			name:             "Failure - Invalid JSON Message",
			expectNoMessages: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				helper.PublishInvalidJSON(t, context.Background(), roundevents.RoundDeleteAuthorized)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestRoundHandler(t) // ✅ Create deps once per test case
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)
			helper.ClearMessages()

			tc.setupAndRun(t, helper, &deps) // ✅ Pass deps to the test function

			if tc.expectDeleted {
				if !helper.WaitForRoundDeleted(1, testTimeout) {
					t.Error("Timed out waiting for deleted message")
				}
			} else if tc.expectError {
				if !helper.WaitForRoundDeleteError(1, testTimeout) {
					t.Error("Timed out waiting for error message")
				}
			} else if tc.expectNoMessages {
				time.Sleep(500 * time.Millisecond)
			}

			deletedMsgs := helper.GetRoundDeletedMessages()
			errorMsgs := helper.GetRoundDeleteErrorMessages()

			if tc.expectDeleted {
				if len(deletedMsgs) == 0 {
					t.Error("Expected deleted message, got none")
				}
				if len(errorMsgs) > 0 {
					t.Errorf("Expected no error messages, got %d", len(errorMsgs))
				}
			} else if tc.expectError {
				if len(errorMsgs) == 0 {
					t.Error("Expected error message, got none")
				}
				if len(deletedMsgs) > 0 {
					t.Errorf("Expected no deleted messages, got %d", len(deletedMsgs))
				}
			} else if tc.expectNoMessages {
				if len(deletedMsgs) > 0 {
					t.Errorf("Expected no deleted messages, got %d", len(deletedMsgs))
				}
				if len(errorMsgs) > 0 {
					t.Errorf("Expected no error messages, got %d", len(errorMsgs))
				}
			}
		})
	}
}
