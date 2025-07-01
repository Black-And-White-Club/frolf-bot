package roundintegrationtests

import (
	"context"
	"strings"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

// TestUpdateScheduledRoundsWithNewTags is the main test function for the service method.
func TestUpdateScheduledRoundsWithNewTags(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) roundevents.ScheduledRoundTagUpdatePayload
		expectedError            bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult)
	}{
		{
			name: "Successful update of scheduled rounds with new tags",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) roundevents.ScheduledRoundTagUpdatePayload {
				generator := testutils.NewTestDataGenerator()

				// Create two upcoming rounds with participants
				round1 := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("creator1"),
					Title:     "Upcoming Round 1",
					State:     roundtypes.RoundStateUpcoming,
				})
				round1.GuildID = "test-guild"
				round1.GuildID = "test-guild"
				// Ensure start_time is set (required field) - check if it's nil or zero
				if round1.StartTime == nil || round1.StartTime.AsTime().IsZero() {
					startTime := sharedtypes.StartTime(time.Now().Add(24 * time.Hour))
					round1.StartTime = &startTime
				}

				// Add participants to round1
				oldTag1 := sharedtypes.TagNumber(100)
				oldTag2 := sharedtypes.TagNumber(200)
				round1.Participants = []roundtypes.Participant{
					{
						UserID:    sharedtypes.DiscordID("user1"),
						TagNumber: &oldTag1,
					},
					{
						UserID:    sharedtypes.DiscordID("user2"),
						TagNumber: &oldTag2,
					},
					{
						UserID:    sharedtypes.DiscordID("user3"), // This user won't have tag changes
						TagNumber: nil,
					},
				}

				round2 := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("creator2"),
					Title:     "Upcoming Round 2",
					State:     roundtypes.RoundStateUpcoming,
				})
				round2.GuildID = "test-guild"
				round2.GuildID = "test-guild"
				// Ensure start_time is set (required field) - check if it's nil or zero
				if round2.StartTime == nil || round2.StartTime.AsTime().IsZero() {
					startTime := sharedtypes.StartTime(time.Now().Add(48 * time.Hour))
					round2.StartTime = &startTime
				}

				// Add participants to round2
				oldTag3 := sharedtypes.TagNumber(300)
				round2.Participants = []roundtypes.Participant{
					{
						UserID:    sharedtypes.DiscordID("user1"), // Same user as in round1
						TagNumber: &oldTag3,
					},
					{
						UserID:    sharedtypes.DiscordID("user4"),
						TagNumber: nil,
					},
				}

				err := deps.DB.CreateRound(ctx, "test-guild", &round1)
				if err != nil {
					t.Fatalf("Failed to create round1 in DB for test setup: %v", err)
				}

				err = deps.DB.CreateRound(ctx, "test-guild", &round2)
				if err != nil {
					t.Fatalf("Failed to create round2 in DB for test setup: %v", err)
				}

				// Define the tag changes
				newTag1 := sharedtypes.TagNumber(111)
				newTag2 := sharedtypes.TagNumber(222)

				return roundevents.ScheduledRoundTagUpdatePayload{
					GuildID: "test-guild",
					ChangedTags: map[sharedtypes.DiscordID]*sharedtypes.TagNumber{
						"user1": &newTag1, // This user is in both rounds
						"user2": &newTag2, // This user is only in round1
						"user5": &newTag1, // This user is not in any rounds
					},
				}
			},
			expectedError: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}

				updatePayload, ok := returnedResult.Success.(*roundevents.TagsUpdatedForScheduledRoundsPayload)
				if !ok {
					t.Errorf("Expected *TagsUpdatedForScheduledRoundsPayload, got %T", returnedResult.Success)
					return
				}

				// Verify that we have the expected number of rounds updated
				if len(updatePayload.UpdatedRounds) != 2 {
					t.Errorf("Expected 2 rounds to update, got %d", len(updatePayload.UpdatedRounds))
				}

				// Verify the summary shows correct counts
				if updatePayload.Summary.RoundsUpdated != 2 {
					t.Errorf("Expected Summary.RoundsUpdated to be 2, got %d", updatePayload.Summary.RoundsUpdated)
				}

				if updatePayload.Summary.ParticipantsUpdated != 3 {
					t.Errorf("Expected Summary.ParticipantsUpdated to be 3 (user1 in 2 rounds + user2 in 1 round), got %d", updatePayload.Summary.ParticipantsUpdated)
				}

				// Verify the tag updates were applied correctly by checking each round's participants
				expectedTagUpdates := map[sharedtypes.DiscordID]sharedtypes.TagNumber{
					"user1": 111,
					"user2": 222,
				}

				foundUsers := make(map[sharedtypes.DiscordID]int) // Track how many times each user appears
				for _, roundInfo := range updatePayload.UpdatedRounds {
					if roundInfo.EventMessageID == "" {
						t.Errorf("Expected EventMessageID to be set for round %s", roundInfo.RoundID)
					}

					for _, participant := range roundInfo.UpdatedParticipants {
						if expectedTag, exists := expectedTagUpdates[participant.UserID]; exists {
							foundUsers[participant.UserID]++
							if participant.TagNumber == nil {
								t.Errorf("Expected participant %s to have tag number %d, but got nil", participant.UserID, expectedTag)
							} else if *participant.TagNumber != expectedTag {
								t.Errorf("Expected participant %s to have tag number %d, got %d", participant.UserID, expectedTag, *participant.TagNumber)
							}
						}
					}
				}

				// Verify user1 appears in 2 rounds and user2 appears in 1 round
				if foundUsers["user1"] != 2 {
					t.Errorf("Expected user1 to appear in 2 rounds, but found %d", foundUsers["user1"])
				}
				if foundUsers["user2"] != 1 {
					t.Errorf("Expected user2 to appear in 1 round, but found %d", foundUsers["user2"])
				}

				// Verify the rounds were actually updated in the database
				rounds, err := deps.DB.GetUpcomingRounds(ctx, "test-guild")
				if err != nil {
					t.Fatalf("Failed to get upcoming rounds from DB: %v", err)
				}

				// Check that the participants in the DB have the updated tag numbers
				updatedParticipants := make(map[sharedtypes.DiscordID]*sharedtypes.TagNumber)
				for _, round := range rounds {
					for _, participant := range round.Participants {
						if participant.TagNumber != nil {
							updatedParticipants[participant.UserID] = participant.TagNumber
						}
					}
				}

				// Verify user1 has the new tag
				if tag, exists := updatedParticipants["user1"]; !exists || tag == nil || *tag != 111 {
					t.Errorf("Expected user1 to have tag 111 in DB, got %v", tag)
				}

				// Verify user2 has the new tag
				if tag, exists := updatedParticipants["user2"]; !exists || tag == nil || *tag != 222 {
					t.Errorf("Expected user2 to have tag 222 in DB, got %v", tag)
				}
			},
		},
		{
			name: "No rounds to update when no participants match changed tags",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) roundevents.ScheduledRoundTagUpdatePayload {
				generator := testutils.NewTestDataGenerator()

				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("creator1"),
					Title:     "Upcoming Round",
					State:     roundtypes.RoundStateUpcoming,
				})
				round.GuildID = "test-guild"
				round.GuildID = "test-guild"
				// Ensure start_time is set (required field) - check if it's nil or zero
				if round.StartTime == nil || round.StartTime.AsTime().IsZero() {
					startTime := sharedtypes.StartTime(time.Now().Add(24 * time.Hour))
					round.StartTime = &startTime
				}

				// Add participants that won't match the changed tags
				oldTag := sharedtypes.TagNumber(100)
				round.Participants = []roundtypes.Participant{
					{
						UserID:    sharedtypes.DiscordID("user1"),
						TagNumber: &oldTag,
					},
				}

				err := deps.DB.CreateRound(ctx, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create round in DB for test setup: %v", err)
				}

				// Define tag changes for users not in any rounds
				newTag := sharedtypes.TagNumber(999)
				return roundevents.ScheduledRoundTagUpdatePayload{
					GuildID: "test-guild",
					ChangedTags: map[sharedtypes.DiscordID]*sharedtypes.TagNumber{
						"nonexistent_user": &newTag,
					},
				}
			},
			expectedError: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}

				updatePayload, ok := returnedResult.Success.(*roundevents.TagsUpdatedForScheduledRoundsPayload)
				if !ok {
					t.Errorf("Expected *TagsUpdatedForScheduledRoundsPayload, got %T", returnedResult.Success)
					return
				}

				// Verify that no rounds need to be updated
				if len(updatePayload.UpdatedRounds) != 0 {
					t.Errorf("Expected 0 rounds to update, got %d", len(updatePayload.UpdatedRounds))
				}

				// Verify summary shows no updates
				if updatePayload.Summary.RoundsUpdated != 0 {
					t.Errorf("Expected Summary.RoundsUpdated to be 0, got %d", updatePayload.Summary.RoundsUpdated)
				}

				if updatePayload.Summary.ParticipantsUpdated != 0 {
					t.Errorf("Expected Summary.ParticipantsUpdated to be 0, got %d", updatePayload.Summary.ParticipantsUpdated)
				}
			},
		},
		{
			name: "Update with empty ChangedTags map",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) roundevents.ScheduledRoundTagUpdatePayload {
				generator := testutils.NewTestDataGenerator()

				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("creator1"),
					Title:     "Upcoming Round",
					State:     roundtypes.RoundStateUpcoming,
				})
				round.GuildID = "test-guild"
				round.GuildID = "test-guild"
				// Ensure start_time is set (required field) - check if it's nil or zero
				if round.StartTime == nil || round.StartTime.AsTime().IsZero() {
					startTime := sharedtypes.StartTime(time.Now().Add(24 * time.Hour))
					round.StartTime = &startTime
				}

				oldTag := sharedtypes.TagNumber(100)
				round.Participants = []roundtypes.Participant{
					{
						UserID:    sharedtypes.DiscordID("user1"),
						TagNumber: &oldTag,
					},
				}

				err := deps.DB.CreateRound(ctx, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create round in DB for test setup: %v", err)
				}

				return roundevents.ScheduledRoundTagUpdatePayload{
					GuildID:     "test-guild",
					ChangedTags: make(map[sharedtypes.DiscordID]*sharedtypes.TagNumber),
				}
			},
			expectedError: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}

				updatePayload, ok := returnedResult.Success.(*roundevents.TagsUpdatedForScheduledRoundsPayload)
				if !ok {
					t.Errorf("Expected *TagsUpdatedForScheduledRoundsPayload, got %T", returnedResult.Success)
					return
				}

				// Verify that no rounds need to be updated (empty changed tags)
				if len(updatePayload.UpdatedRounds) != 0 {
					t.Errorf("Expected 0 rounds to update, got %d", len(updatePayload.UpdatedRounds))
				}

				// Verify summary shows no updates
				if updatePayload.Summary.RoundsUpdated != 0 {
					t.Errorf("Expected Summary.RoundsUpdated to be 0, got %d", updatePayload.Summary.RoundsUpdated)
				}

				if updatePayload.Summary.ParticipantsUpdated != 0 {
					t.Errorf("Expected Summary.ParticipantsUpdated to be 0, got %d", updatePayload.Summary.ParticipantsUpdated)
				}
			},
		},
		{
			name: "Update with nil tag values",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) roundevents.ScheduledRoundTagUpdatePayload {
				generator := testutils.NewTestDataGenerator()

				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("creator1"),
					Title:     "Upcoming Round",
					State:     roundtypes.RoundStateUpcoming,
				})
				round.GuildID = "test-guild"
				round.GuildID = "test-guild"
				// Ensure start_time is set (required field) - check if it's nil or zero
				if round.StartTime == nil || round.StartTime.AsTime().IsZero() {
					startTime := sharedtypes.StartTime(time.Now().Add(24 * time.Hour))
					round.StartTime = &startTime
				}

				oldTag := sharedtypes.TagNumber(100)
				round.Participants = []roundtypes.Participant{
					{
						UserID:    sharedtypes.DiscordID("user1"),
						TagNumber: &oldTag,
					},
				}

				err := deps.DB.CreateRound(ctx, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create round in DB for test setup: %v", err)
				}

				// Set tag to nil (removing the tag)
				return roundevents.ScheduledRoundTagUpdatePayload{
					GuildID: "test-guild",
					ChangedTags: map[sharedtypes.DiscordID]*sharedtypes.TagNumber{
						"user1": nil,
					},
				}
			},
			expectedError: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}

				updatePayload, ok := returnedResult.Success.(*roundevents.TagsUpdatedForScheduledRoundsPayload)
				if !ok {
					t.Errorf("Expected *TagsUpdatedForScheduledRoundsPayload, got %T", returnedResult.Success)
					return
				}

				// Verify that the round is updated with the participant having nil tag
				if len(updatePayload.UpdatedRounds) != 1 {
					t.Errorf("Expected 1 round to update, got %d", len(updatePayload.UpdatedRounds))
				}

				if len(updatePayload.UpdatedRounds) > 0 {
					roundInfo := updatePayload.UpdatedRounds[0]
					foundUser := false
					for _, participant := range roundInfo.UpdatedParticipants {
						if participant.UserID == "user1" {
							foundUser = true
							if participant.TagNumber != nil {
								t.Errorf("Expected participant to have nil tag number, got %v", *participant.TagNumber)
							}
							break
						}
					}
					if !foundUser {
						t.Errorf("Expected to find user1 in updated participants")
					}
				}

				// Verify summary shows 1 participant was updated
				if updatePayload.Summary.RoundsUpdated != 1 {
					t.Errorf("Expected Summary.RoundsUpdated to be 1, got %d", updatePayload.Summary.RoundsUpdated)
				}

				if updatePayload.Summary.ParticipantsUpdated != 1 {
					t.Errorf("Expected Summary.ParticipantsUpdated to be 1, got %d", updatePayload.Summary.ParticipantsUpdated)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)

			var payload roundevents.ScheduledRoundTagUpdatePayload
			if tt.setupTestEnv != nil {
				payload = tt.setupTestEnv(deps.Ctx, deps)
			} else {
				payload = roundevents.ScheduledRoundTagUpdatePayload{
					ChangedTags: make(map[sharedtypes.DiscordID]*sharedtypes.TagNumber),
				}
			}

			// Call the actual service method
			result, err := deps.Service.UpdateScheduledRoundsWithNewTags(deps.Ctx, payload)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				} else if tt.expectedErrorMessagePart != "" && !strings.Contains(err.Error(), tt.expectedErrorMessagePart) {
					t.Errorf("Expected error message to contain '%s', but got: '%v'", tt.expectedErrorMessagePart, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			}

			if tt.validateResult != nil {
				tt.validateResult(t, deps.Ctx, deps, result)
			}
		})
	}
}
