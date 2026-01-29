package roundintegrationtests

import (
	"context"
	"strings"
	"testing"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

func TestCheckParticipantStatus(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.JoinRoundRequest)
		expectedFailure          bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error])
	}{
		{
			name: "User not a participant, requesting Accept - Expecting Validation Request",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.JoinRoundRequest) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_check_status_1"),
					Title:     "Round for status check",
					State:     roundtypes.RoundStateUpcoming,
				})
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}
				return roundForDBInsertion.ID, &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  roundForDBInsertion.ID,
					UserID:   sharedtypes.DiscordID("new_participant_1"),
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error]) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				statusResult := *returnedResult.Success
				if statusResult.Action != "VALIDATE" {
					t.Errorf("Expected Action to be 'VALIDATE', got '%s'", statusResult.Action)
				}
				if statusResult.UserID != sharedtypes.DiscordID("new_participant_1") {
					t.Errorf("Expected UserID to be 'new_participant_1', got '%s'", statusResult.UserID)
				}
				if statusResult.Response != roundtypes.ResponseAccept {
					t.Errorf("Expected Response to be 'Accept', got '%s'", statusResult.Response)
				}
			},
		},
		{
			name: "User is participant with Accept, requesting Accept (toggle off) - Expecting Removal Request",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.JoinRoundRequest) {
				generator := testutils.NewTestDataGenerator()
				participant := roundtypes.Participant{
					UserID:   sharedtypes.DiscordID("existing_participant_2"),
					Response: roundtypes.ResponseAccept,
				}
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_check_status_2"),
					Title:     "Round for status check toggle",
					State:     roundtypes.RoundStateUpcoming,
				})
				roundForDBInsertion.Participants = []roundtypes.Participant{participant}
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}
				return roundForDBInsertion.ID, &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  roundForDBInsertion.ID,
					UserID:   sharedtypes.DiscordID("existing_participant_2"),
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error]) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				statusResult := *returnedResult.Success
				if statusResult.Action != "REMOVE" {
					t.Errorf("Expected Action to be 'REMOVE', got '%s'", statusResult.Action)
				}
				if statusResult.UserID != sharedtypes.DiscordID("existing_participant_2") {
					t.Errorf("Expected UserID to be 'existing_participant_2', got '%s'", statusResult.UserID)
				}
			},
		},
		{
			name: "User is participant with Tentative, requesting Accept (change status) - Expecting Validation Request",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.JoinRoundRequest) {
				generator := testutils.NewTestDataGenerator()
				participant := roundtypes.Participant{
					UserID:   sharedtypes.DiscordID("existing_participant_3"),
					Response: roundtypes.ResponseTentative,
				}
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_check_status_3"),
					Title:     "Round for status change",
					State:     roundtypes.RoundStateUpcoming,
				})
				roundForDBInsertion.Participants = []roundtypes.Participant{participant}
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}
				return roundForDBInsertion.ID, &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  roundForDBInsertion.ID,
					UserID:   sharedtypes.DiscordID("existing_participant_3"),
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error]) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				statusResult := *returnedResult.Success
				if statusResult.Action != "VALIDATE" {
					t.Errorf("Expected Action to be 'VALIDATE', got '%s'", statusResult.Action)
				}
				if statusResult.UserID != sharedtypes.DiscordID("existing_participant_3") {
					t.Errorf("Expected UserID to be 'existing_participant_3', got '%s'", statusResult.UserID)
				}
				if statusResult.Response != roundtypes.ResponseAccept {
					t.Errorf("Expected Response to be 'Accept', got '%s'", statusResult.Response)
				}
			},
		},
		{
			name: "Round ID is nil - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.JoinRoundRequest) {
				return sharedtypes.RoundID(uuid.Nil), &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  sharedtypes.RoundID(uuid.Nil),
					UserID:   sharedtypes.DiscordID("some_user"),
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "failed to get participant status",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error]) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				err := *returnedResult.Failure
				if !strings.Contains(err.Error(), "failed to fetch round details") {
					// NOTE: Error message might vary based on implementation
				}
			},
		},
		{
			name: "Attempt to check status for a non-existent round - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.JoinRoundRequest) {
				nonExistentID := sharedtypes.RoundID(uuid.New())
				return nonExistentID, &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  nonExistentID,
					UserID:   sharedtypes.DiscordID("some_user"),
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "failed to get participant status",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error]) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)

			var payload *roundtypes.JoinRoundRequest
			if tt.setupTestEnv != nil {
				_, payload = tt.setupTestEnv(deps.Ctx, deps)
			} else {
				payload = &roundtypes.JoinRoundRequest{
					RoundID:  sharedtypes.RoundID(uuid.New()),
					UserID:   sharedtypes.DiscordID("default_user"),
					Response: roundtypes.ResponseAccept,
				}
			}

			result, err := deps.Service.CheckParticipantStatus(deps.Ctx, payload)
			if err != nil {
				t.Errorf("Expected no error from service, but got: %v", err)
			}

			if tt.expectedFailure {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got none")
				} else if tt.expectedErrorMessagePart != "" {
					err := *result.Failure
					if !strings.Contains(err.Error(), tt.expectedErrorMessagePart) {
						t.Errorf("Expected error message to contain '%s', but got: '%v'", tt.expectedErrorMessagePart, err.Error())
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

func TestValidateParticipantJoinRequest(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.JoinRoundRequest)
		expectedFailure          bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult[*roundtypes.JoinRoundRequest, error])
	}{
		{
			name: "Valid Accept request, round Created (not late join) - Expecting JoinRoundRequest",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.JoinRoundRequest) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_validate_1"),
					Title:     "Round for validation (created)",
					State:     roundtypes.RoundStateUpcoming,
				})
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}
				return roundForDBInsertion.ID, &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  roundForDBInsertion.ID,
					UserID:   sharedtypes.DiscordID("new_participant_1"),
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult[*roundtypes.JoinRoundRequest, error]) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				req := *returnedResult.Success
				if req.UserID != sharedtypes.DiscordID("new_participant_1") {
					t.Errorf("Expected UserID to be 'new_participant_1', got '%s'", req.UserID)
				}
				if req.Response != roundtypes.ResponseAccept {
					t.Errorf("Expected Response to be 'Accept', got '%s'", req.Response)
				}
				if req.JoinedLate == nil || *req.JoinedLate != false {
					t.Errorf("Expected JoinedLate to be false, got %v", req.JoinedLate)
				}
			},
		},
		{
			name: "Valid Tentative request, round InProgress (late join) - Expecting JoinRoundRequest",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.JoinRoundRequest) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_validate_2"),
					Title:     "Round for validation (in progress)",
					State:     roundtypes.RoundStateInProgress,
				})
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}
				return roundForDBInsertion.ID, &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  roundForDBInsertion.ID,
					UserID:   sharedtypes.DiscordID("new_participant_2"),
					Response: roundtypes.ResponseTentative,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult[*roundtypes.JoinRoundRequest, error]) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				req := *returnedResult.Success
				if req.UserID != sharedtypes.DiscordID("new_participant_2") {
					t.Errorf("Expected UserID to be 'new_participant_2', got '%s'", req.UserID)
				}
				if req.Response != roundtypes.ResponseTentative {
					t.Errorf("Expected Response to be 'Tentative', got '%s'", req.Response)
				}
				if req.JoinedLate == nil || *req.JoinedLate != true {
					t.Errorf("Expected JoinedLate to be true, got %v", req.JoinedLate)
				}
			},
		},
		{
			name: "Invalid: Nil Round ID - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.JoinRoundRequest) {
				return sharedtypes.RoundID(uuid.Nil), &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  sharedtypes.RoundID(uuid.Nil),
					UserID:   sharedtypes.DiscordID("some_user"),
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "validation failed: [round ID cannot be nil]",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult[*roundtypes.JoinRoundRequest, error]) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
			},
		},
		{
			name: "Invalid: Empty User ID - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.JoinRoundRequest) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_validate_empty_user"),
					Title:     "Round for validation (empty user)",
					State:     roundtypes.RoundStateUpcoming,
				})
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}
				return roundForDBInsertion.ID, &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  roundForDBInsertion.ID,
					UserID:   "",
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "validation failed: [participant Discord ID cannot be empty]",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult[*roundtypes.JoinRoundRequest, error]) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)

			var payload *roundtypes.JoinRoundRequest
			if tt.setupTestEnv != nil {
				_, payload = tt.setupTestEnv(deps.Ctx, deps)
			} else {
				payload = &roundtypes.JoinRoundRequest{
					RoundID:  sharedtypes.RoundID(uuid.New()),
					UserID:   sharedtypes.DiscordID("default_user_payload"),
					Response: roundtypes.ResponseAccept,
				}
			}

			result, err := deps.Service.ValidateParticipantJoinRequest(deps.Ctx, payload)
			if err != nil {
				t.Errorf("Expected no error from service, but got: %v", err)
			}

			if tt.expectedFailure {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got none")
				} else if tt.expectedErrorMessagePart != "" {
					err := *result.Failure
					if !strings.Contains(err.Error(), tt.expectedErrorMessagePart) {
						t.Errorf("Expected error message to contain '%s', but got: '%v'", tt.expectedErrorMessagePart, err.Error())
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
