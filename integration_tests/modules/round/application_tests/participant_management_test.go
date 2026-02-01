package roundintegrationtests

import (
	"context"
	"strings"
	"testing"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

func TestJoinRound(t *testing.T) {
	tests := []struct {
		name            string
		setupTestEnv    func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest
		expectedFailure bool
		validateResult  func(t *testing.T, ctx context.Context, deps RoundTestDeps, res *roundtypes.Round, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID)
	}{
		{
			name: "Successfully join a round",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest {
				generator := testutils.NewTestDataGenerator()
				roundID := sharedtypes.RoundID(uuid.New())
				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					ID:    roundID,
					Title: "Round to join",
					State: roundtypes.RoundStateUpcoming,
				})
				round.GuildID = "test-guild"
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create round: %v", err)
				}

				return &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  roundID,
					UserID:   "new-joiner",
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, res *roundtypes.Round, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) {
				if res == nil {
					t.Fatalf("Expected success round, got nil")
				}
				if res.ID != roundID {
					t.Errorf("Expected round ID %s, got %s", roundID, res.ID)
				}
				found := false
				for _, p := range res.Participants {
					if p.UserID == userID {
						found = true
						if p.Response != roundtypes.ResponseAccept {
							t.Errorf("Expected response %s, got %s", roundtypes.ResponseAccept, p.Response)
						}
					}
				}
				if !found {
					t.Errorf("User %s not found in participants", userID)
				}
			},
		},
		{
			name: "Join a non-existent round - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest {
				return &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  sharedtypes.RoundID(uuid.New()),
					UserID:   "some-user",
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			req := tt.setupTestEnv(deps.Ctx, deps)

			result, err := deps.Service.JoinRound(deps.Ctx, req)
			if err != nil {
				t.Fatalf("Unexpected error from service: %v", err)
			}

			if tt.expectedFailure {
				if result.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
			} else {
				if result.Success == nil {
					t.Fatalf("Expected success result, but got failure: %+v", result.Failure)
				}
				if tt.validateResult != nil {
					tt.validateResult(t, deps.Ctx, deps, *result.Success, req.RoundID, req.UserID)
				}
			}
		})
	}
}

func TestUpdateParticipantStatus(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest
		expectedFailure          bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, res *roundtypes.Round, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID)
	}{
		{
			name: "Successfully update participant status",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest {
				generator := testutils.NewTestDataGenerator()
				roundID := sharedtypes.RoundID(uuid.New())
				userID := sharedtypes.DiscordID("status-updater")
				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					ID:    roundID,
					Title: "Round for status update",
					State: roundtypes.RoundStateUpcoming,
				})
				round.Participants = []roundtypes.Participant{
					{UserID: userID, Response: roundtypes.ResponseTentative},
				}
				round.GuildID = "test-guild"
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create round: %v", err)
				}

				return &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  roundID,
					UserID:   userID,
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, res *roundtypes.Round, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) {
				if res == nil {
					t.Fatalf("Expected success round, got nil")
				}
				found := false
				for _, p := range res.Participants {
					if p.UserID == userID {
						found = true
						if p.Response != roundtypes.ResponseAccept {
							t.Errorf("Expected updated response %s, got %s", roundtypes.ResponseAccept, p.Response)
						}
					}
				}
				if !found {
					t.Errorf("User %s not found in participants", userID)
				}
			},
		},
		{
			name: "Status and tag number set after lookup",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest {
				generator := testutils.NewTestDataGenerator()
				roundID := sharedtypes.RoundID(uuid.New())
				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					ID:    roundID,
					Title: "Round for direct update",
					State: roundtypes.RoundStateUpcoming,
				})
				tagNum1 := sharedtypes.TagNumber(1)
				round.Participants = []roundtypes.Participant{
					{
						UserID:    sharedtypes.DiscordID("existing_participant"),
						TagNumber: &tagNum1,
						Response:  roundtypes.ResponseTentative,
					},
				}
				round.GuildID = "test-guild"
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				tag := sharedtypes.TagNumber(123)
				joinedLate := false
				return &roundtypes.JoinRoundRequest{
					GuildID:    "test-guild",
					RoundID:    roundID,
					UserID:     sharedtypes.DiscordID("existing_participant"),
					Response:   roundtypes.ResponseAccept,
					TagNumber:  &tag,
					JoinedLate: &joinedLate,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, res *roundtypes.Round, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) {
				if res == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				found := false
				for _, p := range res.Participants {
					if p.UserID == sharedtypes.DiscordID("existing_participant") {
						found = true
						if p.TagNumber == nil || *p.TagNumber != 123 {
							t.Errorf("Expected TagNumber to be 123, got %v", p.TagNumber)
						}
						if p.Response != roundtypes.ResponseAccept {
							t.Errorf("Expected Response to be Accept, got %s", p.Response)
						}
						break
					}
				}
				if !found {
					t.Errorf("Participant 'existing_participant' not found in returned round")
				}
			},
		},
		{
			name: "Participant accepts without TagNumber (adds participant with nil tag)",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest {
				generator := testutils.NewTestDataGenerator()
				roundID := sharedtypes.RoundID(uuid.New())
				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					ID:    roundID,
					Title: "Round for nil tag participant",
					State: roundtypes.RoundStateInProgress,
				})
				round.GuildID = "test-guild"
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				joinedLate := false
				return &roundtypes.JoinRoundRequest{
					GuildID:    "test-guild",
					RoundID:    roundID,
					UserID:     sharedtypes.DiscordID("participant_needs_tag"),
					Response:   roundtypes.ResponseAccept,
					TagNumber:  nil,
					JoinedLate: &joinedLate,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, res *roundtypes.Round, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) {
				if res == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				found := false
				for _, p := range res.Participants {
					if p.UserID == sharedtypes.DiscordID("participant_needs_tag") {
						found = true
						if p.TagNumber != nil {
							t.Errorf("Expected TagNumber to be nil, got %v", p.TagNumber)
						}
						break
					}
				}
				if !found {
					t.Errorf("Participant 'participant_needs_tag' not found")
				}
			},
		},
		{
			name: "Attempt to update participant in non-existent round - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest {
				nonExistentID := sharedtypes.RoundID(uuid.New())
				return &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  nonExistentID,
					UserID:   sharedtypes.DiscordID("some_user"),
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "failed to update participant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			req := tt.setupTestEnv(deps.Ctx, deps)

			result, err := deps.Service.UpdateParticipantStatus(deps.Ctx, req)
			if err != nil {
				t.Fatalf("Unexpected error from service: %v", err)
			}

			if tt.expectedFailure {
				if result.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				} else if tt.expectedErrorMessagePart != "" {
					errMsg := (*result.Failure).Error()
					if !strings.Contains(errMsg, tt.expectedErrorMessagePart) {
						t.Errorf("Expected error to contain %q, got %q", tt.expectedErrorMessagePart, errMsg)
					}
				}
			} else {
				if result.Success == nil {
					t.Fatalf("Expected success result, but got failure: %+v", result.Failure)
				}
				if tt.validateResult != nil {
					tt.validateResult(t, deps.Ctx, deps, *result.Success, req.RoundID, req.UserID)
				}
			}
		})
	}
}

func TestParticipantRemoval(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest
		expectedFailure          bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, res *roundtypes.Round, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID)
	}{
		{
			name: "Successfully remove participant",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest {
				generator := testutils.NewTestDataGenerator()
				roundID := sharedtypes.RoundID(uuid.New())
				userID := sharedtypes.DiscordID("to-be-removed")
				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					ID:    roundID,
					Title: "Round for removal",
					State: roundtypes.RoundStateUpcoming,
				})
				round.Participants = []roundtypes.Participant{
					{UserID: userID, Response: roundtypes.ResponseAccept},
					{UserID: "stayer", Response: roundtypes.ResponseAccept},
				}
				round.GuildID = "test-guild"
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create round: %v", err)
				}

				return &roundtypes.JoinRoundRequest{
					GuildID: "test-guild",
					RoundID: roundID,
					UserID:  userID,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, res *roundtypes.Round, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) {
				if res == nil {
					t.Fatalf("Expected success round, got nil")
				}
				found := false
				foundStayer := false
				for _, p := range res.Participants {
					if p.UserID == userID {
						found = true
					}
					if p.UserID == "stayer" {
						foundStayer = true
					}
				}
				if found {
					t.Errorf("User %s should have been removed from participants", userID)
				}
				if !foundStayer {
					t.Errorf("Stayer should still be in participants")
				}
			},
		},
		{
			name: "Attempt to remove non-existent participant - Should succeed (no-op)",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest {
				generator := testutils.NewTestDataGenerator()
				roundID := sharedtypes.RoundID(uuid.New())
				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					ID:    roundID,
					Title: "Round for no-op removal",
					State: roundtypes.RoundStateUpcoming,
				})
				round.Participants = []roundtypes.Participant{
					{UserID: "existing", Response: roundtypes.ResponseAccept},
				}
				round.GuildID = "test-guild"
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create round: %v", err)
				}

				return &roundtypes.JoinRoundRequest{
					GuildID: "test-guild",
					RoundID: roundID,
					UserID:  "non-existent",
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, res *roundtypes.Round, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) {
				if res == nil {
					t.Fatalf("Expected success round, got nil")
				}
				if len(res.Participants) != 1 {
					t.Errorf("Expected 1 participant, got %d", len(res.Participants))
				}
			},
		},
		{
			name: "Attempt to remove participant from non-existent round - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest {
				nonExistentID := sharedtypes.RoundID(uuid.New())
				return &roundtypes.JoinRoundRequest{
					GuildID: "test-guild",
					RoundID: nonExistentID,
					UserID:  "some-user",
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "failed to remove participant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			req := tt.setupTestEnv(deps.Ctx, deps)

			result, err := deps.Service.ParticipantRemoval(deps.Ctx, req)
			if err != nil {
				t.Fatalf("Unexpected error from service: %v", err)
			}

			if tt.expectedFailure {
				if result.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				} else if tt.expectedErrorMessagePart != "" {
					errMsg := (*result.Failure).Error()
					if !strings.Contains(errMsg, tt.expectedErrorMessagePart) {
						t.Errorf("Expected error to contain %q, got %q", tt.expectedErrorMessagePart, errMsg)
					}
				}
			} else {
				if result.Success == nil {
					t.Fatalf("Expected success result, but got failure: %+v", result.Failure)
				}
				if tt.validateResult != nil {
					tt.validateResult(t, deps.Ctx, deps, *result.Success, req.RoundID, req.UserID)
				}
			}
		})
	}
}

func TestCheckParticipantStatus(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest
		expectedFailure          bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, res *roundtypes.ParticipantStatusCheckResult)
	}{
		{
			name: "User not a participant, requesting Accept - Expecting VALIDATE",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest {
				generator := testutils.NewTestDataGenerator()
				roundID := sharedtypes.RoundID(uuid.New())
				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					ID:    roundID,
					Title: "Round for status check",
					State: roundtypes.RoundStateUpcoming,
				})
				round.GuildID = "test-guild"
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create initial round: %v", err)
				}
				return &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  roundID,
					UserID:   "new-participant",
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, res *roundtypes.ParticipantStatusCheckResult) {
				if res.Action != "VALIDATE" {
					t.Errorf("Expected Action 'VALIDATE', got '%s'", res.Action)
				}
				if res.UserID != "new-participant" {
					t.Errorf("Expected UserID 'new-participant', got '%s'", res.UserID)
				}
			},
		},
		{
			name: "User is participant with Accept, requesting Accept - Expecting REMOVE",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest {
				generator := testutils.NewTestDataGenerator()
				roundID := sharedtypes.RoundID(uuid.New())
				userID := sharedtypes.DiscordID("existing-participant")
				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					ID:    roundID,
					State: roundtypes.RoundStateUpcoming,
				})
				round.Participants = []roundtypes.Participant{{UserID: userID, Response: roundtypes.ResponseAccept}}
				round.GuildID = "test-guild"
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create initial round: %v", err)
				}
				return &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  roundID,
					UserID:   userID,
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, res *roundtypes.ParticipantStatusCheckResult) {
				if res.Action != "REMOVE" {
					t.Errorf("Expected Action 'REMOVE', got '%s'", res.Action)
				}
			},
		},
		{
			name: "User is participant with Tentative, requesting Accept - Expecting VALIDATE",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest {
				generator := testutils.NewTestDataGenerator()
				roundID := sharedtypes.RoundID(uuid.New())
				userID := sharedtypes.DiscordID("tentative-participant")
				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					ID:    roundID,
					State: roundtypes.RoundStateUpcoming,
				})
				round.Participants = []roundtypes.Participant{{UserID: userID, Response: roundtypes.ResponseTentative}}
				round.GuildID = "test-guild"
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create initial round: %v", err)
				}
				return &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  roundID,
					UserID:   userID,
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, res *roundtypes.ParticipantStatusCheckResult) {
				if res.Action != "VALIDATE" {
					t.Errorf("Expected Action 'VALIDATE', got '%s'", res.Action)
				}
			},
		},
		{
			name: "Round ID is nil - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest {
				return &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  sharedtypes.RoundID(uuid.Nil),
					UserID:   "some-user",
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "failed to get participant status",
		},
		{
			name: "Attempt to check status for a non-existent round - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest {
				return &roundtypes.JoinRoundRequest{
					GuildID: "test-guild",
					RoundID: sharedtypes.RoundID(uuid.New()),
					UserID:  "some-user",
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "failed to get participant status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			req := tt.setupTestEnv(deps.Ctx, deps)

			result, err := deps.Service.CheckParticipantStatus(deps.Ctx, req)
			if err != nil {
				t.Fatalf("Unexpected error from service: %v", err)
			}

			if tt.expectedFailure {
				if result.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				} else if tt.expectedErrorMessagePart != "" {
					errMsg := (*result.Failure).Error()
					if !strings.Contains(errMsg, tt.expectedErrorMessagePart) {
						t.Errorf("Expected error message to contain '%s', but got: '%v'", tt.expectedErrorMessagePart, errMsg)
					}
				}
			} else {
				if result.Success == nil {
					t.Fatalf("Expected success result, but got failure: %+v", result.Failure)
				}
				if tt.validateResult != nil {
					tt.validateResult(t, deps.Ctx, deps, *result.Success)
				}
			}
		})
	}
}

func TestValidateParticipantJoinRequest(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest
		expectedFailure          bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, res *roundtypes.JoinRoundRequest)
	}{
		{
			name: "Valid Accept request, round Created (not late join) - Expecting JoinRoundRequest",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest {
				generator := testutils.NewTestDataGenerator()
				roundID := sharedtypes.RoundID(uuid.New())
				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					ID:    roundID,
					Title: "Upcoming Round",
					State: roundtypes.RoundStateUpcoming,
				})
				round.GuildID = "test-guild"
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create initial round: %v", err)
				}
				return &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  roundID,
					UserID:   "new-joiner",
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, res *roundtypes.JoinRoundRequest) {
				if res.JoinedLate == nil || *res.JoinedLate != false {
					t.Errorf("Expected JoinedLate to be false, got %v", res.JoinedLate)
				}
			},
		},
		{
			name: "Valid Tentative request, round InProgress (late join) - Expecting JoinRoundRequest",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest {
				generator := testutils.NewTestDataGenerator()
				roundID := sharedtypes.RoundID(uuid.New())
				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					ID:    roundID,
					Title: "In Progress Round",
					State: roundtypes.RoundStateInProgress,
				})
				round.GuildID = "test-guild"
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create initial round: %v", err)
				}
				return &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  roundID,
					UserID:   "late-joiner",
					Response: roundtypes.ResponseTentative,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, res *roundtypes.JoinRoundRequest) {
				if res.JoinedLate == nil || *res.JoinedLate != true {
					t.Errorf("Expected JoinedLate to be true, got %v", res.JoinedLate)
				}
			},
		},
		{
			name: "Invalid: Nil Round ID - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest {
				return &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  sharedtypes.RoundID(uuid.Nil),
					UserID:   "some-user",
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "validation failed: [round ID cannot be nil]",
		},
		{
			name: "Invalid: Empty User ID - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.JoinRoundRequest {
				return &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  sharedtypes.RoundID(uuid.New()),
					UserID:   "",
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "validation failed: [participant Discord ID cannot be empty]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			req := tt.setupTestEnv(deps.Ctx, deps)

			result, err := deps.Service.ValidateParticipantJoinRequest(deps.Ctx, req)
			if err != nil {
				t.Fatalf("Unexpected error from service: %v", err)
			}

			if tt.expectedFailure {
				if result.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				} else if tt.expectedErrorMessagePart != "" {
					errMsg := (*result.Failure).Error()
					if !strings.Contains(errMsg, tt.expectedErrorMessagePart) {
						t.Errorf("Expected error to contain %q, got %q", tt.expectedErrorMessagePart, errMsg)
					}
				}
			} else {
				if result.Success == nil {
					t.Fatalf("Expected success result, but got failure: %+v", result.Failure)
				}
				if tt.validateResult != nil {
					tt.validateResult(t, deps.Ctx, deps, *result.Success)
				}
			}
		})
	}
}
