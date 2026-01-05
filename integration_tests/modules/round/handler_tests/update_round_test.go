package roundhandler_integration_tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

func TestHandleRoundUpdateRequest(t *testing.T) {
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)

	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Valid Title Update",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create a valid round ID for reference
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{})

				// Create valid round update request payload with title update
				newTitle := roundtypes.Title("Updated Round Title")
				payload := createRoundUpdateRequestPayload(roundID, data.UserID, &newTitle, nil, nil, nil)

				result := publishAndExpectRoundUpdateValidated(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.Round.ID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.Round.ID)
				}
				if result.Round.Title != newTitle {
					t.Errorf("Expected Title '%s', got '%s'", newTitle, result.Round.Title)
				}
				if result.Round.CreatedBy != data.UserID {
					t.Errorf("Expected CreatedBy %s, got %s", data.UserID, result.Round.CreatedBy)
				}
			},
		},
		{
			name: "Success - Valid Description Update",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{})

				// Create payload with description update
				newDesc := roundtypes.Description("Updated description for the round")
				payload := createRoundUpdateRequestPayload(roundID, data.UserID, nil, &newDesc, nil, nil)

				result := publishAndExpectRoundUpdateValidated(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.Round.ID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.Round.ID)
				}
				if result.Round.Description == nil || *result.Round.Description != newDesc {
					t.Errorf("Expected Description '%s', got %v", newDesc, result.Round.Description)
				}
			},
		},
		{
			name: "Success - Valid Location Update",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{})

				// Create payload with location update
				newLocation := roundtypes.Location("Updated Course Location")
				payload := createRoundUpdateRequestPayload(roundID, data.UserID, nil, nil, &newLocation, nil)

				result := publishAndExpectRoundUpdateValidated(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.Round.Location == nil || *result.Round.Location != newLocation {
					t.Errorf("Expected Location '%s', got %v", newLocation, result.Round.Location)
				}
			},
		},
		{
			name: "Success - Multiple Fields Update (Without Start Time)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{})

				// Create payload with multiple field updates (excluding start time to avoid rescheduling)
				newTitle := roundtypes.Title("Multi-Update Round")
				newDesc := roundtypes.Description("Updated with multiple fields")
				newLocation := roundtypes.Location("New Multi-Field Location")

				timezone := roundtypes.Timezone("America/Chicago")
				payload := roundevents.UpdateRoundRequestedPayloadV1{
					GuildID:     "test-guild",
					RoundID:     roundID,
					UserID:      data.UserID,
					ChannelID:   "test_channel_123",
					MessageID:   "test_message_456",
					Timezone:    &timezone,
					Title:       &newTitle,
					Description: &newDesc,
					Location:    &newLocation,
				}

				publishRoundUpdateRequestMessage(t, deps, &payload)

				if !waitForRoundUpdateValidatedFromHandler(deps.MessageCapture, 1) {
					t.Fatalf("Expected round update validated message from %s", roundevents.RoundUpdatedV1)
				}

				msgs := getRoundUpdateValidatedFromHandlerMessages(deps.MessageCapture)
				result := validateRoundUpdateValidatedFromHandler(t, msgs[0])

				// Validate all fields are set
				if result.Round.Title != newTitle {
					t.Errorf("Expected Title '%s', got '%s'", newTitle, result.Round.Title)
				}
				if result.Round.Description == nil || *result.Round.Description != newDesc {
					t.Errorf("Expected Description '%s', got %v", newDesc, result.Round.Description)
				}
				if result.Round.Location == nil || *result.Round.Location != newLocation {
					t.Errorf("Expected Location '%s', got %v", newLocation, result.Round.Location)
				}
			},
		},
		{
			name: "Failure - Zero Round ID",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Use zero UUID for round ID
				zeroRoundID := sharedtypes.RoundID(uuid.Nil)
				newTitle := roundtypes.Title("Title for Zero Round")
				payload := createRoundUpdateRequestPayload(zeroRoundID, data.UserID, &newTitle, nil, nil, nil)

				result := publishAndExpectRoundUpdateError(t, deps, deps.MessageCapture, payload)

				// Validate the error
				if result.Error == "" {
					t.Error("Expected Error message to be populated")
				}
				if !contains(result.Error, "round ID cannot be zero") {
					t.Errorf("Expected error to contain 'round ID cannot be zero', got: %s", result.Error)
				}
			},
		},
		{
			name: "Failure - No Fields to Update",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{})

				// Create payload with no fields to update (all nil/empty)
				payload := createRoundUpdateRequestPayload(roundID, data.UserID, nil, nil, nil, nil)

				result := publishAndExpectRoundUpdateError(t, deps, deps.MessageCapture, payload)

				// Validate the error
				if !contains(result.Error, "at least one field to update must be provided") {
					t.Errorf("Expected error to contain 'at least one field to update must be provided', got: %s", result.Error)
				}
			},
		},
		{
			name: "Failure - Empty Title with No Other Fields",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{})

				// Create payload with empty title and no other fields - using the CORRECT roundID
				emptyTitle := roundtypes.Title("")
				timezone := roundtypes.Timezone("America/Chicago")
				payload := roundevents.UpdateRoundRequestedPayloadV1{
					GuildID:   "test-guild",
					RoundID:   roundID, // Use the actual roundID, not zero UUID
					UserID:    data.UserID,
					ChannelID: "test_channel_123",
					MessageID: "test_message_456",
					Timezone:  &timezone,
					Title:     &emptyTitle,
				}

				result := publishAndExpectRoundUpdateError(t, deps, deps.MessageCapture, payload)

				// Validate the error
				if !contains(result.Error, "no valid fields to update") {
					t.Errorf("Expected error to contain 'no valid fields to update', got: %s", result.Error)
				}
			},
		},
		{
			name: "Failure - Multiple Validation Errors",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Use zero UUID and no fields to update
				zeroRoundID := sharedtypes.RoundID(uuid.Nil)
				payload := createRoundUpdateRequestPayload(zeroRoundID, data.UserID, nil, nil, nil, nil)

				result := publishAndExpectRoundUpdateError(t, deps, deps.MessageCapture, payload)

				// Validate that both errors are present
				if !contains(result.Error, "round ID cannot be zero") {
					t.Errorf("Expected error to contain 'round ID cannot be zero', got: %s", result.Error)
				}
				if !contains(result.Error, "at least one field to update must be provided") {
					t.Errorf("Expected error to contain 'at least one field to update must be provided', got: %s", result.Error)
				}
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				publishInvalidJSONAndExpectNoRoundUpdateMessages(t, deps, deps.MessageCapture)
			},
		},
	}

	// Run all subtests with SHARED setup
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)
			tc.setupAndRun(t, helper, &deps)
		})
	}
}

// Helper functions for creating payloads - UPDATED TO MATCH HANDLER
func createRoundUpdateRequestPayload(
	roundID sharedtypes.RoundID,
	userID sharedtypes.DiscordID,
	title *roundtypes.Title,
	description *roundtypes.Description,
	location *roundtypes.Location,
	startTime *sharedtypes.StartTime,
) roundevents.UpdateRoundRequestedPayloadV1 {
	timezone := roundtypes.Timezone("America/Chicago")
	payload := roundevents.UpdateRoundRequestedPayloadV1{
		GuildID:   "test-guild",
		RoundID:   roundID,
		UserID:    userID,
		ChannelID: "test_channel_123", // Required Discord field
		MessageID: "test_message_456", // Required Discord field
		Timezone:  &timezone,          // Required for time parsing
	}

	// Set optional fields if provided
	if title != nil {
		payload.Title = title
	}
	if description != nil {
		payload.Description = description
	}
	if location != nil {
		payload.Location = location
	}
	if startTime != nil {
		// Create a definitive future time in Chicago timezone
		chicagoLocation, _ := time.LoadLocation("America/Chicago")
		now := time.Now().In(chicagoLocation)
		futureTime := now.Add(48 * time.Hour)
		timeStr := futureTime.Format("2006-01-02 15:04")
		payload.StartTime = &timeStr
	}

	return payload
}

// Publishing functions - UPDATED TO MATCH HANDLER
func publishRoundUpdateRequestMessage(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.UpdateRoundRequestedPayloadV1) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	// Publish to the correct topic that the handler subscribes to
	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundUpdateRequestedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// Test expectation functions - UPDATED SIGNATURES
func publishAndExpectRoundUpdateValidated(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.UpdateRoundRequestedPayloadV1) *roundevents.RoundEntityUpdatedPayloadV1 {
	publishRoundUpdateRequestMessage(t, deps, &payload)

	if !waitForRoundUpdateValidatedFromHandler(capture, 1) {
		t.Fatalf("Expected round update validated message from %s", roundevents.RoundUpdatedV1)
	}

	msgs := getRoundUpdateValidatedFromHandlerMessages(capture)
	result := validateRoundUpdateValidatedFromHandler(t, msgs[0])

	return result
}

func publishAndExpectRoundUpdateError(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.UpdateRoundRequestedPayloadV1) *roundevents.RoundUpdateErrorPayloadV1 {
	// Wait longer to let any in-flight async messages from previous tests complete
	time.Sleep(50 * time.Millisecond)

	// Clear capture to ensure we start with a clean slate
	capture.Clear()

	// Publish our test message
	publishRoundUpdateRequestMessage(t, deps, &payload)

	// Wait for at least one error message to arrive
	if !waitForRoundUpdateErrorFromHandler(capture, 1) {
		t.Fatalf("Timed out waiting for round update error message from %s",
			roundevents.RoundUpdateErrorV1)
	}

	// Then filter by round ID with deadline-based polling
	// Note: RoundUpdateRequest may be nil for early validation failures
	deadline := time.Now().Add(defaultTimeout)
	var foundMsg *message.Message

	for time.Now().Before(deadline) {
		msgs := getRoundUpdateErrorFromHandlerMessages(capture)
		// Look through all messages
		for _, msg := range msgs {
			parsed, err := testutils.ParsePayload[roundevents.RoundUpdateErrorPayloadV1](msg)
			if err == nil {
				// Match by RoundUpdateRequest.RoundID if available (later in processing)
				if parsed.RoundUpdateRequest != nil && parsed.RoundUpdateRequest.RoundID == payload.RoundID {
					foundMsg = msg
					break
				}
				// For early validation failures, RoundUpdateRequest is nil
				// Take the FIRST nil message (should be from our publish since we cleared and waited)
				if parsed.RoundUpdateRequest == nil {
					foundMsg = msg
					break
				}
			}
		}
		if foundMsg != nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if foundMsg == nil {
		t.Fatalf("Expected round update error message from %s for round %s", roundevents.RoundUpdateErrorV1, payload.RoundID)
	}

	return validateRoundUpdateErrorFromHandler(t, foundMsg)
}

func publishInvalidJSONAndExpectNoRoundUpdateMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	t.Helper()

	// Create invalid JSON message
	invalidMsg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
	invalidMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	// Publish to the correct topic
	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundUpdateRequestedV1, invalidMsg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Wait a bit to ensure no messages are published
	time.Sleep(500 * time.Millisecond)

	validatedMsgs := getRoundUpdateValidatedFromHandlerMessages(capture)
	errorMsgs := getRoundUpdateErrorFromHandlerMessages(capture)

	if len(validatedMsgs) > 0 {
		t.Errorf("Expected no validated messages for invalid JSON, got %d", len(validatedMsgs))
	}

	if len(errorMsgs) > 0 {
		t.Errorf("Expected no error messages for invalid JSON, got %d", len(errorMsgs))
	}
}

// Wait functions - UNIQUE TO ROUND UPDATE REQUEST TESTS
func waitForRoundUpdateValidatedFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundUpdatedV1, count, defaultTimeout)
}

func waitForRoundUpdateErrorFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundUpdateErrorV1, count, defaultTimeout)
}

// Message retrieval functions - UNIQUE TO ROUND UPDATE REQUEST TESTS
func getRoundUpdateValidatedFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundUpdatedV1)
}

func getRoundUpdateErrorFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundUpdateErrorV1)
}

// Validation functions - UNIQUE TO ROUND UPDATE REQUEST TESTS
func validateRoundUpdateValidatedFromHandler(t *testing.T, msg *message.Message) *roundevents.RoundEntityUpdatedPayloadV1 {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.RoundEntityUpdatedPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse round update validated message: %v", err)
	}

	// Validate that required fields are set
	if result.Round.ID == sharedtypes.RoundID(uuid.Nil) {
		t.Error("Expected Round.ID to be set")
	}

	if result.Round.CreatedBy == "" {
		t.Error("Expected Round.CreatedBy to be set")
	}

	return result
}

func validateRoundUpdateErrorFromHandler(t *testing.T, msg *message.Message) *roundevents.RoundUpdateErrorPayloadV1 {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.RoundUpdateErrorPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse round update error message: %v", err)
	}

	if result.Error == "" {
		t.Error("Expected error message to be populated")
	}

	// Note: RoundUpdateRequest may be nil for early validation errors
	if result.RoundUpdateRequest == nil {
		t.Logf("RoundUpdateRequest is nil (expected for early validation errors)")
	}

	// Log what we got for debugging
	t.Logf("Round update request failed with error: %s", result.Error)

	return result
}

// Helper utility functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findInString(s, substr))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
