package roundhandler_integration_tests

import (
	"context"
	"testing"
	"time"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

// TestHandleCreateRoundRequest runs integration tests for the create round handler
func TestHandleCreateRoundRequest(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	user := generator.GenerateUsers(1)[0]
	userID := sharedtypes.DiscordID(user.UserID)

	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper)
		expectError bool
	}{
		{
			name: "Success - Create Valid Round",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				req := helper.CreateValidRequest(userID)
				helper.PublishRoundRequest(t, context.Background(), req)
				helper.ExpectSuccess(t, req, 3*time.Second)
			},
		},
		{
			name: "Success - Create Round with Minimal Information",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				req := helper.CreateMinimalRequest(userID)
				helper.PublishRoundRequest(t, context.Background(), req)
				helper.ExpectSuccess(t, req, 3*time.Second)
			},
		},
		{
			name: "Failure - Empty Description",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				req := helper.CreateInvalidRequest(userID, "empty_description")
				helper.PublishRoundRequest(t, context.Background(), req)
				helper.ExpectValidationFailure(t, req, 3*time.Second)
			},
		},
		{
			name: "Failure - Invalid Time Format",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				req := helper.CreateInvalidRequest(userID, "invalid_time")
				helper.PublishRoundRequest(t, context.Background(), req)
				helper.ExpectValidationFailure(t, req, 3*time.Second)
			},
		},
		{
			name: "Failure - Past Start Time",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				req := helper.CreateInvalidRequest(userID, "past_time")
				helper.PublishRoundRequest(t, context.Background(), req)
				helper.ExpectValidationFailure(t, req, 3*time.Second)
			},
		},
		{
			name: "Failure - Empty Title",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				req := helper.CreateInvalidRequest(userID, "empty_title")
				helper.PublishRoundRequest(t, context.Background(), req)
				helper.ExpectValidationFailure(t, req, 3*time.Second)
			},
		},
		{
			name: "Failure - Empty Location",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				req := helper.CreateInvalidRequest(userID, "empty_location")
				helper.PublishRoundRequest(t, context.Background(), req)
				helper.ExpectValidationFailure(t, req, 3*time.Second)
			},
		},
		{
			name:        "Failure - Invalid JSON Message",
			expectError: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				helper.PublishInvalidJSON(t, context.Background())
				helper.ExpectNoMessages(t, 1*time.Second)
			},
		},
		{
			name: "Failure - Missing Required Fields",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				req := helper.CreateInvalidRequest(userID, "missing_fields")
				helper.PublishRoundRequest(t, context.Background(), req)
				helper.ExpectValidationFailure(t, req, 3*time.Second)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestRoundHandler(t)
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)

			// Clear any existing captured messages
			helper.ClearMessages()

			// Run the test
			tc.setupAndRun(t, helper)
		})
	}
}
