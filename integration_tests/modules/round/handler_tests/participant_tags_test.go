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

func TestHandleScheduledRoundTagUpdate(t *testing.T) {
	tests := []struct {
		name                   string
		expectedOutgoingTopics []string
		timeout                time.Duration
	}{
		{
			name:                   "Success - Tag Update for Single Round with Multiple Participants",
			expectedOutgoingTopics: []string{roundevents.TagsUpdatedForScheduledRoundsV1},
			timeout:                5 * time.Second,
		},
		{
			name:                   "Success - Tag Update for Multiple Rounds with Same Participant",
			expectedOutgoingTopics: []string{roundevents.TagsUpdatedForScheduledRoundsV1},
			timeout:                5 * time.Second,
		},
		{
			name:                   "Success - Empty Tag Update (No Upcoming Rounds with Matching Participants)",
			expectedOutgoingTopics: []string{},
			timeout:                2 * time.Second,
		},
		{
			name:                   "Success - Tag Update Only Affects Upcoming Rounds",
			expectedOutgoingTopics: []string{roundevents.TagsUpdatedForScheduledRoundsV1},
			timeout:                5 * time.Second,
		},
		{
			name:                   "Invalid JSON - Scheduled Round Tag Update Handler",
			expectedOutgoingTopics: []string{},
			timeout:                2 * time.Second,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestRoundHandler(t)

			// Generate shared test data for this specific subtest
			// This ensures SetupFn and PublishMsgFn use the SAME IDs.
			data1 := NewTestData()
			data2 := NewTestData()

			genericCase := testutils.TestCase{
				Name: tc.name,
				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
					helper := testutils.NewRoundTestHelper(env.EventBus, nil)

					switch tc.name {
					case "Success - Tag Update for Single Round with Multiple Participants":
						oldTag1, oldTag2 := sharedtypes.TagNumber(10), sharedtypes.TagNumber(20)
						helper.CreateUpcomingRoundWithParticipantAndTagInDB(t, deps.DB, data1.UserID, roundtypes.ResponseAccept, &oldTag1)
						helper.CreateUpcomingRoundWithParticipantAndTagInDB(t, deps.DB, data2.UserID, roundtypes.ResponseAccept, &oldTag2)

					case "Success - Tag Update for Multiple Rounds with Same Participant":
						oldTag := sharedtypes.TagNumber(50)
						helper.CreateUpcomingRoundWithParticipantAndTagInDB(t, deps.DB, data1.UserID, roundtypes.ResponseAccept, &oldTag)
						helper.CreateUpcomingRoundWithParticipantAndTagInDB(t, deps.DB, data1.UserID, roundtypes.ResponseAccept, &oldTag)

					case "Success - Empty Tag Update (No Upcoming Rounds with Matching Participants)",
						"Success - Tag Update Only Affects Upcoming Rounds":
						oldTag := sharedtypes.TagNumber(75)
						helper.CreateUpcomingRoundWithParticipantAndTagInDB(t, deps.DB, data1.UserID, roundtypes.ResponseAccept, &oldTag)
					}
					return nil
				},
				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
					var changedTags map[sharedtypes.DiscordID]*sharedtypes.TagNumber
					newTag1 := sharedtypes.TagNumber(42)
					newTag2 := sharedtypes.TagNumber(99)

					switch tc.name {
					case "Success - Tag Update for Single Round with Multiple Participants":
						changedTags = map[sharedtypes.DiscordID]*sharedtypes.TagNumber{
							data1.UserID: &newTag1,
							data2.UserID: &newTag2,
						}
					case "Success - Tag Update for Multiple Rounds with Same Participant",
						"Success - Tag Update Only Affects Upcoming Rounds":
						changedTags = map[sharedtypes.DiscordID]*sharedtypes.TagNumber{
							data1.UserID: &newTag1,
						}
					case "Success - Empty Tag Update (No Upcoming Rounds with Matching Participants)":
						changedTags = map[sharedtypes.DiscordID]*sharedtypes.TagNumber{
							sharedtypes.DiscordID(uuid.New().String()): &newTag1,
						}
					case "Invalid JSON - Scheduled Round Tag Update Handler":
						msg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
						msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
						testutils.PublishMessage(t, env.EventBus, env.Ctx, sharedevents.TagUpdateForScheduledRoundsV1, msg)
						return msg
					}

					payload := createScheduledRoundTagUpdatePayload(changedTags)
					payloadBytes, err := json.Marshal(payload)
					if err != nil {
						t.Fatalf("Failed to marshal payload: %v", err)
					}
					msg := message.NewMessage(uuid.New().String(), payloadBytes)
					msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
					if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, sharedevents.TagUpdateForScheduledRoundsV1, msg); err != nil {
						t.Fatalf("Failed to publish message: %v", err)
					}
					return msg
				},
				ExpectedTopics: tc.expectedOutgoingTopics,
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
					if len(tc.expectedOutgoingTopics) == 0 {
						successMsgs := receivedMsgs[roundevents.TagsUpdatedForScheduledRoundsV1]
						if len(successMsgs) > 0 {
							t.Errorf("Expected 0 messages, got %d", len(successMsgs))
						}
						return
					}

					msgs := receivedMsgs[roundevents.TagsUpdatedForScheduledRoundsV1]
					if len(msgs) == 0 {
						t.Fatalf("Expected at least one message on topic %q", roundevents.TagsUpdatedForScheduledRoundsV1)
					}

					var payload roundevents.TagsUpdatedForScheduledRoundsPayloadV1
					if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
						t.Fatalf("Failed to unmarshal payload: %v", err)
					}

					if tc.name == "Success - Tag Update for Single Round with Multiple Participants" {
						if payload.Summary.ParticipantsUpdated < 1 {
							t.Errorf("Expected updates, but summary shows 0 participants updated")
						}
					}
				},
				MessageTimeout: tc.timeout,
			}
			testutils.RunTest(t, genericCase, deps.TestEnvironment)
		})
	}
}

func createScheduledRoundTagUpdatePayload(changedTags map[sharedtypes.DiscordID]*sharedtypes.TagNumber) roundevents.ScheduledRoundTagUpdatePayloadV1 {
	return roundevents.ScheduledRoundTagUpdatePayloadV1{
		GuildID:     "test-guild",
		ChangedTags: changedTags,
	}
}
