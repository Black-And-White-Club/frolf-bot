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

func TestHandleRoundUpdateValidated(t *testing.T) {
	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		timeout                time.Duration
	}{
		{
			name:    "Success - Update Title Only",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				data2 := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: nil}})

				newTitle := roundtypes.Title("Updated Round Title")
				payload := createRoundUpdateValidatedPayload(roundID, data.UserID, &newTitle, nil, nil, nil, nil)
				b, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundUpdateValidatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundUpdatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundUpdatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundUpdatedV1)
				}
				var result roundevents.RoundEntityUpdatedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &result); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if result.Round.Title != "Updated Round Title" {
					t.Errorf("Expected Title '%s', got '%s'", "Updated Round Title", result.Round.Title)
				}
			},
			timeout: 10 * time.Second,
		},
		{
			name:    "Success - Update Description Only",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{})

				newDesc := roundtypes.Description("Updated description for the round")
				payload := createRoundUpdateValidatedPayload(roundID, data.UserID, nil, &newDesc, nil, nil, nil)
				b, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundUpdateValidatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundUpdatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundUpdatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundUpdatedV1)
				}
				var result roundevents.RoundEntityUpdatedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &result); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if result.Round.Description != "Updated description for the round" {
					t.Errorf("Expected Description '%s', got %v", "Updated description for the round", result.Round.Description)
				}
			},
			timeout: 10 * time.Second,
		},
		{
			name:    "Success - Update Location Only",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{})

				newLocation := roundtypes.Location("Updated Course Location")
				payload := createRoundUpdateValidatedPayload(roundID, data.UserID, nil, nil, &newLocation, nil, nil)
				b, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundUpdateValidatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundUpdatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundUpdatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundUpdatedV1)
				}
				var result roundevents.RoundEntityUpdatedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &result); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if result.Round.Location != "Updated Course Location" {
					t.Errorf("Expected Location '%s', got %v", "Updated Course Location", result.Round.Location)
				}
			},
			timeout: 10 * time.Second,
		},
		{
			name:    "Success - Update Start Time Only",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{})

				futureTime := time.Now().Add(48 * time.Hour)
				startTime := sharedtypes.StartTime(futureTime)
				payload := createRoundUpdateValidatedPayload(roundID, data.UserID, nil, nil, nil, &startTime, nil)
				b, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundUpdateValidatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundUpdatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundUpdatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundUpdatedV1)
				}
				var result roundevents.RoundEntityUpdatedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &result); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if result.Round.StartTime == nil {
					t.Error("Expected StartTime to be set")
				}
			},
			timeout: 10 * time.Second,
		},
		{
			name:    "Success - Update Event Type Only",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{})

				newEventType := roundtypes.DefaultEventType
				payload := createRoundUpdateValidatedPayload(roundID, data.UserID, nil, nil, nil, nil, &newEventType)
				b, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundUpdateValidatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundUpdatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundUpdatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundUpdatedV1)
				}
				var result roundevents.RoundEntityUpdatedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &result); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if result.Round.EventType == nil || *result.Round.EventType != roundtypes.DefaultEventType {
					t.Errorf("Expected EventType '%s', got %v", roundtypes.DefaultEventType, result.Round.EventType)
				}
			},
			timeout: 10 * time.Second,
		},
		{
			name:    "Success - Update Multiple Fields",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				data2 := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{{UserID: data2.UserID, Response: roundtypes.ResponseTentative, Score: nil}})

				newTitle := roundtypes.Title("Multi-Update Round")
				newDesc := roundtypes.Description("Updated with multiple fields")
				newLocation := roundtypes.Location("New Multi-Field Location")
				futureTime := time.Now().Add(72 * time.Hour)
				startTime := sharedtypes.StartTime(futureTime)
				newEventType := roundtypes.DefaultEventType

				payload := createRoundUpdateValidatedPayload(roundID, data.UserID, &newTitle, &newDesc, &newLocation, &startTime, &newEventType)
				b, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundUpdateValidatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundUpdatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundUpdatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundUpdatedV1)
				}
				var result roundevents.RoundEntityUpdatedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &result); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if result.Round.Title != "Multi-Update Round" {
					t.Errorf("Expected Title '%s', got '%s'", "Multi-Update Round", result.Round.Title)
				}
			},
			timeout: 10 * time.Second,
		},
		{
			name:    "Success - Update Round with Existing Participants",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				data2 := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				score1 := sharedtypes.Score(3)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1}})

				newTitle := roundtypes.Title("Updated Round with Participants")
				payload := createRoundUpdateValidatedPayload(roundID, data.UserID, &newTitle, nil, nil, nil, nil)
				b, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundUpdateValidatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundUpdatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundUpdatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundUpdatedV1)
				}
				var result roundevents.RoundEntityUpdatedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &result); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if len(result.Round.Participants) != 1 {
					t.Errorf("Expected 1 participant, got %d", len(result.Round.Participants))
				}
			},
			timeout: 10 * time.Second,
		},
		{
			name:    "Failure - Round Not Found",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				newTitle := roundtypes.Title("Title for Nonexistent Round")
				payload := createRoundUpdateValidatedPayload(nonExistentRoundID, data.UserID, &newTitle, nil, nil, nil, nil)
				b, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundUpdateValidatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundUpdateErrorV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundUpdateErrorV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundUpdateErrorV1)
				}
				var result roundevents.RoundUpdateErrorPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &result); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if result.Error == "" {
					t.Error("Expected Error message to be populated")
				}
			},
			timeout: 10 * time.Second,
		},
		{
			name:    "Failure - Invalid JSON Message",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				invalidMsg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
				invalidMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundUpdateValidatedV1, invalidMsg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return invalidMsg
			},
			expectedOutgoingTopics: []string{},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				if len(receivedMsgs) != 0 {
					t.Errorf("Expected no messages for invalid JSON, got %d topics", len(receivedMsgs))
				}
			},
			timeout: 10 * time.Second,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestRoundHandler(t)
			// Ensure streams are created
			ensureStreams(t, deps.TestEnvironment)

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

// Helper functions for creating payloads - UNIQUE TO ROUND UPDATE VALIDATED TESTS
func createRoundUpdateValidatedPayload(
	roundID sharedtypes.RoundID,
	userID sharedtypes.DiscordID,
	title *roundtypes.Title,
	description *roundtypes.Description,
	location *roundtypes.Location,
	startTime *sharedtypes.StartTime,
	eventType *roundtypes.EventType,
) roundevents.RoundUpdateValidatedPayloadV1 {
	// Create the inner request payload
	requestPayload := roundevents.RoundUpdateRequestPayloadV1{}
	requestPayload.RoundID = roundID
	requestPayload.UserID = userID
	requestPayload.GuildID = "test-guild" // Always set for multi-tenant correctness

	// Set optional fields if provided
	if title != nil {
		requestPayload.Title = title
	}
	if description != nil {
		requestPayload.Description = description
	}
	if location != nil {
		requestPayload.Location = location
	}
	if startTime != nil {
		requestPayload.StartTime = startTime
	}
	if eventType != nil {
		requestPayload.EventType = eventType
	}

	return roundevents.RoundUpdateValidatedPayloadV1{
		GuildID:                   "test-guild",
		RoundUpdateRequestPayload: requestPayload,
	}
}

// Publishing functions - UNIQUE TO ROUND UPDATE VALIDATED TESTS
// NOTE: MessageCapture- and context.Background()-dependent helpers removed.
// Use per-test PublishMsgFn (which should call testutils.PublishMessage with
// env.EventBus and env.Ctx) and perform outgoing message inspection inside the
// TestCase ValidateFn using the receivedMsgs map and
// deps.TestHelpers.UnmarshalPayload(msg, &payload).
