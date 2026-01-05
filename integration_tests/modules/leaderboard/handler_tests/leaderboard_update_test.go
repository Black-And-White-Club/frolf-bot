package leaderboardhandler_integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

// Helper: Validate success response for leaderboard update
func validateLeaderboardUpdatedPayload(t *testing.T, req leaderboardevents.LeaderboardUpdateRequestedPayloadV1, outgoing *message.Message, incoming *message.Message) leaderboardevents.LeaderboardUpdatedPayloadV1 {
	t.Helper()
	var res leaderboardevents.LeaderboardUpdatedPayloadV1
	if err := json.Unmarshal(outgoing.Payload, &res); err != nil {
		t.Fatalf("Failed to parse payload: %v", err)
	}

	if res.GuildID != req.GuildID {
		t.Errorf("GuildID mismatch: expected %s, got %s", req.GuildID, res.GuildID)
	}

	if res.RoundID != req.RoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", req.RoundID, res.RoundID)
	}

	// Validate correlation ID matches the incoming message when present
	inCorr := incoming.Metadata.Get(middleware.CorrelationIDMetadataKey)
	outCorr := outgoing.Metadata.Get(middleware.CorrelationIDMetadataKey)
	if inCorr != "" && outCorr != inCorr {
		t.Errorf("Correlation ID mismatch: expected %q, got %q", inCorr, outCorr)
	}
	return res
}

// Integration test for LeaderboardUpdateRequested handler
func TestHandleLeaderboardUpdateRequested(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	testCases := []struct {
		name                   string
		users                  []testutils.User
		setupFn                func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard
		publishMsgFn           func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message
		validateFn             func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard)
		expectedOutgoingTopics []string
		expectHandlerError     bool
		timeout                time.Duration
	}{
		{
			name:  "Success - Sorted participant tags apply to new leaderboard",
			users: generator.GenerateUsers(3),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				// Insert the generated users into the database first
				_, err := deps.DB.NewInsert().Model(&users).Exec(context.Background())
				if err != nil {
					t.Fatalf("Failed to insert users: %v", err)
				}

				// Create initial assignments with UNUSED tags to avoid conflicts
				initial := leaderboardtypes.LeaderboardData{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 10}, // User 0 starts with tag 10
					{UserID: sharedtypes.DiscordID(users[1].UserID), TagNumber: 11}, // User 1 starts with tag 11
					{UserID: sharedtypes.DiscordID(users[2].UserID), TagNumber: 12}, // User 2 starts with tag 12
				}

				initialLeaderboard := testutils.SetupLeaderboardWithEntries(t, deps.DB, initial, true, sharedtypes.RoundID(uuid.New()))
				t.Logf("SetupFn: Initial Leaderboard ID: %d, UpdateID: %s, IsActive: %t",
					initialLeaderboard.ID, initialLeaderboard.UpdateID, initialLeaderboard.IsActive)
				t.Logf("SetupFn: Created users with IDs: [%s, %s, %s]",
					users[0].UserID, users[1].UserID, users[2].UserID)
				return initialLeaderboard
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				sorted := []string{
					fmt.Sprintf("3:%s", users[2].UserID), // User 2 should get tag 3
					fmt.Sprintf("1:%s", users[0].UserID), // User 0 should get tag 1
					fmt.Sprintf("2:%s", users[1].UserID), // User 1 should get tag 2
				}
				// The RoundID in the payload should be the identifier for the NEW leaderboard state.
				payload := leaderboardevents.LeaderboardUpdateRequestedPayloadV1{
					GuildID:               sharedtypes.GuildID("test_guild"),
					RoundID:               sharedtypes.RoundID(uuid.New()),
					SortedParticipantTags: sorted,
					Source:                "integration-test",
					UpdateID:              uuid.New().String(),
				}
				payloadBytes, _ := json.Marshal(payload)
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				t.Logf("PublishMsgFn: Publishing message with RoundID: %s, SortedParticipantTags: %v",
					payload.RoundID, payload.SortedParticipantTags)
				t.Logf("PublishMsgFn: Using actual user IDs: [%s, %s, %s]",
					users[0].UserID, users[1].UserID, users[2].UserID)
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.LeaderboardUpdateRequestedV1, msg); err != nil {
					t.Fatalf("failed publishing message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				msgs := received[leaderboardevents.LeaderboardUpdatedV1]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 success message, got %d", len(msgs))
				}
				var req leaderboardevents.LeaderboardUpdateRequestedPayloadV1
				if err := json.Unmarshal(incoming.Payload, &req); err != nil {
					t.Fatalf("Invalid request payload: %v", err)
				}
				// Validate the payload of the success message
				_ = validateLeaderboardUpdatedPayload(t, req, msgs[0], incoming)

				// DB assertions
				leaderboards, err := testutils.QueryLeaderboards(t, context.Background(), deps.DB)
				if err != nil {
					t.Fatalf("DB query failed: %v", err)
				}
				t.Logf("ValidateFn: Retrieved %d leaderboards from DB", len(leaderboards))
				for i, lb := range leaderboards {
					// Corrected logging to use UpdateID
					t.Logf("ValidateFn: DB Leaderboard %d: ID: %d, UpdateID: %s, IsActive: %t, Data: %+v",
						i, lb.ID, lb.UpdateID, lb.IsActive, lb.LeaderboardData)
				}

				// We expect two leaderboards: the initial one (now inactive) and the new one (active).
				if len(leaderboards) != 2 {
					t.Fatalf("Expected 2 leaderboards (1 old + 1 new), got %d", len(leaderboards))
				}

				// Explicitly find old and new leaderboards using UpdateID and IsActive
				var oldLB, newLB *leaderboarddb.Leaderboard
				for i := range leaderboards {
					lb := &leaderboards[i] // Use pointer to avoid copying
					if lb.IsActive {
						// The new active leaderboard should have the UpdateID matching the RoundID from the incoming request
						if lb.UpdateID == req.RoundID {
							newLB = lb
						}
					} else {
						// The old inactive leaderboard should have the UpdateID of the initial leaderboard
						if lb.UpdateID == initial.UpdateID {
							oldLB = lb
						}
					}
				}

				if oldLB == nil {
					// Corrected logging to use UpdateID
					t.Fatalf("Could not find the old inactive leaderboard with UpdateID %s", initial.UpdateID)
				}
				if newLB == nil {
					// Corrected logging to use UpdateID
					t.Fatalf("Could not find the new active leaderboard with UpdateID %s", req.RoundID)
				}

				// Corrected logging to use UpdateID
				t.Logf("ValidateFn: Identified Old Leaderboard ID: %d, UpdateID: %s, IsActive: %t", oldLB.ID, oldLB.UpdateID, oldLB.IsActive)
				// Corrected logging to use UpdateID
				t.Logf("ValidateFn: Identified New Leaderboard ID: %d, UpdateID: %s, IsActive: %t", newLB.ID, newLB.UpdateID, newLB.IsActive)

				// --- Add logging for raw LeaderboardData from the new leaderboard ---
				t.Logf("ValidateFn: Raw New Leaderboard Data (newLB.LeaderboardData): %+v", newLB.LeaderboardData)
				// --- End added logging ---

				// Assert that the old leaderboard is indeed inactive
				if oldLB.IsActive {
					t.Errorf("Old leaderboard should be inactive")
				}
				// Assert that the new leaderboard is indeed active
				if !newLB.IsActive {
					t.Errorf("New leaderboard should be active")
				}

				// Compare the data of the NEW leaderboard using maps
				expectedDataMap := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)
				// --- UPDATED LOGIC HERE ---
				for _, entryStr := range req.SortedParticipantTags {
					parts := strings.Split(entryStr, ":")
					if len(parts) == 2 {
						tag, err := strconv.Atoi(parts[0])
						if err != nil {
							t.Fatalf("Failed to parse tag number from sorted participant tag string '%s': %v", entryStr, err)
						}
						userID := sharedtypes.DiscordID(parts[1])
						expectedDataMap[userID] = sharedtypes.TagNumber(tag)
					} else {
						t.Fatalf("Invalid format in sorted participant tags: '%s'", entryStr)
					}
				}
				// --- END UPDATED LOGIC ---

				t.Logf("ValidateFn: Expected Data Map: %+v", expectedDataMap)

				actualDataMap := testutils.ExtractLeaderboardDataMap(newLB.LeaderboardData)
				t.Logf("ValidateFn: Actual Data Map (from newLB): %+v", actualDataMap)

				// Check if the number of entries match
				if len(expectedDataMap) != len(actualDataMap) {
					t.Errorf("Leaderboard entry count mismatch: expected %d, got %d", len(expectedDataMap), len(actualDataMap))
				}

				// Check if each user has the expected tag in the actual data
				for userID, expectedTag := range expectedDataMap {
					actualTag, ok := actualDataMap[userID]
					if !ok {
						t.Errorf("User %s not found in new leaderboard data", userID)
					} else if actualTag != expectedTag {
						t.Errorf("Tag mismatch for user %s: expected %d, got %d", userID, expectedTag, actualTag)
					}
				}
			},
			expectedOutgoingTopics: []string{leaderboardevents.LeaderboardUpdatedV1},
			expectHandlerError:     false,
		},
		{
			name:  "Failure - Invalid JSON Payload",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				entries := []leaderboardtypes.LeaderboardEntry{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 99},
				}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, entries, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				msg := message.NewMessage(uuid.New().String(), []byte("not-a-json"))
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.LeaderboardUpdateRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				if len(received[leaderboardevents.LeaderboardUpdatedV1]) > 0 {
					t.Errorf("Unexpected success message published")
				}
				if len(received[leaderboardevents.LeaderboardUpdateFailedV1]) > 0 {
					t.Errorf("Unexpected failure message published")
				}
			},
			expectedOutgoingTopics: []string{},
			expectHandlerError:     true,
		},
		{
			name:  "Failure - Empty SortedParticipantTags",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, []leaderboardtypes.LeaderboardEntry{}, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				payload := leaderboardevents.LeaderboardUpdateRequestedPayloadV1{
					GuildID:               sharedtypes.GuildID("test_guild"),
					RoundID:               sharedtypes.RoundID(uuid.New()),
					SortedParticipantTags: []string{},
					Source:                "integration-test",
					UpdateID:              uuid.New().String(),
				}
				b, _ := json.Marshal(payload)
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.LeaderboardUpdateRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				if len(received[leaderboardevents.LeaderboardUpdatedV1]) > 0 {
					t.Errorf("Expected no success messages, but found some")
				}
				if len(received[leaderboardevents.LeaderboardUpdateFailedV1]) > 0 {
					t.Errorf("Unexpected failure message published")
				}
			},
			expectedOutgoingTopics: []string{},
			expectHandlerError:     true,
		},
		{
			name:  "Failure - Incorrect Tag Format",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, []leaderboardtypes.LeaderboardEntry{}, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				payload := leaderboardevents.LeaderboardUpdateRequestedPayloadV1{
					GuildID: sharedtypes.GuildID("test_guild"),
					RoundID: sharedtypes.RoundID(uuid.New()),
					SortedParticipantTags: []string{
						"user_no_tag",
					},
					Source:   "integration-test",
					UpdateID: uuid.New().String(),
				}
				b, _ := json.Marshal(payload)
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.LeaderboardUpdateRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				if len(received[leaderboardevents.LeaderboardUpdatedV1]) > 0 {
					t.Errorf("Unexpected success message")
				}
				if len(received[leaderboardevents.LeaderboardUpdateFailedV1]) > 0 {
					t.Errorf("Unexpected failure message")
				}
			},
			expectedOutgoingTopics: []string{},
			expectHandlerError:     true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestLeaderboardHandler(t)
			users := tc.users

			genericCase := testutils.TestCase{
				Name: tc.name,
				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
					return tc.setupFn(t, deps, users)
				},
				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
					return tc.publishMsgFn(t, deps, users)
				},
				ExpectedTopics: tc.expectedOutgoingTopics,
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, incoming *message.Message, received map[string][]*message.Message, initialState interface{}) {
					tc.validateFn(t, deps, incoming, received, initialState.(*leaderboarddb.Leaderboard))
				},
				ExpectError: tc.expectHandlerError,
			}

			testutils.RunTest(t, genericCase, deps.TestEnvironment)
		})
	}
}
