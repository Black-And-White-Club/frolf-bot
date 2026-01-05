package roundhandler_integration_tests

import (
	"context"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

// createValidRequest creates a valid round request for testing
func createValidRequest(userID sharedtypes.DiscordID) testutils.RoundRequest {
	return testutils.RoundRequest{
		UserID:      userID,
		GuildID:     "test-guild",
		ChannelID:   "test-channel",
		Title:       "Weekly Frolf Championship",
		Description: "Join us for our weekly championship round!",
		Location:    "Central Park Course",
		StartTime:   "tomorrow at 3:00 PM",
		Timezone:    "UTC",
	}
}

// createMinimalRequest creates a minimal but valid round request
func createMinimalRequest(userID sharedtypes.DiscordID) testutils.RoundRequest {
	return testutils.RoundRequest{
		UserID:      userID,
		GuildID:     "test-guild",
		ChannelID:   "test-channel",
		Title:       "Quick Round",
		Description: "Quick round for today",
		Location:    "Local Course",
		StartTime:   "tomorrow at 3:00 PM",
		Timezone:    "UTC",
	}
}

// createInvalidRequest creates various types of invalid requests for testing
func createInvalidRequest(userID sharedtypes.DiscordID, invalidType string) testutils.RoundRequest {
	base := createValidRequest(userID)

	switch invalidType {
	case "empty_title":
		base.Title = ""
	case "empty_description":
		base.Description = ""
	case "empty_location":
		base.Location = ""
	case "invalid_time":
		base.StartTime = "not-a-valid-time"
	case "past_time":
		base.StartTime = "yesterday at 3:00 PM"
	case "missing_fields":
		return testutils.RoundRequest{
			UserID:      userID,
			Description: "Description only",
		}
	}

	return base
}

// expectSuccess validates successful round creation
func expectSuccess(t *testing.T, helper *testutils.RoundTestHelper, originalRequest testutils.RoundRequest, timeout time.Duration) {
	t.Helper()

	if !helper.WaitForRoundEntityCreated(1, timeout) {
		t.Fatalf("Expected round.entity.created message within %v", timeout)
	}

	msgs := helper.GetRoundEntityCreatedMessages()
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 success message, got %d", len(msgs))
	}

	result := helper.ValidateRoundEntityCreated(t, msgs[0], originalRequest.UserID)

	// Validate specific transformation for create round
	if result.Round.Title != roundtypes.Title(originalRequest.Title) {
		t.Errorf("Title mismatch: expected %s, got %s", originalRequest.Title, result.Round.Title)
	}

	if result.Round.Location == nil || *result.Round.Location != roundtypes.Location(originalRequest.Location) {
		t.Errorf("Location mismatch: expected %s, got %v", originalRequest.Location, result.Round.Location)
	}

	if len(result.Round.Participants) != 0 {
		t.Errorf("Expected empty participants, got %d", len(result.Round.Participants))
	}
}

// expectValidationFailure validates validation failure scenarios
func expectValidationFailure(t *testing.T, helper *testutils.RoundTestHelper, originalRequest testutils.RoundRequest, timeout time.Duration) {
	t.Helper()

	if !helper.WaitForRoundValidationFailed(1, timeout) {
		t.Fatalf("Expected validation failure message within %v", timeout)
	}

	msgs := helper.GetRoundValidationFailedMessages()
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 failure message, got %d", len(msgs))
	}

	helper.ValidateRoundValidationFailed(t, msgs[0], originalRequest.UserID)

	// Ensure no success message was published
	successMsgs := helper.GetRoundEntityCreatedMessages()
	if len(successMsgs) > 0 {
		t.Errorf("Expected no success messages, got %d", len(successMsgs))
	}
}

// TestHandleCreateRoundRequest runs integration tests for the create round handler
func TestHandleCreateRoundRequest(t *testing.T) {
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper)
		expectError bool
	}{
		{
			name: "Success - Create Valid Round",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				PrepareSubTest(deps) // Clear message capture for test isolation
				data := NewTestData()
				req := createValidRequest(data.UserID)
				helper.PublishRoundRequest(t, context.Background(), req)
				expectSuccess(t, helper, req, 500*time.Millisecond)
			},
		},
		{
			name: "Success - Create Round with Minimal Information",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				PrepareSubTest(deps) // Clear message capture for test isolation
				data := NewTestData()
				req := createMinimalRequest(data.UserID)
				helper.PublishRoundRequest(t, context.Background(), req)
				expectSuccess(t, helper, req, 500*time.Millisecond)
			},
		},
		{
			name: "Failure - Empty Description",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				PrepareSubTest(deps) // Clear message capture for test isolation
				data := NewTestData()
				req := createInvalidRequest(data.UserID, "empty_description")
				helper.PublishRoundRequest(t, context.Background(), req)
				expectValidationFailure(t, helper, req, 500*time.Millisecond)
			},
		},
		{
			name: "Failure - Invalid Time Format",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				PrepareSubTest(deps) // Clear message capture for test isolation
				data := NewTestData()
				req := createInvalidRequest(data.UserID, "invalid_time")
				helper.PublishRoundRequest(t, context.Background(), req)
				expectValidationFailure(t, helper, req, 500*time.Millisecond)
			},
		},
		{
			name: "Failure - Past Start Time",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				PrepareSubTest(deps) // Clear message capture for test isolation
				data := NewTestData()
				req := createInvalidRequest(data.UserID, "past_time")
				helper.PublishRoundRequest(t, context.Background(), req)
				expectValidationFailure(t, helper, req, 500*time.Millisecond)
			},
		},
		{
			name: "Failure - Empty Title",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				PrepareSubTest(deps) // Clear message capture for test isolation
				data := NewTestData()
				req := createInvalidRequest(data.UserID, "empty_title")
				helper.PublishRoundRequest(t, context.Background(), req)
				expectValidationFailure(t, helper, req, 500*time.Millisecond)
			},
		},
		{
			name: "Failure - Empty Location",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				PrepareSubTest(deps) // Clear message capture for test isolation
				data := NewTestData()
				req := createInvalidRequest(data.UserID, "empty_location")
				helper.PublishRoundRequest(t, context.Background(), req)
				expectValidationFailure(t, helper, req, 500*time.Millisecond)
			},
		},
		{
			name:        "Failure - Invalid JSON Message",
			expectError: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				PrepareSubTest(deps) // Clear message capture for test isolation
				// Count messages before
				createdMsgsBefore := len(helper.GetRoundEntityCreatedMessages())
				failedMsgsBefore := len(helper.GetRoundValidationFailedMessages())

				helper.PublishInvalidJSON(t, context.Background(), roundevents.RoundCreationRequestedV1)

				// Wait a bit to ensure no messages are published
				time.Sleep(300 * time.Millisecond)

				createdMsgsAfter := len(helper.GetRoundEntityCreatedMessages())
				failedMsgsAfter := len(helper.GetRoundValidationFailedMessages())

				newCreatedMsgs := createdMsgsAfter - createdMsgsBefore
				newFailedMsgs := failedMsgsAfter - failedMsgsBefore

				if newCreatedMsgs > 0 {
					t.Errorf("Expected no NEW round.entity.created messages, got %d new", newCreatedMsgs)
				}

				if newFailedMsgs > 0 {
					t.Errorf("Expected no NEW validation failure messages, got %d new", newFailedMsgs)
				}
			},
		},
		{
			name: "Failure - Missing Required Fields",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				PrepareSubTest(deps) // Clear message capture for test isolation
				data := NewTestData()
				req := createInvalidRequest(data.UserID, "missing_fields")
				helper.PublishRoundRequest(t, context.Background(), req)
				expectValidationFailure(t, helper, req, 500*time.Millisecond)
			},
		},
	}

	// Run all subtests with SHARED setup - no need to clear messages between tests!
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Clear message capture before each subtest
			deps.MessageCapture.Clear()
			// Create helper for each subtest
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)

			// Run the test - no cleanup needed!
			tc.setupAndRun(t, helper)
		})
	}
}
