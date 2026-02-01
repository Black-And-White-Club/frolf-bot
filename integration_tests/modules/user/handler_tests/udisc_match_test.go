package userhandler_integration_tests

import (
	"encoding/json"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

func TestHandleScorecardParsed(t *testing.T) {
	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		expectHandlerError     bool
	}{
		{
			name: "Success - all players matched",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				guildID := sharedtypes.GuildID("test-guild")
				userID := sharedtypes.DiscordID("test-user")

				// Create user in DB and set UDisc username
				err := testutils.InsertUser(t, env.DB, userID, guildID, sharedtypes.UserRoleUser)
				if err != nil {
					t.Fatalf("Failed to insert user: %v", err)
				}

				_, err = env.DB.NewUpdate().
					Table("users").
					Set("udisc_username = ?", "udiscuser1").
					Where("user_id = ?", userID).
					Exec(env.Ctx)
				if err != nil {
					t.Fatalf("Failed to set udisc_username: %v", err)
				}

				return userID
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				payload := roundevents.ParsedScorecardPayloadV1{
					GuildID:  "test-guild",
					RoundID:  sharedtypes.RoundID(uuid.New()),
					ImportID: "import-1",
					UserID:   "uploader",
					ParsedData: &roundtypes.ParsedScorecard{
						PlayerScores: []roundtypes.PlayerScoreRow{
							{PlayerName: "udiscuser1"},
						},
					},
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.ScorecardParsedV1, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{userevents.UDiscMatchConfirmedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				expectedUserID := initialState.(sharedtypes.DiscordID)
				msgs := receivedMsgs[userevents.UDiscMatchConfirmedV1]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 UDiscMatchConfirmed message, got %d", len(msgs))
				}

				var payload userevents.UDiscMatchConfirmedPayloadV1
				if err := deps.UserModule.Helper.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Unmarshal error: %v", err)
				}

				if len(payload.Mappings) != 1 {
					t.Fatalf("Expected 1 mapping, got %d", len(payload.Mappings))
				}
				if payload.Mappings[0].DiscordUserID != expectedUserID {
					t.Errorf("Expected DiscordUserID %s, got %s", expectedUserID, payload.Mappings[0].DiscordUserID)
				}
				if payload.Mappings[0].PlayerName != "udiscuser1" {
					t.Errorf("Expected PlayerName udiscuser1, got %s", payload.Mappings[0].PlayerName)
				}
			},
		},
		{
			name: "Success - some players unmatched",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// No users created
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				payload := roundevents.ParsedScorecardPayloadV1{
					GuildID:  "test-guild",
					RoundID:  sharedtypes.RoundID(uuid.New()),
					ImportID: "import-2",
					UserID:   "uploader",
					ParsedData: &roundtypes.ParsedScorecard{
						PlayerScores: []roundtypes.PlayerScoreRow{
							{PlayerName: "unknown_player"},
						},
					},
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.ScorecardParsedV1, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{userevents.UDiscMatchConfirmationRequiredV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[userevents.UDiscMatchConfirmationRequiredV1]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 UDiscMatchConfirmationRequired message, got %d", len(msgs))
				}

				var payload userevents.UDiscMatchConfirmationRequiredPayloadV1
				if err := deps.UserModule.Helper.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Unmarshal error: %v", err)
				}

				if len(payload.UnmatchedPlayers) != 1 || payload.UnmatchedPlayers[0] != "unknown_player" {
					t.Errorf("Expected unmatched player unknown_player, got %v", payload.UnmatchedPlayers)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestUserHandler(t)

			genericCase := testutils.TestCase{
				Name: tc.name,
				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
					return tc.setupFn(t, deps, env)
				},
				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
					return tc.publishMsgFn(t, deps, env)
				},
				ExpectedTopics: tc.expectedOutgoingTopics,
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
					tc.validateFn(t, deps, env, incomingMsg, receivedMsgs, initialState)
				},
				ExpectError:    tc.expectHandlerError,
				MessageTimeout: 5 * time.Second,
			}
			testutils.RunTest(t, genericCase, deps.TestEnvironment)
		})
	}
}
