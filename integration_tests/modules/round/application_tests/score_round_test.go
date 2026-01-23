package roundintegrationtests

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// InsertRoundHelper creates and inserts a round with the given data using bun.
func InsertRoundHelper(t *testing.T, db *bun.DB, roundData roundtypes.Round) (*roundtypes.Round, error) {
	t.Helper()
	_, err := db.NewInsert().Model(&roundData).Exec(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to insert round: %w", err)
	}
	return &roundData, nil
}

// SetupRoundWithParticipantsHelper generates a round with specified properties and participants
// using testutils.TestDataGenerator and inserts it into the database.
func SetupRoundWithParticipantsHelper(t *testing.T, db *bun.DB, roundID sharedtypes.RoundID, roundTitle roundtypes.Title, eventMessageID string, participantsData []roundtypes.Participant) (*roundtypes.Round, []roundtypes.Participant) {
	t.Helper()
	gen := testutils.NewTestDataGenerator()

	// Prepare users for GenerateRoundWithConstraints
	var usersForGeneration []testutils.User
	for _, pData := range participantsData {
		usersForGeneration = append(usersForGeneration, testutils.User{UserID: testutils.DiscordID(pData.UserID)})
	}

	// Create RoundOptions to pass to the generator
	roundOptions := testutils.RoundOptions{
		ID:               roundID,
		CreatedBy:        testutils.DiscordID("test_creator"), // A dummy creator ID
		ParticipantCount: len(participantsData),
		Users:            usersForGeneration,
		Title:            roundTitle,
		// Other fields like State, StartTime, Finalized will be generated or can be overridden in options
	}

	// Generate the base round with participants
	round := gen.GenerateRoundWithConstraints(roundOptions)

	// --- START OF MODIFICATION ---
	// Explicitly set the EventMessageID from the helper's argument.
	// This ensures the value passed to the helper is used, overriding any default from the generator.
	round.EventMessageID = eventMessageID
	// Set the GuildID to match test expectations
	round.GuildID = sharedtypes.GuildID("test-guild")
	// --- END OF MODIFICATION ---

	// Override participants with the exact data provided for the test case
	round.Participants = make([]roundtypes.Participant, len(participantsData))
	for i, pData := range participantsData {
		round.Participants[i] = roundtypes.Participant{
			UserID:    pData.UserID,
			TagNumber: pData.TagNumber,
			Response:  pData.Response,
			Score:     pData.Score,
		}
	}

	// Insert the round (which includes its participants) into the database
	insertedRound, err := InsertRoundHelper(t, db, round)
	if err != nil {
		t.Fatalf("Failed to insert round during setup: %v", err)
	}

	return insertedRound, insertedRound.Participants
}

// --- Integration Test Functions ---

// TestValidateScoreUpdateRequest tests the score update validation functionality.
func TestValidateScoreUpdateRequest(t *testing.T) {
	tests := []struct {
		name                  string
		payload               roundevents.ScoreUpdateRequestPayloadV1
		expectedError         bool
		expectedErrorContains string
	}{
		{
			name: "Valid score update request",
			payload: roundevents.ScoreUpdateRequestPayloadV1{
				GuildID:   sharedtypes.GuildID("test-guild"),
				RoundID:   sharedtypes.RoundID(uuid.New()),
				UserID:    sharedtypes.DiscordID("123456789"),
				Score:     func() *sharedtypes.Score { s := sharedtypes.Score(72); return &s }(),
				ChannelID: "test-channel",
				MessageID: "test-message",
			},
			expectedError: false,
		},
		{
			name: "Invalid request - zero round ID",
			payload: roundevents.ScoreUpdateRequestPayloadV1{
				GuildID:   sharedtypes.GuildID("test-guild"),
				RoundID:   sharedtypes.RoundID(uuid.Nil),
				UserID:    sharedtypes.DiscordID("123456789"),
				Score:     func() *sharedtypes.Score { s := sharedtypes.Score(72); return &s }(),
				ChannelID: "test-channel",
				MessageID: "test-message",
			},
			expectedError:         false, // Service uses failure payload instead of error
			expectedErrorContains: "round ID cannot be zero",
		},
		{
			name: "Invalid request - empty participant",
			payload: roundevents.ScoreUpdateRequestPayloadV1{
				GuildID:   sharedtypes.GuildID("test-guild"),
				RoundID:   sharedtypes.RoundID(uuid.New()), // Fixed: use valid UUID instead of Nil
				UserID:    sharedtypes.DiscordID(""),
				Score:     func() *sharedtypes.Score { s := sharedtypes.Score(72); return &s }(),
				ChannelID: "test-channel",
				MessageID: "test-message",
			},
			expectedError:         false, // Service uses failure payload instead of error
			expectedErrorContains: "participant Discord ID cannot be empty",
		},
		{
			name: "Invalid request - nil score",
			payload: roundevents.ScoreUpdateRequestPayloadV1{
				GuildID:   sharedtypes.GuildID("test-guild"),
				RoundID:   sharedtypes.RoundID(uuid.New()),
				UserID:    sharedtypes.DiscordID("123456789"),
				Score:     nil,
				ChannelID: "test-channel",
				MessageID: "test-message",
			},
			expectedError:         false, // Service uses failure payload instead of error
			expectedErrorContains: "score cannot be empty",
		},
		{
			name: "Invalid request - multiple validation errors",
			payload: roundevents.ScoreUpdateRequestPayloadV1{
				GuildID:   sharedtypes.GuildID("test-guild"),
				RoundID:   sharedtypes.RoundID(uuid.Nil),
				UserID:    sharedtypes.DiscordID(""),
				Score:     nil,
				ChannelID: "test-channel",
				MessageID: "test-message",
			},
			expectedError:         false, // Service uses failure payload instead of error
			expectedErrorContains: "round ID cannot be zero; participant Discord ID cannot be empty; score cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup dependencies for this test function
			deps := SetupTestRoundService(t)
			// No defer deps.Cleanup() here, as per your request. Cleanup is external.

			// Service is now part of deps
			result, err := deps.Service.ValidateScoreUpdateRequest(deps.Ctx, tt.payload)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				}
				if result.Failure == nil {
					t.Errorf("Expected a failure payload, but got nil")
				}
				if result.Success != nil {
					t.Errorf("Expected nil success payload, but got %v", result.Success)
				}

				failurePayload, ok := result.Failure.(*roundevents.RoundScoreUpdateErrorPayloadV1)
				if !ok {
					t.Errorf("Expected *RoundScoreUpdateErrorPayload, got %T", result.Failure)
				}
				if !strings.Contains(failurePayload.Error, tt.expectedErrorContains) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedErrorContains, failurePayload.Error)
				}
				if failurePayload.ScoreUpdateRequest == nil {
					t.Errorf("Expected ScoreUpdateRequest in failure payload, but got nil")
				} else if *failurePayload.ScoreUpdateRequest != tt.payload {
					t.Errorf("Expected ScoreUpdateRequest in failure payload to be %v, got %v", tt.payload, *failurePayload.ScoreUpdateRequest)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}

				// Handle both success and failure cases when expectedError is false
				if result.Failure != nil && result.Success != nil {
					t.Errorf("Got both failure and success payloads - should only have one")
				}
				if result.Failure == nil && result.Success == nil {
					t.Errorf("Expected either a success or failure payload, but got neither")
				}

				// Check for validation failures when expectedError is false but validation fails
				if result.Failure != nil && tt.expectedErrorContains != "" {
					failurePayload, ok := result.Failure.(*roundevents.RoundScoreUpdateErrorPayloadV1)
					if !ok {
						t.Errorf("Expected *RoundScoreUpdateErrorPayload, got %T", result.Failure)
					}
					if !strings.Contains(failurePayload.Error, tt.expectedErrorContains) {
						t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedErrorContains, failurePayload.Error)
					}
					if failurePayload.ScoreUpdateRequest == nil {
						t.Errorf("Expected ScoreUpdateRequest in failure payload, but got nil")
					} else if *failurePayload.ScoreUpdateRequest != tt.payload {
						t.Errorf("Expected ScoreUpdateRequest in failure payload to be %v, got %v", tt.payload, *failurePayload.ScoreUpdateRequest)
					}
				}

				// Check for success when validation passes
				if result.Success != nil {
					successPayload, ok := result.Success.(*roundevents.ScoreUpdateValidatedPayloadV1)
					if !ok {
						t.Errorf("Expected *ScoreUpdateValidatedPayload pointer, got %T", result.Success)
					}
					if successPayload.ScoreUpdateRequestPayload != tt.payload {
						t.Errorf("Expected ScoreUpdateRequestPayload to be %v, got %v", tt.payload, successPayload.ScoreUpdateRequestPayload)
					}
				}
			}
		})
	}
}

func TestUpdateParticipantScore(t *testing.T) {
	score72 := sharedtypes.Score(72)
	tag1 := sharedtypes.TagNumber(1)

	tests := []struct {
		name             string
		roundID          sharedtypes.RoundID
		initialSetup     func(t *testing.T, db *bun.DB, roundID sharedtypes.RoundID)
		payload          roundevents.ScoreUpdateValidatedPayloadV1
		expectedError    bool
		validateResponse func(t *testing.T, result results.OperationResult, roundID sharedtypes.RoundID)
	}{
		{
			name:    "Successful score update",
			roundID: sharedtypes.RoundID(uuid.New()),
			initialSetup: func(t *testing.T, db *bun.DB, roundID sharedtypes.RoundID) {
				_ = SetupRoundWithParticipantsAndGroupsHelper(t, db, roundID,
					roundtypes.Title("Test Round"), "msg123",
					[]roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("123456789"), TagNumber: &tag1, Response: roundtypes.ResponseAccept, Score: nil},
					})
			},
			payload: roundevents.ScoreUpdateValidatedPayloadV1{
				GuildID: sharedtypes.GuildID("test-guild"),
				ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayloadV1{
					GuildID:   sharedtypes.GuildID("test-guild"),
					RoundID:   sharedtypes.RoundID(uuid.Nil), // replaced in loop
					UserID:    sharedtypes.DiscordID("123456789"),
					Score:     &score72,
					ChannelID: "test-channel",
					MessageID: "test-message",
				},
			},
			expectedError: false,
			validateResponse: func(t *testing.T, result results.OperationResult, roundID sharedtypes.RoundID) {
				if result.Success == nil {
					t.Fatalf("Expected success payload, got failure instead: %+v", result.Failure)
				}
				successPayload, ok := result.Success.(*roundevents.ParticipantScoreUpdatedPayloadV1)
				if !ok {
					t.Fatalf("Expected ParticipantScoreUpdatedPayload pointer, got %T", result.Success)
				}
				if successPayload.UserID != sharedtypes.DiscordID("123456789") {
					t.Errorf("Expected participant '123456789', got '%s'", successPayload.UserID)
				}
				if successPayload.Score != score72 {
					t.Errorf("Expected score 72, got %d", successPayload.Score)
				}
				if successPayload.EventMessageID != "msg123" {
					t.Errorf("Expected EventMessageID 'msg123', got '%s'", successPayload.EventMessageID)
				}
				if len(successPayload.Participants) != 1 {
					t.Errorf("Expected 1 participant, got %d", len(successPayload.Participants))
				}
				if successPayload.Participants[0].UserID != sharedtypes.DiscordID("123456789") {
					t.Errorf("Expected participant in list '123456789', got '%s'", successPayload.Participants[0].UserID)
				}
				if successPayload.Participants[0].Score == nil || *successPayload.Participants[0].Score != score72 {
					t.Errorf("Expected participant score in list 72, got %v", successPayload.Participants[0].Score)
				}
			},
		},
		{
			name:    "Database update failure (non-existent round)",
			roundID: sharedtypes.RoundID(uuid.New()),
			initialSetup: func(t *testing.T, db *bun.DB, roundID sharedtypes.RoundID) {
				// No setup: this round doesn't exist
			},
			payload: roundevents.ScoreUpdateValidatedPayloadV1{
				GuildID: sharedtypes.GuildID("test-guild"),
				ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayloadV1{
					GuildID:   sharedtypes.GuildID("test-guild"),
					RoundID:   sharedtypes.RoundID(uuid.Nil),
					UserID:    sharedtypes.DiscordID("nonexistent"),
					Score:     &score72,
					ChannelID: "test-channel",
					MessageID: "test-message",
				},
			},
			expectedError: false,
			validateResponse: func(t *testing.T, result results.OperationResult, roundID sharedtypes.RoundID) {
				if result.Failure == nil {
					t.Fatalf("Expected failure payload, but got nil")
				}
				failurePayload, ok := result.Failure.(*roundevents.RoundScoreUpdateErrorPayloadV1)
				if !ok {
					t.Fatalf("Expected *RoundScoreUpdateErrorPayload, got %T", result.Failure)
				}
				if !strings.Contains(failurePayload.Error, "Failed to update score in database") {
					t.Errorf("Expected error message to contain 'Failed to update score in database', got '%s'", failurePayload.Error)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			tt.roundID = sharedtypes.RoundID(uuid.New())
			tt.payload.ScoreUpdateRequestPayload.RoundID = tt.roundID

			if tt.initialSetup != nil {
				tt.initialSetup(t, deps.BunDB, tt.roundID)
			}

			result, err := deps.Service.UpdateParticipantScore(deps.Ctx, tt.payload)
			if tt.expectedError && err == nil {
				t.Errorf("Expected error, got none")
			}

			if tt.validateResponse != nil {
				tt.validateResponse(t, result, tt.roundID)
			}
		})
	}
}

func SetupRoundWithParticipantsAndGroupsHelper(
	t *testing.T,
	db *bun.DB,
	roundID sharedtypes.RoundID,
	title roundtypes.Title,
	messageID string,
	participants []roundtypes.Participant,
) *roundtypes.Round {
	t.Helper()

	round := &roundtypes.Round{
		ID:             roundID,
		Title:          title,
		Description:    "Test Description",
		Location:       "Test Location",
		StartTime:      (*sharedtypes.StartTime)(ptrToTime(time.Now().Add(time.Hour))),
		CreatedBy:      sharedtypes.DiscordID("test-user"),
		State:          roundtypes.RoundStateUpcoming,
		Participants:   participants,
		GuildID:        sharedtypes.GuildID("test-guild"),
		EventMessageID: messageID,
	}

	// Insert round
	_, err := db.NewInsert().
		Model(round).
		Exec(context.Background())
	require.NoError(t, err, "failed to insert round")

	// Materialize groups
	repo := rounddb.NewRepository(db) // Assuming your repo constructor
	err = repo.CreateRoundGroups(context.Background(), round.ID, participants)
	require.NoError(t, err, "failed to create round groups")

	return round
}

func ptrToTime(t time.Time) *time.Time {
	return &t
}
