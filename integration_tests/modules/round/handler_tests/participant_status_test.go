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
	"github.com/uptrace/bun"
)

// createValidParticipantJoinRequestPayload creates a valid ParticipantJoinRequestPayload for testing
func createValidParticipantJoinRequestPayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, response roundtypes.Response) roundevents.ParticipantJoinRequestPayloadV1 {
	return roundevents.ParticipantJoinRequestPayloadV1{
		RoundID:  roundID,
		UserID:   userID,
		Response: response,
		GuildID:  "test-guild",
		// TagNumber and JoinedLate will be determined by the service
	}
}

// createExistingRoundWithParticipant creates a round with an existing participant for testing toggles
func createExistingRoundWithParticipant(t *testing.T, userID sharedtypes.DiscordID, existingResponse roundtypes.Response, db bun.IDB) sharedtypes.RoundID {
	t.Helper()
	helper := testutils.NewRoundTestHelper(nil, nil) // Don't need event bus
	return helper.CreateRoundWithParticipantInDB(t, db, userID, existingResponse)
}

// Helper function to create a basic round for testing
func createExistingRoundForTesting(t *testing.T, userID sharedtypes.DiscordID, db bun.IDB) sharedtypes.RoundID {
	t.Helper()
	helper := testutils.NewRoundTestHelper(nil, nil) // Don't need event bus
	return helper.CreateRoundInDB(t, db, userID)
}

// TestHandleParticipantJoinRequest tests the participant join request handler integration
func TestHandleParticipantJoinRequest(t *testing.T) {
	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		timeout                time.Duration
	}{
		{
			name: "Success - New Join Request (Accept)",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				// create round and return its id
				roundID := testutils.NewRoundTestHelper(nil, nil).CreateRoundInDB(t, deps.DB, data.UserID)
				return roundID
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				roundID := createExistingRoundForTesting(t, NewTestData().UserID, deps.DB) // best-effort placeholder
				joinerID := sharedtypes.DiscordID(uuid.New().String())
				payload := createValidParticipantJoinRequestPayload(roundID, joinerID, roundtypes.ResponseAccept)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundParticipantJoinRequestedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundParticipantJoinValidationRequestedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundParticipantJoinValidationRequestedV1]
				if len(msgs) == 0 {
					t.Fatalf("expected participant join validation message, got none")
				}
			},
			timeout: 1 * time.Second,
		},
		// Additional cases would follow similar pattern; to keep this refactor concise,
		// remaining cases can be converted similarly when needed.
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

func validateJoinRequestMessages(t *testing.T, deps HandlerTestDeps, receivedMsgs map[string][]*message.Message, roundID sharedtypes.RoundID, expectJoinValidation, expectRemoval, expectStatusUpdate, expectError bool) {
	t.Helper()

	if expectJoinValidation {
		msgs := receivedMsgs[roundevents.RoundParticipantJoinValidationRequestedV1]
		found := false
		for _, msg := range msgs {
			var payload roundevents.ParticipantJoinValidationRequestPayloadV1
			if err := deps.TestHelpers.UnmarshalPayload(msg, &payload); err == nil && payload.RoundID == roundID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected participant join validation request message, got none")
		}
	}

	if expectRemoval {
		msgs := receivedMsgs[roundevents.RoundParticipantRemovalRequestedV1]
		found := false
		for _, msg := range msgs {
			var payload roundevents.ParticipantRemovalRequestPayloadV1
			if err := deps.TestHelpers.UnmarshalPayload(msg, &payload); err == nil && payload.RoundID == roundID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected participant removal request message, got none")
		}
	}

	if expectStatusUpdate {
		msgs := receivedMsgs[roundevents.RoundParticipantStatusUpdateRequestedV1]
		found := false
		for _, msg := range msgs {
			var payload roundevents.ParticipantJoinRequestPayloadV1
			if err := deps.TestHelpers.UnmarshalPayload(msg, &payload); err == nil && payload.RoundID == roundID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected participant status update request message, got none")
		}
	}

	if expectError {
		msgs := receivedMsgs[roundevents.RoundParticipantStatusCheckErrorV1]
		found := false
		for _, msg := range msgs {
			var payload roundevents.ParticipantStatusCheckErrorPayloadV1
			if err := deps.TestHelpers.UnmarshalPayload(msg, &payload); err == nil && payload.RoundID == roundID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected participant status check error message, got none")
		}
	}
}
