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
func createValidRoundDeleteRequestPayload(roundID sharedtypes.RoundID, requestingUserID sharedtypes.DiscordID) roundevents.RoundDeleteRequestPayloadV1 {
	return roundevents.RoundDeleteRequestPayloadV1{
		GuildID:              "test-guild",
		RoundID:              roundID,
		RequestingUserUserID: requestingUserID,
	}
}

// createValidRoundDeleteAuthorizedPayload creates a valid RoundDeleteAuthorizedPayload for testing
func createValidRoundDeleteAuthorizedPayload(roundID sharedtypes.RoundID) roundevents.RoundDeleteAuthorizedPayloadV1 {
	return roundevents.RoundDeleteAuthorizedPayloadV1{
		GuildID: "test-guild",
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
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	const testTimeout = 500 * time.Millisecond

	testCases := []struct {
		name             string
		setupAndRun      func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
		expectAuthorized bool
		expectError      bool
		expectNoMessages bool
	}{
		{
			name:             "Success - Valid Delete Request",
			expectAuthorized: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := createExistingRoundForDeletion(t, helper, data.UserID, deps.DB)
				payload := createValidRoundDeleteRequestPayload(roundID, data.UserID)
				helper.PublishRoundDeleteRequest(t, context.Background(), payload)
			},
		},
		{
			name:             "Failure - Nil Round ID",
			expectNoMessages: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				payload := createValidRoundDeleteRequestPayload(sharedtypes.RoundID(uuid.Nil), data.UserID)
				helper.PublishRoundDeleteRequest(t, context.Background(), payload)
			},
		},
		{
			name:        "Failure - Non-Existent Round ID",
			expectError: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				payload := createValidRoundDeleteRequestPayload(nonExistentRoundID, data.UserID)
				helper.PublishRoundDeleteRequest(t, context.Background(), payload)
			},
		},
		{
			name:        "Failure - Unauthorized User",
			expectError: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data1 := NewTestData()
				data2 := NewTestData()
				roundID := createExistingRoundForDeletion(t, helper, data1.UserID, deps.DB)
				payload := createValidRoundDeleteRequestPayload(roundID, data2.UserID)
				helper.PublishRoundDeleteRequest(t, context.Background(), payload)
			},
		},
		{
			name:             "Failure - Invalid JSON Message",
			expectNoMessages: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				helper.PublishInvalidJSON(t, context.Background(), roundevents.RoundDeleteRequestedV1)
			},
		},
	}

	// Run all subtests with SHARED setup - no need to clear messages between tests!
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {			// Clear message capture before each subtest
			deps.MessageCapture.Clear()			// Create helper for each subtest
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)

			// Run the test - no cleanup needed!
			tc.setupAndRun(t, helper, &deps)

			if tc.expectAuthorized {
				if !helper.WaitForRoundDeleteAuthorized(1, testTimeout) {
					t.Error("Timed out waiting for authorized message")
				}
			} else if tc.expectError {
				if !helper.WaitForRoundDeleteError(1, testTimeout) {
					t.Error("Timed out waiting for error message")
				}
			} else if tc.expectNoMessages {
				time.Sleep(testTimeout)
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
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	const testTimeout = 500 * time.Millisecond

	testCases := []struct {
		name             string
		setupAndRun      func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
		expectDeleted    bool
		expectError      bool
		expectNoMessages bool
	}{
		{
			name:          "Success - Delete Existing Round",
			expectDeleted: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := createExistingRoundForDeletion(t, helper, data.UserID, deps.DB)
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
				helper.PublishInvalidJSON(t, context.Background(), roundevents.RoundDeleteAuthorizedV1)
			},
		},
	}

	// Run all subtests with SHARED setup
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Clear messages before each subtest to ensure isolation
			deps.MessageCapture.Clear()

			// Create helper for each subtest
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)

			// Run the test
			tc.setupAndRun(t, helper, &deps)

			if tc.expectDeleted {
				if !helper.WaitForRoundDeleted(1, testTimeout) {
					t.Error("Timed out waiting for deleted message")
				}
			} else if tc.expectError {
				if !helper.WaitForRoundDeleteError(1, testTimeout) {
					t.Error("Timed out waiting for error message")
				}
			} else if tc.expectNoMessages {
				time.Sleep(testTimeout)
			}

			deletedMsgs := helper.GetRoundDeletedMessages()
			errorMsgs := helper.GetRoundDeleteErrorMessages()

			if tc.expectDeleted {
				if len(deletedMsgs) == 0 {
					t.Error("Expected deleted message, got none")
				}
				if len(errorMsgs) > 0 {
					t.Errorf("Expected no error messages, got %d", len(errorMsgs))
					// Log the actual error to help debug
					if len(errorMsgs) > 0 {
						result, err := testutils.ParsePayload[roundevents.RoundDeleteErrorPayloadV1](errorMsgs[0])
						if err == nil {
							t.Logf("Error payload: %+v", result)
						}
					}
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
