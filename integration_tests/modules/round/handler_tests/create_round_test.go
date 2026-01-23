package roundhandler_integration_tests

import (
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

// createCreateRoundPayload converts a RoundRequest to a CreateRoundRequestedPayloadV1
func createCreateRoundPayload(req testutils.RoundRequest) roundevents.CreateRoundRequestedPayloadV1 {
	location := roundtypes.Location(req.Location)
	return roundevents.CreateRoundRequestedPayloadV1{
		GuildID:     req.GuildID,
		UserID:      req.UserID,
		ChannelID:   req.ChannelID,
		Title:       roundtypes.Title(req.Title),
		Description: roundtypes.Description(req.Description),
		Location:    location,
		StartTime:   req.StartTime,
		Timezone:    roundtypes.Timezone(req.Timezone),
	}
}

// TestHandleCreateRoundRequest runs integration tests for the create round handler
func TestHandleCreateRoundRequest(t *testing.T) {
	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		timeout                time.Duration
	}{
		{
			name: "Success - Create Valid Round",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				req := createValidRequest(data.UserID)
				payload := createCreateRoundPayload(req)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundCreationRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundEntityCreatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				expectedTopic := roundevents.RoundEntityCreatedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				receivedMsg := msgs[0]
				var payload roundevents.RoundEntityCreatedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(receivedMsg, &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}

				if payload.Round.Title != roundtypes.Title("Weekly Frolf Championship") {
					t.Errorf("Title mismatch: expected %s, got %s", "Weekly Frolf Championship", payload.Round.Title)
				}

				if payload.Round.Location != roundtypes.Location("Central Park Course") {
					t.Errorf("Location mismatch: expected %s, got %v", "Central Park Course", payload.Round.Location)
				}

				if len(payload.Round.Participants) != 0 {
					t.Errorf("Expected empty participants, got %d", len(payload.Round.Participants))
				}

				if receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != triggerMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Success - Create Round with Minimal Information",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				req := createMinimalRequest(data.UserID)
				payload := createCreateRoundPayload(req)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundCreationRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundEntityCreatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				expectedTopic := roundevents.RoundEntityCreatedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}

				receivedMsg := msgs[0]
				var payload roundevents.RoundEntityCreatedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(receivedMsg, &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}

				if payload.Round.Title != roundtypes.Title("Quick Round") {
					t.Errorf("Title mismatch: expected %s, got %s", "Quick Round", payload.Round.Title)
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Failure - Empty Description",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				req := createInvalidRequest(data.UserID, "empty_description")
				payload := createCreateRoundPayload(req)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundCreationRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundValidationFailedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				expectedTopic := roundevents.RoundValidationFailedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}

				// Ensure no success message was published
				successMsgs := receivedMsgs[roundevents.RoundEntityCreatedV1]
				if len(successMsgs) > 0 {
					t.Errorf("Expected no success messages, got %d", len(successMsgs))
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Failure - Invalid Time Format",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				req := createInvalidRequest(data.UserID, "invalid_time")
				payload := createCreateRoundPayload(req)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundCreationRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundValidationFailedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				expectedTopic := roundevents.RoundValidationFailedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected validation failed message on topic %q", expectedTopic)
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Failure - Past Start Time",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				req := createInvalidRequest(data.UserID, "past_time")
				payload := createCreateRoundPayload(req)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundCreationRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundValidationFailedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				expectedTopic := roundevents.RoundValidationFailedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected validation failed message on topic %q", expectedTopic)
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Failure - Empty Title",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				req := createInvalidRequest(data.UserID, "empty_title")
				payload := createCreateRoundPayload(req)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundCreationRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundValidationFailedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				expectedTopic := roundevents.RoundValidationFailedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected validation failed message on topic %q", expectedTopic)
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Failure - Empty Location",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				req := createInvalidRequest(data.UserID, "empty_location")
				payload := createCreateRoundPayload(req)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundCreationRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundValidationFailedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				expectedTopic := roundevents.RoundValidationFailedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected validation failed message on topic %q", expectedTopic)
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Failure - Invalid JSON Message",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				// Publish invalid JSON
				invalidJSON := []byte(`invalid json`)
				msg := message.NewMessage(uuid.New().String(), invalidJSON)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundCreationRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{}, // No valid messages expected for invalid JSON
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				// Verify no success or error messages were created
				createdMsgs := receivedMsgs[roundevents.RoundEntityCreatedV1]
				failedMsgs := receivedMsgs[roundevents.RoundValidationFailedV1]
				if len(createdMsgs) > 0 || len(failedMsgs) > 0 {
					t.Errorf("Expected no messages for invalid JSON, but got created=%d, failed=%d", len(createdMsgs), len(failedMsgs))
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Failure - Missing Required Fields",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				req := createInvalidRequest(data.UserID, "missing_fields")
				payload := createCreateRoundPayload(req)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundCreationRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundValidationFailedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				expectedTopic := roundevents.RoundValidationFailedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected validation failed message on topic %q", expectedTopic)
				}
			},
			timeout: 5 * time.Second,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestRoundHandler(t)

			genericCase := testutils.TestCase{
				Name: tc.name,
				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
					return tc.setupFn(t, deps, env)
				},
				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
					return tc.publishMsgFn(t, deps, env)
				},
				ExpectedTopics: tc.expectedOutgoingTopics,
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
					tc.validateFn(t, deps, env, triggerMsg, receivedMsgs, initialState)
				},
				ExpectError:    false,
				MessageTimeout: tc.timeout,
			}
			testutils.RunTest(t, genericCase, deps.TestEnvironment)
		})
	}
}
