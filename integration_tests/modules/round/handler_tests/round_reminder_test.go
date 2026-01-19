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

func TestHandleRoundReminder(t *testing.T) {
	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		timeout                time.Duration
	}{
		{
			name: "Success - Process Round Reminder for Upcoming Round with Participants",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				response := roundtypes.ResponseAccept
				tagNumber := sharedtypes.TagNumber(1)
				roundID := helper.CreateUpcomingRoundWithParticipantAndTagInDB(t, deps.DB, data.UserID, response, &tagNumber)
				return struct {
					RoundID sharedtypes.RoundID
					UserID  sharedtypes.DiscordID
				}{RoundID: roundID, UserID: data.UserID}
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				response := roundtypes.ResponseAccept
				tagNumber := sharedtypes.TagNumber(1)
				roundID := helper.CreateUpcomingRoundWithParticipantAndTagInDB(t, deps.DB, data.UserID, response, &tagNumber)

				reminderTime := time.Now().Add(1 * time.Hour)
				payload := createRoundReminderPayload(roundID, "15min", []sharedtypes.DiscordID{data.UserID}, &reminderTime)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundReminderScheduledV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundReminderSentV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundReminderSentV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundReminderSentV1)
				}
				var payload roundevents.DiscordReminderPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("Expected RoundID to be set")
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Success - Process Round Reminder with No Participants",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)
				return struct {
					RoundID sharedtypes.RoundID
					UserID  sharedtypes.DiscordID
				}{RoundID: roundID, UserID: data.UserID}
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)
				reminderTime := time.Now().Add(30 * time.Minute)
				payload := createRoundReminderPayload(roundID, "30min", []sharedtypes.DiscordID{data.UserID}, &reminderTime)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundReminderScheduledV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				validatedMsgs := receivedMsgs[roundevents.RoundReminderSentV1]
				errorMsgs := receivedMsgs[roundevents.RoundReminderFailedV1]
				if len(validatedMsgs) > 0 || len(errorMsgs) > 0 {
					t.Errorf("Expected no messages for empty participants, got validated=%d, error=%d", len(validatedMsgs), len(errorMsgs))
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Success - Process Round Reminder with Empty User List",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)
				return struct {
					RoundID sharedtypes.RoundID
					UserID  sharedtypes.DiscordID
				}{RoundID: roundID, UserID: data.UserID}
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)
				reminderTime := time.Now().Add(1 * time.Hour)
				payload := createRoundReminderPayload(roundID, "1hour", []sharedtypes.DiscordID{}, &reminderTime)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundReminderScheduledV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				validatedMsgs := receivedMsgs[roundevents.RoundReminderSentV1]
				errorMsgs := receivedMsgs[roundevents.RoundReminderFailedV1]
				if len(validatedMsgs) > 0 || len(errorMsgs) > 0 {
					t.Errorf("Expected no messages for empty user list, got validated=%d, error=%d", len(validatedMsgs), len(errorMsgs))
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Failure - Process Reminder for Non-Existent Round",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				reminderTime := time.Now().Add(1 * time.Hour)
				payload := createRoundReminderPayload(nonExistentRoundID, "15min", []sharedtypes.DiscordID{data.UserID}, &reminderTime)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundReminderScheduledV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundReminderFailedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundReminderFailedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one error message on topic %q", roundevents.RoundReminderFailedV1)
				}
				var payload roundevents.RoundErrorPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("Expected RoundID to be set in error payload")
				}
				if payload.Error == "" {
					t.Error("Expected error message to be populated")
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name:    "Failure - Invalid JSON Message",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				invalidJSON := []byte("invalid json")
				msg := message.NewMessage(uuid.New().String(), invalidJSON)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundReminderScheduledV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				validatedMsgs := receivedMsgs[roundevents.RoundReminderSentV1]
				errorMsgs := receivedMsgs[roundevents.RoundReminderFailedV1]
				if len(validatedMsgs) > 0 || len(errorMsgs) > 0 {
					t.Errorf("Expected no messages for invalid JSON, got validated=%d, error=%d", len(validatedMsgs), len(errorMsgs))
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
				MessageTimeout: 5 * time.Second,
			}

			testutils.RunTest(t, genericCase, deps.TestEnvironment)
		})
	}
}

// Helper functions for creating payloads - UNIQUE TO ROUND REMINDER TESTS
func createRoundReminderPayload(roundID sharedtypes.RoundID, reminderType string, userIDs []sharedtypes.DiscordID, startTime *time.Time) roundevents.DiscordReminderPayloadV1 {
	var sharedStartTime *sharedtypes.StartTime
	if startTime != nil {
		converted := sharedtypes.StartTime(*startTime)
		sharedStartTime = &converted
	}

	location := roundtypes.Location("Test Location")
	return roundevents.DiscordReminderPayloadV1{
		GuildID:          "test-guild",
		RoundID:          roundID,
		ReminderType:     reminderType,
		RoundTitle:       roundtypes.Title("Test Round"),
		StartTime:        sharedStartTime,
		Location:         &location,
		UserIDs:          userIDs,
		DiscordChannelID: "test-channel-123",
		DiscordGuildID:   "test-guild-456",
		EventMessageID:   "test-message-789",
	}
}
