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

func TestHandleDiscordMessageIDUpdated(t *testing.T) {
	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		timeout                time.Duration
	}{
		{
			name: "Success - Schedule Events for Future Round",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				startTime := time.Now().Add(2 * time.Hour)
				roundID := helper.CreateRoundInDBWithTime(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming, &startTime)

				payload := createScheduleRoundPayload(roundID, "Test Round", &startTime, "test-message-123")
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundEventMessageIDUpdatedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{}, // Success means no error
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				// Verify no error messages
				errorMsgs := receivedMsgs[roundevents.RoundErrorV1]
				if len(errorMsgs) > 0 {
					t.Errorf("Expected no error messages, got %d", len(errorMsgs))
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Success - Schedule Events for Round Less Than 1 Hour Away",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				startTime := time.Now().Add(30 * time.Minute)
				roundID := helper.CreateRoundInDBWithTime(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming, &startTime)

				payload := createScheduleRoundPayload(roundID, "Test Round", &startTime, "test-message-456")
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundEventMessageIDUpdatedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				errorMsgs := receivedMsgs[roundevents.RoundErrorV1]
				if len(errorMsgs) > 0 {
					t.Errorf("Expected no error messages, got %d", len(errorMsgs))
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Success - Schedule Events for Round Far in Future",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				startTime := time.Now().Add(24 * time.Hour)
				roundID := helper.CreateRoundInDBWithTime(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming, &startTime)

				payload := createScheduleRoundPayload(roundID, "Future Round", &startTime, "test-message-789")
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundEventMessageIDUpdatedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				errorMsgs := receivedMsgs[roundevents.RoundErrorV1]
				if len(errorMsgs) > 0 {
					t.Errorf("Expected no error messages, got %d", len(errorMsgs))
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Success - Handle Round with Past Start Time",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				startTime := time.Now().Add(-1 * time.Hour)
				roundID := helper.CreateRoundInDBWithTime(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming, &startTime)

				payload := createScheduleRoundPayload(roundID, "Past Round", &startTime, "test-message-past")
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundEventMessageIDUpdatedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				errorMsgs := receivedMsgs[roundevents.RoundErrorV1]
				if len(errorMsgs) > 0 {
					t.Errorf("Expected no error messages, got %d", len(errorMsgs))
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
				invalidJSON := []byte(`invalid json`)
				msg := message.NewMessage(uuid.New().String(), invalidJSON)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundEventMessageIDUpdatedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				errorMsgs := receivedMsgs[roundevents.RoundErrorV1]
				if len(errorMsgs) > 0 {
					t.Errorf("Expected no error messages for invalid JSON, got %d", len(errorMsgs))
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

// Helper functions for creating payloads - UNIQUE TO SCHEDULE ROUND TESTS
func createScheduleRoundPayload(roundID sharedtypes.RoundID, title string, startTime *time.Time, eventMessageID string) roundevents.RoundScheduledPayloadV1 {
	var sharedStartTime *sharedtypes.StartTime
	if startTime != nil {
		converted := sharedtypes.StartTime(*startTime)
		sharedStartTime = &converted
	}

	desc := roundtypes.Description("Test Description")
	loc := roundtypes.Location("Test Location")
	return roundevents.RoundScheduledPayloadV1{
		GuildID: "test-guild",
		BaseRoundPayload: roundtypes.BaseRoundPayload{
			RoundID:     roundID,
			Title:       roundtypes.Title(title),
			Description: desc,
			Location:    loc,
			StartTime:   sharedStartTime,
		},
		EventMessageID: eventMessageID,
	}
}
