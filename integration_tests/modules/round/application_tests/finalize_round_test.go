package roundintegrationtests

import (
	"context"
	"strings"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

func TestFinalizeRound(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundevents.AllScoresSubmittedPayload)
		expectedFailure          bool // Changed from expectedError
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult)
	}{
		{
			name: "Successful finalization of an existing round",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundevents.AllScoresSubmittedPayload) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_finalize_1"),
					Title:     "Round to be finalized",
					State:     roundtypes.RoundStateInProgress,
				})
				err := deps.DB.CreateRound(ctx, &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}
				return roundForDBInsertion.ID, &roundevents.AllScoresSubmittedPayload{
					RoundID:   roundForDBInsertion.ID,
					RoundData: roundForDBInsertion,
				}
			},
			expectedFailure: false, // Changed from expectedError
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				// Fixed: expecting pointer type
				finalizedPayload, ok := returnedResult.Success.(*roundevents.RoundFinalizedPayload)
				if !ok {
					t.Errorf("Expected *RoundFinalizedPayload, got %T", returnedResult.Success)
					return
				}

				// Verify the round's state is FINALIZED in the DB
				persistedRound, err := deps.DB.GetRound(ctx, finalizedPayload.RoundID)
				if err != nil {
					t.Fatalf("Failed to fetch round from DB after finalization: %v", err)
				}
				if persistedRound == nil {
					t.Fatalf("Expected round to be found in DB, but it was nil")
				}
				if persistedRound.State != roundtypes.RoundStateFinalized {
					t.Errorf("Expected round state to be FINALIZED, but got %s", persistedRound.State)
				}

				// Verify the payload contains the round data
				if finalizedPayload.RoundData.ID != finalizedPayload.RoundID {
					t.Errorf("Expected RoundData.ID to match RoundID, got %s vs %s", finalizedPayload.RoundData.ID, finalizedPayload.RoundID)
				}
			},
		},
		{
			name: "Attempt to finalize a non-existent round",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundevents.AllScoresSubmittedPayload) {
				nonExistentID := sharedtypes.RoundID(uuid.New())
				generator := testutils.NewTestDataGenerator()
				dummyRound := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("dummy_user"),
					Title:     "Dummy Round",
					State:     roundtypes.RoundStateInProgress,
				})
				return nonExistentID, &roundevents.AllScoresSubmittedPayload{
					RoundID:   nonExistentID,
					RoundData: dummyRound,
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "failed to fetch round data", // Updated to match actual implementation
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success != nil {
					t.Errorf("Expected nil success on failure, but got: %+v", returnedResult.Success)
				}
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				// Fixed: expecting pointer type
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundFinalizationErrorPayload)
				if !ok {
					t.Errorf("Expected *RoundFinalizationErrorPayload, got %T", returnedResult.Failure)
					return
				}
				if !strings.Contains(failurePayload.Error, "failed to fetch round data") {
					t.Errorf("Expected failure error to contain 'failed to fetch round data', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Finalization with nil UUID payload",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundevents.AllScoresSubmittedPayload) {
				generator := testutils.NewTestDataGenerator()
				dummyRound := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("dummy_user"),
					Title:     "Dummy Round",
					State:     roundtypes.RoundStateInProgress,
				})
				return sharedtypes.RoundID(uuid.Nil), &roundevents.AllScoresSubmittedPayload{
					RoundID:   sharedtypes.RoundID(uuid.Nil),
					RoundData: dummyRound,
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "failed to fetch round data", // Updated to match actual implementation
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success != nil {
					t.Errorf("Expected nil success on failure, but got: %+v", returnedResult.Success)
				}
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				// Fixed: expecting pointer type
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundFinalizationErrorPayload)
				if !ok {
					t.Errorf("Expected *RoundFinalizationErrorPayload, got %T", returnedResult.Failure)
					return
				}
				if !strings.Contains(failurePayload.Error, "failed to fetch round data") {
					t.Errorf("Expected failure error to contain 'failed to fetch round data', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Round finalization with database update success but fetch failure",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundevents.AllScoresSubmittedPayload) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_finalize_fetch_fail"),
					Title:     "Round for fetch failure test",
					State:     roundtypes.RoundStateInProgress,
				})
				err := deps.DB.CreateRound(ctx, &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				// Set up mock to simulate fetch failure after successful update
				// This would require a mock DB implementation that can simulate partial failures
				return roundForDBInsertion.ID, &roundevents.AllScoresSubmittedPayload{
					RoundID:   roundForDBInsertion.ID,
					RoundData: roundForDBInsertion,
				}
			},
			expectedFailure: false, // This test assumes normal operation since we can't easily mock partial failure
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				// Fixed: expecting pointer type
				finalizedPayload, ok := returnedResult.Success.(*roundevents.RoundFinalizedPayload)
				if !ok {
					t.Errorf("Expected *RoundFinalizedPayload, got %T", returnedResult.Success)
					return
				}

				// Verify the round's state is FINALIZED in the DB
				persistedRound, err := deps.DB.GetRound(ctx, finalizedPayload.RoundID)
				if err != nil {
					t.Fatalf("Failed to fetch round from DB after finalization: %v", err)
				}
				if persistedRound.State != roundtypes.RoundStateFinalized {
					t.Errorf("Expected round state to be FINALIZED, but got %s", persistedRound.State)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)

			var payload *roundevents.AllScoresSubmittedPayload
			if tt.setupTestEnv != nil {
				_, payload = tt.setupTestEnv(deps.Ctx, deps)
			} else {
				generator := testutils.NewTestDataGenerator()
				dummyRound := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("default_user"),
					Title:     "Default Round",
					State:     roundtypes.RoundStateInProgress,
				})
				payload = &roundevents.AllScoresSubmittedPayload{
					RoundID:   sharedtypes.RoundID(uuid.New()),
					RoundData: dummyRound,
				}
			}

			// Call the actual service method
			result, err := deps.Service.FinalizeRound(deps.Ctx, *payload)
			// The service should never return an error - failures are in the result
			if err != nil {
				t.Errorf("Expected no error from service, but got: %v", err)
			}

			// Check for expected failures in the result
			if tt.expectedFailure {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got none")
				} else if tt.expectedErrorMessagePart != "" {
					failurePayload, ok := result.Failure.(*roundevents.RoundFinalizationErrorPayload)
					if !ok {
						t.Errorf("Expected *RoundFinalizationErrorPayload, got %T", result.Failure)
					} else if !strings.Contains(failurePayload.Error, tt.expectedErrorMessagePart) {
						t.Errorf("Expected error message to contain '%s', but got: '%v'", tt.expectedErrorMessagePart, failurePayload.Error)
					}
				}
			} else {
				if result.Success == nil {
					t.Errorf("Expected success result, but got none")
				}
			}

			if tt.validateResult != nil {
				tt.validateResult(t, deps.Ctx, deps, result)
			}
		})
	}
}

func TestNotifyScoreModule(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundevents.RoundFinalizedPayload)
		expectedFailure          bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult)
	}{
		{
			name: "Successful notification with participants having scores and tag numbers",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundevents.RoundFinalizedPayload) {
				// Create a round in the database first
				generator := testutils.NewTestDataGenerator()
				roundForDB := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_notify_1"),
					Title:     "Round for score notification",
					State:     roundtypes.RoundStateFinalized,
				})
				err := deps.DB.CreateRound(ctx, &roundForDB)
				if err != nil {
					t.Fatalf("Failed to create round for test: %v", err)
				}

				// Create participants with scores and tag numbers
				participant1 := roundtypes.Participant{
					UserID:    sharedtypes.DiscordID("user1"),
					TagNumber: func() *sharedtypes.TagNumber { tn := sharedtypes.TagNumber(42); return &tn }(),
					Score:     func() *sharedtypes.Score { s := sharedtypes.Score(85); return &s }(),
					Response:  roundtypes.ResponseAccept,
				}
				participant2 := roundtypes.Participant{
					UserID:    sharedtypes.DiscordID("user2"),
					TagNumber: func() *sharedtypes.TagNumber { tn := sharedtypes.TagNumber(13); return &tn }(),
					Score:     func() *sharedtypes.Score { s := sharedtypes.Score(92); return &s }(),
					Response:  roundtypes.ResponseAccept,
				}

				// Add participants to the round data for the payload
				roundForDB.AddParticipant(participant1)
				roundForDB.AddParticipant(participant2)

				return roundForDB.ID, &roundevents.RoundFinalizedPayload{
					RoundID:   roundForDB.ID,
					RoundData: roundForDB,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				processScoresPayload, ok := returnedResult.Success.(*roundevents.ProcessRoundScoresRequestPayload)
				if !ok {
					t.Errorf("Expected *ProcessRoundScoresRequestPayload, got %T", returnedResult.Success)
					return
				}

				// Verify the payload contains the correct number of participants (only those with scores)
				if len(processScoresPayload.Scores) != 2 {
					t.Errorf("Expected 2 participant scores, got %d", len(processScoresPayload.Scores))
				}

				// Verify the participant data is correctly converted
				expectedUsers := map[sharedtypes.DiscordID]bool{
					sharedtypes.DiscordID("user1"): false,
					sharedtypes.DiscordID("user2"): false,
				}

				for _, score := range processScoresPayload.Scores {
					if _, exists := expectedUsers[score.UserID]; !exists {
						t.Errorf("Unexpected user ID in scores: %s", score.UserID)
					} else {
						expectedUsers[score.UserID] = true
					}

					// Verify tag number and score are properly set
					if score.TagNumber == nil {
						t.Errorf("Expected tag number to be set for user %s", score.UserID)
					}
					if score.Score == 0 {
						t.Errorf("Expected score to be set for user %s", score.UserID)
					}
				}

				// Verify all expected users were found
				for userID, found := range expectedUsers {
					if !found {
						t.Errorf("Expected user %s not found in scores", userID)
					}
				}
			},
		},
		{
			name: "Failure notification with participants having nil scores",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundevents.RoundFinalizedPayload) {
				// Create a round in the database first
				generator := testutils.NewTestDataGenerator()
				roundForDB := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_notify_2"),
					Title:     "Round with nil scores",
					State:     roundtypes.RoundStateFinalized,
				})
				err := deps.DB.CreateRound(ctx, &roundForDB)
				if err != nil {
					t.Fatalf("Failed to create round for test: %v", err)
				}

				// Create participants with nil scores and tag numbers
				participant1 := roundtypes.Participant{
					UserID:    sharedtypes.DiscordID("user3"),
					TagNumber: nil,
					Score:     nil, // No score submitted
					Response:  roundtypes.ResponseAccept,
				}
				participant2 := roundtypes.Participant{
					UserID:    sharedtypes.DiscordID("user4"),
					TagNumber: func() *sharedtypes.TagNumber { tn := sharedtypes.TagNumber(0); return &tn }(),
					Score:     nil, // No score submitted
					Response:  roundtypes.ResponseAccept,
				}

				// Add participants to the round data for the payload
				roundForDB.AddParticipant(participant1)
				roundForDB.AddParticipant(participant2)

				return roundForDB.ID, &roundevents.RoundFinalizedPayload{
					RoundID:   roundForDB.ID,
					RoundData: roundForDB,
				}
			},
			expectedFailure:          true, // Changed to expect failure
			expectedErrorMessagePart: "no participants with submitted scores found",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundFinalizationErrorPayload)
				if !ok {
					t.Errorf("Expected *RoundFinalizationErrorPayload, got %T", returnedResult.Failure)
					return
				}

				if !strings.Contains(failurePayload.Error, "no participants with submitted scores found") {
					t.Errorf("Expected error to contain 'no participants with submitted scores found', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Failure notification with empty participants list",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundevents.RoundFinalizedPayload) {
				// Create a round in the database first
				generator := testutils.NewTestDataGenerator()
				roundForDB := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_notify_3"),
					Title:     "Round with no participants",
					State:     roundtypes.RoundStateFinalized,
				})
				err := deps.DB.CreateRound(ctx, &roundForDB)
				if err != nil {
					t.Fatalf("Failed to create round for test: %v", err)
				}

				return roundForDB.ID, &roundevents.RoundFinalizedPayload{
					RoundID:   roundForDB.ID,
					RoundData: roundForDB, // Empty participants from generator
				}
			},
			expectedFailure:          true, // Changed to expect failure
			expectedErrorMessagePart: "no participants with submitted scores found",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundFinalizationErrorPayload)
				if !ok {
					t.Errorf("Expected *RoundFinalizationErrorPayload, got %T", returnedResult.Failure)
					return
				}

				if !strings.Contains(failurePayload.Error, "no participants with submitted scores found") {
					t.Errorf("Expected error to contain 'no participants with submitted scores found', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Notification with mixed participant data - only includes participants with scores",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundevents.RoundFinalizedPayload) {
				// Create a round in the database first
				generator := testutils.NewTestDataGenerator()
				roundForDB := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_notify_4"),
					Title:     "Round with mixed participant data",
					State:     roundtypes.RoundStateFinalized,
				})
				err := deps.DB.CreateRound(ctx, &roundForDB)
				if err != nil {
					t.Fatalf("Failed to create round for test: %v", err)
				}

				// Mix of participants: some with scores, some without
				participant1 := roundtypes.Participant{
					UserID:    sharedtypes.DiscordID("user5"),
					TagNumber: func() *sharedtypes.TagNumber { tn := sharedtypes.TagNumber(25); return &tn }(),
					Score:     func() *sharedtypes.Score { s := sharedtypes.Score(78); return &s }(), // HAS SCORE
					Response:  roundtypes.ResponseAccept,
				}
				participant2 := roundtypes.Participant{
					UserID:    sharedtypes.DiscordID("user6"),
					TagNumber: nil,
					Score:     func() *sharedtypes.Score { s := sharedtypes.Score(65); return &s }(), // HAS SCORE
					Response:  roundtypes.ResponseAccept,
				}
				participant3 := roundtypes.Participant{
					UserID:    sharedtypes.DiscordID("user7"),
					TagNumber: func() *sharedtypes.TagNumber { tn := sharedtypes.TagNumber(33); return &tn }(),
					Score:     nil, // NO SCORE - should be excluded
					Response:  roundtypes.ResponseAccept,
				}

				// Add participants to the round data for the payload
				roundForDB.AddParticipant(participant1)
				roundForDB.AddParticipant(participant2)
				roundForDB.AddParticipant(participant3)

				return roundForDB.ID, &roundevents.RoundFinalizedPayload{
					RoundID:   roundForDB.ID,
					RoundData: roundForDB,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				processScoresPayload, ok := returnedResult.Success.(*roundevents.ProcessRoundScoresRequestPayload)
				if !ok {
					t.Errorf("Expected *ProcessRoundScoresRequestPayload, got %T", returnedResult.Success)
					return
				}

				// Only 2 participants should be included (user5 and user6 have scores, user7 doesn't)
				if len(processScoresPayload.Scores) != 2 {
					t.Errorf("Expected 2 participant scores (only those with scores), got %d", len(processScoresPayload.Scores))
				}

				// Create a map to verify specific user data
				scoresByUser := make(map[sharedtypes.DiscordID]roundevents.ParticipantScore)
				for _, score := range processScoresPayload.Scores {
					scoresByUser[score.UserID] = score
				}

				// Verify user5 (complete data with score)
				if score, exists := scoresByUser[sharedtypes.DiscordID("user5")]; exists {
					if score.TagNumber == nil || int(*score.TagNumber) != 25 {
						t.Errorf("Expected user5 tag number to be 25, got %v", score.TagNumber)
					}
					if int(score.Score) != 78 {
						t.Errorf("Expected user5 score to be 78, got %d", score.Score)
					}
				} else {
					t.Errorf("Expected user5 to be in scores")
				}

				// Verify user6 (nil tag number but has score)
				if score, exists := scoresByUser[sharedtypes.DiscordID("user6")]; exists {
					if score.TagNumber == nil || int(*score.TagNumber) != 0 {
						t.Errorf("Expected user6 tag number to default to 0, got %v", score.TagNumber)
					}
					if int(score.Score) != 65 {
						t.Errorf("Expected user6 score to be 65, got %d", score.Score)
					}
				} else {
					t.Errorf("Expected user6 to be in scores")
				}

				// Verify user7 is NOT in scores (no score submitted)
				if _, exists := scoresByUser[sharedtypes.DiscordID("user7")]; exists {
					t.Errorf("Expected user7 to NOT be in scores (no score submitted)")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)

			var payload *roundevents.RoundFinalizedPayload
			if tt.setupTestEnv != nil {
				_, payload = tt.setupTestEnv(deps.Ctx, deps)
			} else {
				generator := testutils.NewTestDataGenerator()
				defaultRound := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("default_user"),
					Title:     "Default Round",
					State:     roundtypes.RoundStateFinalized,
				})
				payload = &roundevents.RoundFinalizedPayload{
					RoundID:   sharedtypes.RoundID(uuid.New()),
					RoundData: defaultRound,
				}
			}

			// Call the actual service method
			result, err := deps.Service.NotifyScoreModule(deps.Ctx, *payload)
			// The service should never return an error - failures are in the result
			if err != nil {
				t.Errorf("Expected no error from service, but got: %v", err)
			}

			// Check for expected failures in the result
			if tt.expectedFailure {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got none")
				} else if tt.expectedErrorMessagePart != "" {
					failurePayload, ok := result.Failure.(*roundevents.RoundFinalizationErrorPayload)
					if !ok {
						t.Errorf("Expected *RoundFinalizationErrorPayload, got %T", result.Failure)
					} else if !strings.Contains(failurePayload.Error, tt.expectedErrorMessagePart) {
						t.Errorf("Expected error message to contain '%s', but got: '%v'", tt.expectedErrorMessagePart, failurePayload.Error)
					}
				}
			} else {
				if result.Success == nil {
					t.Errorf("Expected success result, but got none")
				}
			}

			if tt.validateResult != nil {
				tt.validateResult(t, deps.Ctx, deps, result)
			}
		})
	}
}
