package roundhandler_integration_tests

import (
	"encoding/json"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

func TestHandleTagNumberFound(t *testing.T) {
	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		timeout                time.Duration
	}{
		{
			name: "Success - Accept Response with Tag Found",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				id := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)
				return id
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)
				tagNumber := sharedtypes.TagNumber(42)
				payload := createTagLookupFoundPayload(roundID, data.UserID, &tagNumber, roundtypes.ResponseAccept, boolPtr(false))
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, sharedevents.RoundTagLookupFoundV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundParticipantJoinedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundParticipantJoinedV1]
				if len(msgs) == 0 {
					t.Fatalf("expected participant joined message, got none")
				}
			},
			timeout: 10 * time.Second,
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

// Helper functions for creating payloads - UNIQUE TO TAG LOOKUP TESTS
func createTagLookupFoundPayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, tagNumber *sharedtypes.TagNumber, originalResponse roundtypes.Response, originalJoinedLate *bool) sharedevents.RoundTagLookupResultPayloadV1 {
	return sharedevents.RoundTagLookupResultPayloadV1{
		ScopedGuildID:      sharedevents.ScopedGuildID{GuildID: "test-guild"},
		RoundID:            roundID,
		UserID:             userID,
		TagNumber:          tagNumber,
		OriginalResponse:   originalResponse,
		OriginalJoinedLate: originalJoinedLate,
	}
}

func createTagLookupNotFoundPayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, originalResponse roundtypes.Response, originalJoinedLate *bool) sharedevents.RoundTagLookupResultPayloadV1 {
	return sharedevents.RoundTagLookupResultPayloadV1{
		ScopedGuildID:      sharedevents.ScopedGuildID{GuildID: "test-guild"},
		RoundID:            roundID,
		UserID:             userID,
		TagNumber:          nil, // No tag found
		OriginalResponse:   originalResponse,
		OriginalJoinedLate: originalJoinedLate,
	}
}

// Publishing functions - UNIQUE TO TAG LOOKUP TESTS
// MessageCapture-dependent helpers removed; tests should use testutils.RunTest and validate via
// receivedMsgs map with deps.TestHelpers.UnmarshalPayload in the ValidateFn.
