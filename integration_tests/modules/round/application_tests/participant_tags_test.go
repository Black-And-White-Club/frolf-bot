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

				err := deps.DB.CreateRound(ctx, &round1)
				if err != nil {
					t.Fatalf("Failed to create round1 in DB for test setup: %v", err)
				}

				err = deps.DB.CreateRound(ctx, &round2)
				if err != nil {
					t.Fatalf("Failed to create round2 in DB for test setup: %v", err)
				}

				// Define the tag changes
				newTag1 := sharedtypes.TagNumber(111)
				newTag2 := sharedtypes.TagNumber(222)

				return roundevents.ScheduledRoundTagUpdatePayload{
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

				updatePayload, ok := returnedResult.Success.(roundevents.DiscordRoundUpdatePayload)
				if !ok {
					t.Errorf("Expected DiscordRoundUpdatePayload, got %T", returnedResult.Success)
					return
				}

				// Verify that we have the expected number of participants to update
				if len(updatePayload.Participants) != 3 {
					t.Errorf("Expected 3 participants to update (user1 in 2 rounds + user2 in 1 round), got %d", len(updatePayload.Participants))
				}

				// Verify that we have updates for 2 rounds
				if len(updatePayload.RoundIDs) != 2 {
					t.Errorf("Expected 2 rounds to update, got %d", len(updatePayload.RoundIDs))
				}

				if len(updatePayload.EventMessageIDs) != 2 {
					t.Errorf("Expected 2 event message IDs, got %d", len(updatePayload.EventMessageIDs))
				}

				// Verify the tag updates were applied correctly
				expectedTagUpdates := map[sharedtypes.DiscordID]sharedtypes.TagNumber{
					"user1": 111,
					"user2": 222,
				}

				for _, participant := range updatePayload.Participants {
					expectedTag, exists := expectedTagUpdates[participant.UserID]
					if !exists {
						t.Errorf("Unexpected participant %s in update payload", participant.UserID)
						continue
					}

					if participant.TagNumber == nil {
						t.Errorf("Expected participant %s to have tag number %d, but got nil", participant.UserID, expectedTag)
					} else if *participant.TagNumber != expectedTag {
						t.Errorf("Expected participant %s to have tag number %d, got %d", participant.UserID, expectedTag, *participant.TagNumber)
					}
				}

				// Verify the rounds were actually updated in the database
				rounds, err := deps.DB.GetUpcomingRounds(ctx)
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

				err := deps.DB.CreateRound(ctx, &round)
				if err != nil {
					t.Fatalf("Failed to create round in DB for test setup: %v", err)
				}

				// Define tag changes for users not in any rounds
				newTag := sharedtypes.TagNumber(999)
				return roundevents.ScheduledRoundTagUpdatePayload{
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

				updatePayload, ok := returnedResult.Success.(roundevents.DiscordRoundUpdatePayload)
				if !ok {
					t.Errorf("Expected DiscordRoundUpdatePayload, got %T", returnedResult.Success)
					return
				}

				// Verify that no participants need to be updated
				if len(updatePayload.Participants) != 0 {
					t.Errorf("Expected 0 participants to update, got %d", len(updatePayload.Participants))
				}

				if len(updatePayload.RoundIDs) != 0 {
					t.Errorf("Expected 0 rounds to update, got %d", len(updatePayload.RoundIDs))
				}

				if len(updatePayload.EventMessageIDs) != 0 {
					t.Errorf("Expected 0 event message IDs, got %d", len(updatePayload.EventMessageIDs))
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

				err := deps.DB.CreateRound(ctx, &round)
				if err != nil {
					t.Fatalf("Failed to create round in DB for test setup: %v", err)
				}

				return roundevents.ScheduledRoundTagUpdatePayload{
					ChangedTags: make(map[sharedtypes.DiscordID]*sharedtypes.TagNumber),
				}
			},
			expectedError: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}

				updatePayload, ok := returnedResult.Success.(roundevents.DiscordRoundUpdatePayload)
				if !ok {
					t.Errorf("Expected DiscordRoundUpdatePayload, got %T", returnedResult.Success)
					return
				}

				// Verify that no participants need to be updated
				if len(updatePayload.Participants) != 0 {
					t.Errorf("Expected 0 participants to update, got %d", len(updatePayload.Participants))
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

				err := deps.DB.CreateRound(ctx, &round)
				if err != nil {
					t.Fatalf("Failed to create round in DB for test setup: %v", err)
				}

				// Set tag to nil (removing the tag)
				return roundevents.ScheduledRoundTagUpdatePayload{
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

				updatePayload, ok := returnedResult.Success.(roundevents.DiscordRoundUpdatePayload)
				if !ok {
					t.Errorf("Expected DiscordRoundUpdatePayload, got %T", returnedResult.Success)
					return
				}

				// Verify that the participant is updated with nil tag
				if len(updatePayload.Participants) != 1 {
					t.Errorf("Expected 1 participant to update, got %d", len(updatePayload.Participants))
				}

				if len(updatePayload.Participants) > 0 {
					participant := updatePayload.Participants[0]
					if participant.TagNumber != nil {
						t.Errorf("Expected participant to have nil tag number, got %v", *participant.TagNumber)
					}
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
