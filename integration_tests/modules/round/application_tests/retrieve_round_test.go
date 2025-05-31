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
	"github.com/google/uuid"
)

// TestGetRound is the main integration test function for the GetRound service method.
func TestGetRound(t *testing.T) {
	nonexistentRoundID := sharedtypes.RoundID(uuid.New())
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) sharedtypes.RoundID
		roundIDToFetch           sharedtypes.RoundID
		expectedError            bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult)
	}{
		{
			name: "Successful retrieval of an existing round",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) sharedtypes.RoundID {
				generator := testutils.NewTestDataGenerator()

				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_creator_123"),
					Title:     "Integration Test Round",
					State:     roundtypes.RoundStateUpcoming,
				})

				// Set the fields that are NOT in RoundOptions directly on the returned 'round' object
				description := roundtypes.Description("This is a test round for GetRound.")
				round.Description = &description

				location := roundtypes.Location("Test Course")
				round.Location = &location

				eventType := roundtypes.EventType("Practice")
				round.EventType = &eventType

				// Ensure start_time is set (required field)
				if round.StartTime == nil || round.StartTime.AsTime().IsZero() {
					startTime := sharedtypes.StartTime(time.Now().Add(24 * time.Hour))
					round.StartTime = &startTime
				}

				tag1 := sharedtypes.TagNumber(1)
				tag2 := sharedtypes.TagNumber(2)
				// Add participants with a mix of tags, no tags, and different responses
				round.Participants = []roundtypes.Participant{
					{
						UserID:    sharedtypes.DiscordID("user_a"),
						TagNumber: &tag1,
						Response:  roundtypes.ResponseAccept,
					},
					{
						UserID:    sharedtypes.DiscordID("user_b"),
						TagNumber: &tag2,
						Response:  roundtypes.ResponseTentative,
					},
					{
						UserID:    sharedtypes.DiscordID("user_c"),
						TagNumber: nil, // Declined user should not have a tag
						Response:  roundtypes.ResponseDecline,
					},
					{
						UserID:    sharedtypes.DiscordID("user_d"),
						TagNumber: nil, // Accepted but no tag assigned yet
						Response:  roundtypes.ResponseAccept,
					},
					{
						UserID:    sharedtypes.DiscordID("user_e"),
						TagNumber: nil, // Tentative, no tag
						Response:  roundtypes.ResponseTentative,
					},
				}

				err := deps.DB.CreateRound(ctx, &round)
				if err != nil {
					t.Fatalf("Failed to create round in DB for test setup: %v", err)
				}
				return round.ID
			},
			// Correctly convert uuid.Nil to sharedtypes.RoundID (which is a string alias)
			roundIDToFetch:           sharedtypes.RoundID(uuid.Nil),
			expectedError:            false,
			expectedErrorMessagePart: "",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}

				retrievedRound, ok := returnedResult.Success.(roundtypes.Round)
				if !ok {
					t.Errorf("Expected result to be of type roundtypes.Round, got %T", returnedResult.Success)
					return
				}

				// Retrieve the original round from the DB to compare
				// Assumed deps.DB.GetUpcomingRounds returns []*roundtypes.Round based on observed compiler error
				rounds, err := deps.DB.GetUpcomingRounds(ctx)
				if err != nil {
					t.Fatalf("Failed to get rounds from DB for validation: %v", err)
				}

				var expectedRound *roundtypes.Round
				for _, r := range rounds {
					if r.ID == retrievedRound.ID {
						expectedRound = r
						break
					}
				}

				if expectedRound == nil {
					t.Fatalf("Could not find the expected round in the database for validation.")
				}

				if retrievedRound.ID != expectedRound.ID {
					t.Errorf("Expected RoundID %s, got %s", expectedRound.ID, retrievedRound.ID)
				}
				if retrievedRound.Title != expectedRound.Title {
					t.Errorf("Expected Title '%s', got '%s'", expectedRound.Title, retrievedRound.Title)
				}

				// Validate Description (pointer comparison and printing with %v)
				if (retrievedRound.Description == nil && expectedRound.Description != nil) ||
					(retrievedRound.Description != nil && expectedRound.Description == nil) ||
					(retrievedRound.Description != nil && expectedRound.Description != nil && *retrievedRound.Description != *expectedRound.Description) {
					t.Errorf("Expected Description %v, got %v", expectedRound.Description, retrievedRound.Description)
				}

				// Validate Location (pointer comparison and printing with %v)
				if (retrievedRound.Location == nil && expectedRound.Location != nil) ||
					(retrievedRound.Location != nil && expectedRound.Location == nil) ||
					(retrievedRound.Location != nil && expectedRound.Location != nil && *retrievedRound.Location != *expectedRound.Location) {
					t.Errorf("Expected Location %v, got %v", expectedRound.Location, retrievedRound.Location)
				}

				// Validate EventType (pointer comparison and printing with %v)
				if (retrievedRound.EventType == nil && expectedRound.EventType != nil) ||
					(retrievedRound.EventType != nil && expectedRound.EventType == nil) ||
					(retrievedRound.EventType != nil && expectedRound.EventType != nil && *retrievedRound.EventType != *expectedRound.EventType) {
					t.Errorf("Expected EventType %v, got %v", expectedRound.EventType, retrievedRound.EventType)
				}

				if retrievedRound.CreatedBy != expectedRound.CreatedBy {
					t.Errorf("Expected CreatedBy '%s', got '%s'", expectedRound.CreatedBy, retrievedRound.CreatedBy)
				}
				if retrievedRound.State != expectedRound.State {
					t.Errorf("Expected State '%s', got '%s'", expectedRound.State, retrievedRound.State)
				}

				// Validate participants
				if len(retrievedRound.Participants) != len(expectedRound.Participants) {
					t.Errorf("Expected %d participants, got %d", len(expectedRound.Participants), len(retrievedRound.Participants))
					return // If counts don't match, further detailed validation might be misleading.
				}

				expectedParticipantsMap := make(map[sharedtypes.DiscordID]roundtypes.Participant)
				for _, p := range expectedRound.Participants {
					expectedParticipantsMap[p.UserID] = p
				}

				for _, retrievedP := range retrievedRound.Participants {
					expectedP, exists := expectedParticipantsMap[retrievedP.UserID]
					if !exists {
						t.Errorf("Unexpected participant with UserID '%s' found in retrieved round", retrievedP.UserID)
						continue
					}

					// Validate TagNumber
					if (retrievedP.TagNumber == nil && expectedP.TagNumber != nil) ||
						(retrievedP.TagNumber != nil && expectedP.TagNumber == nil) ||
						(retrievedP.TagNumber != nil && expectedP.TagNumber != nil && *retrievedP.TagNumber != *expectedP.TagNumber) {
						t.Errorf("Participant '%s': Expected TagNumber %v, got %v", retrievedP.UserID, expectedP.TagNumber, retrievedP.TagNumber)
					}

					// Validate Response
					if retrievedP.Response != expectedP.Response {
						t.Errorf("Participant '%s': Expected Response '%s', got '%s'", retrievedP.UserID, expectedP.Response, retrievedP.Response)
					}

					// Specific rule: Declined participants should have nil TagNumber
					if retrievedP.Response == roundtypes.ResponseDecline {
						if retrievedP.TagNumber != nil {
							t.Errorf("Participant '%s' with Response 'Decline': Expected TagNumber to be nil, but got %v", retrievedP.UserID, *retrievedP.TagNumber)
						}
					}
				}
			},
		},
		{
			name: "Retrieval of a non-existent round",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) sharedtypes.RoundID {
				// No rounds are created for this test, as we want to test fetching a non-existent one.
				return nonexistentRoundID
			},
			roundIDToFetch:           nonexistentRoundID,
			expectedError:            true,
			expectedErrorMessagePart: "failed to retrieve round", // This is the top-level error from the service
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}

				failurePayload, ok := returnedResult.Failure.(roundevents.RoundErrorPayload)
				if !ok {
					t.Fatalf("Expected returnedResult.Failure to be of type roundevents.RoundErrorPayload, got %T", returnedResult.Failure)
				}

				if failurePayload.RoundID != nonexistentRoundID {
					t.Errorf("Expected failure RoundID to be '%s', got '%s'", nonexistentRoundID, failurePayload.RoundID)
				}
				expectedDBErrorMessagePart := "not found"
				if !strings.Contains(failurePayload.Error, expectedDBErrorMessagePart) {
					t.Errorf("Expected failure payload error message to contain '%s', got '%s'", expectedDBErrorMessagePart, failurePayload.Error)
				}
				if !strings.Contains(failurePayload.Error, sharedtypes.RoundID(nonexistentRoundID).String()) {
					t.Errorf("Expected failure payload error message to contain the round ID '%s', got '%s'", nonexistentRoundID, failurePayload.Error)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)

			roundIDToFetch := tt.setupTestEnv(deps.Ctx, deps)
			if tt.roundIDToFetch != sharedtypes.RoundID(uuid.Nil) {
				roundIDToFetch = tt.roundIDToFetch
			}

			result, err := deps.Service.GetRound(deps.Ctx, roundIDToFetch)

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
