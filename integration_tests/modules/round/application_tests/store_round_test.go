package roundintegrationtests

import (
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
)

func TestStoreRound(t *testing.T) {
	// Helper to create a valid RoundEntityCreatedPayload
	createValidPayload := func() roundevents.RoundEntityCreatedPayloadV1 {
		description := roundtypes.Description("Test Description")
		location := roundtypes.Location("Test Location")
		startTime := sharedtypes.StartTime(time.Now().Add(24 * time.Hour).UTC())
		createdBy := sharedtypes.DiscordID("user_123")
		channelID := "channel_abc"
		eventType := roundtypes.EventType("casual")

		// Directly construct RoundEntityCreatedPayload
		return roundevents.RoundEntityCreatedPayloadV1{
			GuildID: sharedtypes.GuildID("test-guild"),
			Round: roundtypes.Round{
				ID:           sharedtypes.RoundID(uuid.New()),
				Title:        roundtypes.Title("Test Round Title"),
				Description:  &description,
				Location:     &location,
				EventType:    &eventType,
				StartTime:    &startTime,
				Finalized:    false,
				CreatedBy:    createdBy,
				State:        roundtypes.RoundStateUpcoming,
				Participants: []roundtypes.Participant{},
				GuildID:      sharedtypes.GuildID("test-guild"),
			},
			DiscordChannelID: channelID,
			DiscordGuildID:   "test-guild",
		}
	}

	tests := []struct {
		name            string
		payload         roundevents.RoundEntityCreatedPayloadV1
		expectedError   bool
		expectedSuccess bool
		validateResult  func(t *testing.T, deps *RoundTestDeps, result results.OperationResult)
		validateDBState func(t *testing.T, deps *RoundTestDeps, expectedRoundID sharedtypes.RoundID) // Expect the ID from the result
	}{
		{
			name:            "Successful round storage",
			payload:         createValidPayload(),
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps *RoundTestDeps, result results.OperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				successPayload, ok := result.Success.(*roundevents.RoundCreatedPayloadV1)
				if !ok {
					t.Errorf("Expected success result of type *RoundCreatedPayload, but got %T", result.Success)
					return
				}
				if successPayload.RoundID == sharedtypes.RoundID(uuid.Nil) {
					t.Errorf("Expected a non-empty RoundID in success payload, got %s", successPayload.RoundID)
				}
				if successPayload.Title != "Test Round Title" {
					t.Errorf("Expected title 'Test Round Title', got '%s'", successPayload.Title)
				}
				if *successPayload.Description != "Test Description" {
					t.Errorf("Expected description 'Test Description', got '%s'", *successPayload.Description)
				}
				if *successPayload.Location != "Test Location" {
					t.Errorf("Expected location 'Test Location', got '%s'", *successPayload.Location)
				}
				if successPayload.UserID != sharedtypes.DiscordID("user_123") {
					t.Errorf("Expected UserID 'user_123', got '%s'", successPayload.UserID)
				}
				if successPayload.ChannelID != "channel_abc" {
					t.Errorf("Expected ChannelID 'channel_abc', got '%s'", successPayload.ChannelID)
				}
			},
			validateDBState: func(t *testing.T, deps *RoundTestDeps, expectedRoundID sharedtypes.RoundID) {
				// Retrieve the stored round using the ID returned in the success payload
				storedRound, err := deps.DB.GetRound(deps.Ctx, sharedtypes.GuildID("test-guild"), expectedRoundID)
				if err != nil {
					t.Fatalf("Failed to retrieve stored round from DB: %v", err)
				}
				if storedRound.Title != "Test Round Title" {
					t.Errorf("Stored round title mismatch: expected 'Test Round Title', got '%s'", storedRound.Title)
				}
				if *storedRound.Description != "Test Description" {
					t.Errorf("Stored round description mismatch: expected 'Test Description', got '%s'", *storedRound.Description)
				}
				if *storedRound.Location != "Test Location" {
					t.Errorf("Stored round location mismatch: expected 'Test Location', got '%s'", *storedRound.Location)
				}
				if storedRound.CreatedBy != sharedtypes.DiscordID("user_123") {
					t.Errorf("Stored round CreatedBy mismatch: expected 'user_123', got '%s'", storedRound.CreatedBy)
				}
				if storedRound.State != roundtypes.RoundStateUpcoming {
					t.Errorf("Stored round state mismatch: expected 'Upcoming', got '%s'", storedRound.State)
				}
				if *storedRound.EventType != "casual" {
					t.Errorf("Stored round event type mismatch: expected 'casual', got '%s'", *storedRound.EventType)
				}
			},
		},
		{
			name: "Validation failure - missing title in payload",
			payload: func() roundevents.RoundEntityCreatedPayloadV1 {
				p := createValidPayload()
				p.Round.Title = "" // Empty title
				return p
			}(),
			expectedError:   true,
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps *RoundTestDeps, result results.OperationResult) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*roundevents.RoundCreationFailedPayloadV1)
				if !ok {
					t.Errorf("Expected failure result of type *RoundCreationFailedPayload, but got %T", result.Failure)
					return
				}
				expectedErrMsg := "invalid round data"
				if failurePayload.ErrorMessage != expectedErrMsg {
					t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, failurePayload.ErrorMessage)
				}
				if failurePayload.UserID != sharedtypes.DiscordID("user_123") {
					t.Errorf("Expected UserID 'user_123', got '%s'", failurePayload.UserID)
				}
			},
			validateDBState: func(t *testing.T, deps *RoundTestDeps, expectedRoundID sharedtypes.RoundID) {
				// No DB interaction expected for validation failures
			},
		},
		{
			name: "Validation failure - nil description in payload",
			payload: func() roundevents.RoundEntityCreatedPayloadV1 {
				p := createValidPayload()
				p.Round.Description = nil // Nil description
				return p
			}(),
			expectedError:   true,
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps *RoundTestDeps, result results.OperationResult) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*roundevents.RoundCreationFailedPayloadV1)
				if !ok {
					t.Errorf("Expected failure result of type *RoundCreationFailedPayload, but got %T", result.Failure)
					return
				}
				expectedErrMsg := "invalid round data"
				if failurePayload.ErrorMessage != expectedErrMsg {
					t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, failurePayload.ErrorMessage)
				}
			},
			validateDBState: func(t *testing.T, deps *RoundTestDeps, expectedRoundID sharedtypes.RoundID) {
				// No DB interaction expected for validation failures
			},
		},
		{
			name: "Validation failure - nil location in payload",
			payload: func() roundevents.RoundEntityCreatedPayloadV1 {
				p := createValidPayload()
				p.Round.Location = nil // Nil location
				return p
			}(),
			expectedError:   true,
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps *RoundTestDeps, result results.OperationResult) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*roundevents.RoundCreationFailedPayloadV1)
				if !ok {
					t.Errorf("Expected failure result of type *RoundCreationFailedPayload, but got %T", result.Failure)
					return
				}
				expectedErrMsg := "invalid round data"
				if failurePayload.ErrorMessage != expectedErrMsg {
					t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, failurePayload.ErrorMessage)
				}
			},
			validateDBState: func(t *testing.T, deps *RoundTestDeps, expectedRoundID sharedtypes.RoundID) {
				// No DB interaction expected for validation failures
			},
		},
		{
			name: "Validation failure - nil start time in payload",
			payload: func() roundevents.RoundEntityCreatedPayloadV1 {
				p := createValidPayload()
				p.Round.StartTime = nil // Nil start time
				return p
			}(),
			expectedError:   true,
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps *RoundTestDeps, result results.OperationResult) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*roundevents.RoundCreationFailedPayloadV1)
				if !ok {
					t.Errorf("Expected failure result of type *RoundCreationFailedPayload, but got %T", result.Failure)
					return
				}
				expectedErrMsg := "invalid round data"
				if failurePayload.ErrorMessage != expectedErrMsg {
					t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, failurePayload.ErrorMessage)
				}
			},
			validateDBState: func(t *testing.T, deps *RoundTestDeps, expectedRoundID sharedtypes.RoundID) {
				// No DB interaction expected for validation failures
			},
		},
		// Removed "Database creation error" test case as it requires mock-like DB behavior or complex real DB error injection.
		// If you need to test specific database errors (e.g., constraint violations), you would add
		// test cases here that attempt to trigger those conditions on your real database.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize dependencies for each test, using the shared DB connection
			deps := SetupTestRoundService(t)

			result, err := deps.Service.StoreRound(deps.Ctx, sharedtypes.GuildID("test-guild"), tt.payload)

			// Extract the RoundID from the successful result for DB state validation
			var expectedRoundID sharedtypes.RoundID
			if tt.expectedSuccess {
				if successPayload, ok := result.Success.(*roundevents.RoundCreatedPayloadV1); ok {
					expectedRoundID = successPayload.RoundID
				}
			}

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			}

			if tt.expectedSuccess {
				if result.Success == nil {
					t.Errorf("Expected a success result, but got nil")
				}
			} else {
				if result.Success != nil {
					t.Errorf("Expected no success result, but got: %+v", result.Success)
				}
			}

			if tt.validateResult != nil {
				tt.validateResult(t, &deps, result)
			}
			if tt.validateDBState != nil {
				tt.validateDBState(t, &deps, expectedRoundID)
			}
		})
	}
}
