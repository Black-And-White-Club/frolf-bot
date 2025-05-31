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
	"github.com/google/uuid"
)

// EventTypePtr is a helper function to create a pointer to EventType
func EventTypePtr(et roundtypes.EventType) *roundtypes.EventType {
	return &et
}

// TestValidateRoundUpdateRequest tests the ValidateRoundUpdateRequest method.
func TestValidateRoundUpdateRequest(t *testing.T) {
	tests := []struct {
		name                     string
		payload                  roundevents.RoundUpdateRequestPayload
		expectedError            bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, returnedResult roundservice.RoundOperationResult)
	}{
		{
			name: "Valid update request - Title only",
			payload: roundevents.RoundUpdateRequestPayload{
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID: sharedtypes.RoundID(uuid.New()),
					Title:   roundtypes.Title("New Title"),
				},
			},
			expectedError: false,
			validateResult: func(t *testing.T, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				validatedPayload, ok := returnedResult.Success.(*roundevents.RoundUpdateValidatedPayload)
				if !ok {
					t.Errorf("Expected *RoundUpdateValidatedPayload, got %T", returnedResult.Success)
				}
				if validatedPayload.RoundUpdateRequestPayload.Title != "New Title" {
					t.Errorf("Expected title 'New Title', got '%s'", validatedPayload.RoundUpdateRequestPayload.Title)
				}
			},
		},
		{
			name: "Valid update request - All fields including EventType",
			payload: roundevents.RoundUpdateRequestPayload{
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID:     sharedtypes.RoundID(uuid.New()),
					Title:       roundtypes.Title("New Title"),
					Description: roundtypes.DescriptionPtr("New Description"),
					Location:    roundtypes.LocationPtr("New Location"),
					StartTime:   roundtypes.StartTimePtr(time.Now().Add(24 * time.Hour)),
				},
				EventType: EventTypePtr(roundtypes.EventType("tournament")),
			},
			expectedError: false,
			validateResult: func(t *testing.T, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				validatedPayload, ok := returnedResult.Success.(*roundevents.RoundUpdateValidatedPayload)
				if !ok {
					t.Errorf("Expected *RoundUpdateValidatedPayload, got %T", returnedResult.Success)
				}
				if validatedPayload.RoundUpdateRequestPayload.Title != "New Title" {
					t.Errorf("Expected title 'New Title', got '%s'", validatedPayload.RoundUpdateRequestPayload.Title)
				}
				if *validatedPayload.RoundUpdateRequestPayload.Description != "New Description" {
					t.Errorf("Expected description 'New Description', got '%s'", *validatedPayload.RoundUpdateRequestPayload.Description)
				}
				if *validatedPayload.RoundUpdateRequestPayload.Location != "New Location" {
					t.Errorf("Expected location 'New Location', got '%s'", *validatedPayload.RoundUpdateRequestPayload.Location)
				}
				if validatedPayload.RoundUpdateRequestPayload.StartTime == nil {
					t.Errorf("Expected StartTime to be set")
				}
				if validatedPayload.RoundUpdateRequestPayload.EventType == nil || *validatedPayload.RoundUpdateRequestPayload.EventType != roundtypes.EventType("tournament") {
					t.Errorf("Expected EventType to be 'tournament', got '%v'", validatedPayload.RoundUpdateRequestPayload.EventType)
				}
			},
		},
		{
			name: "Invalid update request - Zero RoundID",
			payload: roundevents.RoundUpdateRequestPayload{
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID: sharedtypes.RoundID(uuid.Nil),
					Title:   roundtypes.Title("New Title"),
				},
			},
			expectedError:            true,
			expectedErrorMessagePart: "round ID cannot be zero",
			validateResult: func(t *testing.T, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				errorPayload, ok := returnedResult.Failure.(*roundevents.RoundUpdateErrorPayload)
				if !ok {
					t.Errorf("Expected RoundUpdateErrorPayload, got %T", returnedResult.Failure)
				}
				if !strings.Contains(errorPayload.Error, "round ID cannot be zero") {
					t.Errorf("Expected error to contain 'round ID cannot be zero', got '%s'", errorPayload.Error)
				}
			},
		},
		{
			name: "Invalid update request - No fields to update",
			payload: roundevents.RoundUpdateRequestPayload{
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID: sharedtypes.RoundID(uuid.New()),
				},
			},
			expectedError:            true,
			expectedErrorMessagePart: "at least one field to update must be provided",
			validateResult: func(t *testing.T, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				errorPayload, ok := returnedResult.Failure.(*roundevents.RoundUpdateErrorPayload)
				if !ok {
					t.Errorf("Expected RoundUpdateErrorPayload, got %T", returnedResult.Failure)
				}
				if !strings.Contains(errorPayload.Error, "at least one field to update must be provided") {
					t.Errorf("Expected error to contain 'at least one field to update must be provided', got '%s'", errorPayload.Error)
				}
			},
		},
		{
			name: "Invalid update request - Zero RoundID and no fields to update",
			payload: roundevents.RoundUpdateRequestPayload{
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID: sharedtypes.RoundID(uuid.Nil),
				},
			},
			expectedError:            true,
			expectedErrorMessagePart: "round ID cannot be zero; at least one field to update must be provided",
			validateResult: func(t *testing.T, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				errorPayload, ok := returnedResult.Failure.(*roundevents.RoundUpdateErrorPayload)
				if !ok {
					t.Errorf("Expected RoundUpdateErrorPayload, got %T", returnedResult.Failure)
				}
				if !strings.Contains(errorPayload.Error, "round ID cannot be zero") || !strings.Contains(errorPayload.Error, "at least one field to update must be provided") {
					t.Errorf("Expected error to contain both validation messages, got '%s'", errorPayload.Error)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)

			result, err := deps.Service.ValidateRoundUpdateRequest(deps.Ctx, tt.payload)

			if tt.expectedError {
				// Check for business failure, not Go error
				if err != nil {
					t.Errorf("Expected no Go error (business failure should be in result), but got: %v", err)
				}
				if result.Failure == nil {
					t.Errorf("Expected a failure payload, but got nil")
				}
				if result.Success != nil {
					t.Errorf("Expected nil success payload, but got %v", result.Success)
				}

				if result.Failure != nil && tt.expectedErrorMessagePart != "" {
					errorPayload, ok := result.Failure.(*roundevents.RoundUpdateErrorPayload)
					if ok && !strings.Contains(errorPayload.Error, tt.expectedErrorMessagePart) {
						t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedErrorMessagePart, errorPayload.Error)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
				if result.Failure != nil {
					t.Errorf("Expected nil failure payload, but got %v", result.Failure)
				}
				if result.Success == nil {
					t.Errorf("Expected a success payload, but got nil")
				}
			}

			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

// TestUpdateRoundEntity tests the UpdateRoundEntity method.
func TestUpdateRoundEntity(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) (*roundevents.RoundUpdateValidatedPayload, sharedtypes.RoundID)
		expectedError            bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult)
	}{
		{
			name: "Successful update of title",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (*roundevents.RoundUpdateValidatedPayload, sharedtypes.RoundID) {
				// Initialize originalRound with a new UUID. This ID will be overwritten by the DB.
				originalRound := &roundtypes.Round{
					ID:        sharedtypes.RoundID(uuid.New()), // Initial UUID, will be replaced by DB-generated ID
					Title:     roundtypes.Title("Original Title"),
					StartTime: roundtypes.StartTimePtr(time.Now().Add(24 * time.Hour)),
					EventType: EventTypePtr(roundtypes.EventType("tournament")),
					CreatedBy: "user123",
					State:     roundtypes.RoundState("UPCOMING"),
					Finalized: false,
				}
				// Create the round in the real DB.
				// After this call, originalRound.ID will contain the actual ID assigned by the database.
				err := deps.DB.CreateRound(ctx, originalRound)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB: %v", err)
				}

				// Use the actual ID returned by the database for the update payload
				payload := roundevents.RoundUpdateValidatedPayload{
					RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
						BaseRoundPayload: roundtypes.BaseRoundPayload{
							RoundID: originalRound.ID, // <-- CRITICAL FIX: Use the DB-assigned ID
							Title:   roundtypes.Title("Updated Title"),
						},
					},
				}
				return &payload, originalRound.ID
			},
			expectedError: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				entityUpdatedPayload, ok := returnedResult.Success.(*roundevents.RoundEntityUpdatedPayload)
				if !ok {
					t.Errorf("Expected RoundEntityUpdatedPayload, got %T", returnedResult.Success)
				}
				if entityUpdatedPayload.Round.Title != "Updated Title" {
					t.Errorf("Expected round title to be 'Updated Title', got '%s'", entityUpdatedPayload.Round.Title)
				}

				// Verify DB state by fetching from the real DB
				updatedRoundInDB, err := deps.DB.GetRound(ctx, entityUpdatedPayload.Round.ID)
				if err != nil {
					t.Fatalf("Failed to fetch updated round from DB: %v", err)
				}
				if updatedRoundInDB.Title != "Updated Title" {
					t.Errorf("Expected round in DB to have title 'Updated Title', got '%s'", updatedRoundInDB.Title)
				}
			},
		},
		{
			name: "Successful update of description, location, and start time",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (*roundevents.RoundUpdateValidatedPayload, sharedtypes.RoundID) {
				// Initialize originalRound with a new UUID. This ID will be overwritten by the DB.
				originalRound := &roundtypes.Round{
					ID:          sharedtypes.RoundID(uuid.New()), // Initial UUID, will be replaced by DB-generated ID
					Title:       roundtypes.Title("Original Title"),
					Description: roundtypes.DescriptionPtr("Old Desc"),
					Location:    roundtypes.LocationPtr("Old Loc"),
					StartTime:   roundtypes.StartTimePtr(time.Now().Add(-24 * time.Hour)),
					EventType:   EventTypePtr(roundtypes.EventType("casual")),
					CreatedBy:   "user123",
					State:       roundtypes.RoundState("UPCOMING"),
					Finalized:   false,
				}
				// Create the round in the real DB.
				// After this call, originalRound.ID will contain the actual ID assigned by the database.
				err := deps.DB.CreateRound(ctx, originalRound)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB: %v", err)
				}

				newStartTime := time.Now().Add(48 * time.Hour)
				// Use the actual ID returned by the database for the update payload
				payload := roundevents.RoundUpdateValidatedPayload{
					RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
						BaseRoundPayload: roundtypes.BaseRoundPayload{
							RoundID:     originalRound.ID, // <-- CRITICAL FIX: Use the DB-assigned ID
							Description: roundtypes.DescriptionPtr("New Description"),
							Location:    roundtypes.LocationPtr("New Location"),
							StartTime:   roundtypes.StartTimePtr(newStartTime),
						},
					},
				}
				return &payload, originalRound.ID
			},
			expectedError: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				entityUpdatedPayload, ok := returnedResult.Success.(*roundevents.RoundEntityUpdatedPayload)
				if !ok {
					t.Errorf("Expected RoundEntityUpdatedPayload, got %T", returnedResult.Success)
				}
				if *entityUpdatedPayload.Round.Description != "New Description" {
					t.Errorf("Expected description 'New Description', got '%s'", *entityUpdatedPayload.Round.Description)
				}
				if *entityUpdatedPayload.Round.Location != "New Location" {
					t.Errorf("Expected location 'New Location', got '%s'", *entityUpdatedPayload.Round.Location)
				}
				if entityUpdatedPayload.Round.StartTime == nil || !time.Time(*entityUpdatedPayload.Round.StartTime).After(time.Now()) {
					t.Errorf("Expected StartTime to be updated and in future, got %v", entityUpdatedPayload.Round.StartTime)
				}

				// Verify DB state by fetching from the real DB
				updatedRoundInDB, err := deps.DB.GetRound(ctx, entityUpdatedPayload.Round.ID)
				if err != nil {
					t.Fatalf("Failed to fetch updated round from DB: %v", err)
				}
				if *updatedRoundInDB.Description != "New Description" {
					t.Errorf("Expected round in DB to have description 'New Description', got '%s'", *updatedRoundInDB.Description)
				}
				if *updatedRoundInDB.Location != "New Location" {
					t.Errorf("Expected round in DB to have location 'New Location', got '%s'", *updatedRoundInDB.Location)
				}
				if updatedRoundInDB.StartTime == nil || !time.Time(*updatedRoundInDB.StartTime).After(time.Now()) {
					t.Errorf("Expected round in DB to have updated StartTime, got %v", updatedRoundInDB.StartTime)
				}
			},
		},
		{
			name: "Successful update of EventType",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (*roundevents.RoundUpdateValidatedPayload, sharedtypes.RoundID) {
				// Initialize originalRound with a new UUID. This ID will be overwritten by the DB.
				originalRound := &roundtypes.Round{
					ID:        sharedtypes.RoundID(uuid.New()), // Initial UUID, will be replaced by DB-generated ID
					Title:     roundtypes.Title("Original Title"),
					StartTime: roundtypes.StartTimePtr(time.Now().Add(24 * time.Hour)),
					EventType: EventTypePtr(roundtypes.EventType("casual")),
					CreatedBy: "user123",
					State:     roundtypes.RoundState("UPCOMING"),
					Finalized: false,
				}
				// Create the round in the real DB.
				// After this call, originalRound.ID will contain the actual ID assigned by the database.
				err := deps.DB.CreateRound(ctx, originalRound)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB: %v", err)
				}

				newEventType := roundtypes.EventType("tournament")
				// Use the actual ID returned by the database for the update payload
				payload := roundevents.RoundUpdateValidatedPayload{
					RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
						BaseRoundPayload: roundtypes.BaseRoundPayload{
							RoundID: originalRound.ID, // <-- CRITICAL FIX: Use the DB-assigned ID
						},
						EventType: &newEventType,
					},
				}
				return &payload, originalRound.ID
			},
			expectedError: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				entityUpdatedPayload, ok := returnedResult.Success.(*roundevents.RoundEntityUpdatedPayload)
				if !ok {
					t.Errorf("Expected RoundEntityUpdatedPayload, got %T", returnedResult.Success)
				}
				if entityUpdatedPayload.Round.EventType == nil || *entityUpdatedPayload.Round.EventType != roundtypes.EventType("tournament") {
					t.Errorf("Expected EventType to be 'tournament', got '%v'", entityUpdatedPayload.Round.EventType)
				}

				// Verify DB state by fetching from the real DB
				updatedRoundInDB, err := deps.DB.GetRound(ctx, entityUpdatedPayload.Round.ID)
				if err != nil {
					t.Fatalf("Failed to fetch updated round from DB: %v", err)
				}
				if updatedRoundInDB.EventType == nil || *updatedRoundInDB.EventType != roundtypes.EventType("tournament") {
					t.Errorf("Expected round in DB to have EventType 'tournament', got '%v'", updatedRoundInDB.EventType)
				}
			},
		},
		{
			name: "Failed to fetch existing round (round not in DB)",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (*roundevents.RoundUpdateValidatedPayload, sharedtypes.RoundID) {
				roundID := sharedtypes.RoundID(uuid.New()) // This round will not be created in the DB
				payload := roundevents.RoundUpdateValidatedPayload{
					RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
						BaseRoundPayload: roundtypes.BaseRoundPayload{
							RoundID: roundID,
							Title:   roundtypes.Title("New Title"),
						},
					},
				}
				return &payload, roundID
			},
			expectedError:            true,
			expectedErrorMessagePart: "not found",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				errorPayload, ok := returnedResult.Failure.(*roundevents.RoundUpdateErrorPayload)
				if !ok {
					t.Errorf("Expected RoundUpdateErrorPayload, got %T", returnedResult.Failure)
				}
				if !strings.Contains(errorPayload.Error, "not found") {
					t.Errorf("Expected error to contain 'not found', got '%s'", errorPayload.Error)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t) // Assuming this provides a clean, real DB connection

			payload, _ := tt.setupTestEnv(deps.Ctx, deps)

			result, err := deps.Service.UpdateRoundEntity(deps.Ctx, *payload)

			if tt.expectedError {
				// Check for business failure, not Go error
				if err != nil {
					t.Errorf("Expected no Go error (business failure should be in result), but got: %v", err)
				}
				if result.Failure == nil {
					t.Errorf("Expected a failure payload, but got nil")
				}
				if result.Success != nil {
					t.Errorf("Expected nil success payload, but got %v", result.Success)
				}

				if result.Failure != nil && tt.expectedErrorMessagePart != "" {
					errorPayload, ok := result.Failure.(*roundevents.RoundUpdateErrorPayload)
					if ok && !strings.Contains(errorPayload.Error, tt.expectedErrorMessagePart) {
						t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedErrorMessagePart, errorPayload.Error)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
				if result.Failure != nil {
					t.Errorf("Expected nil failure payload, but got %v", result.Failure)
				}
				if result.Success == nil {
					t.Errorf("Expected a success payload, but got nil")
				}
			}

			if tt.validateResult != nil {
				tt.validateResult(t, deps.Ctx, deps, result)
			}
		})
	}
}

// TestUpdateScheduledRoundEvents tests the UpdateScheduledRoundEvents method.
func TestUpdateScheduledRoundEvents(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) (*roundevents.RoundScheduleUpdatePayload, sharedtypes.RoundID)
		expectedError            bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult)
	}{
		{
			name: "Successful update of scheduled events",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (*roundevents.RoundScheduleUpdatePayload, sharedtypes.RoundID) {
				// Initialize round with a new UUID. This ID will be overwritten by the DB.
				round := &roundtypes.Round{
					ID:        sharedtypes.RoundID(uuid.New()), // Initial UUID, will be replaced by DB-generated ID
					Title:     roundtypes.Title("Scheduled Round"),
					StartTime: roundtypes.StartTimePtr(time.Now().Add(24 * time.Hour)),
					EventType: EventTypePtr(roundtypes.EventType("casual")),
					CreatedBy: "user123",
					State:     roundtypes.RoundState("UPCOMING"),
					Finalized: false,
				}
				// Create the round in the real DB.
				// After this call, round.ID will contain the actual ID assigned by the database.
				err := deps.DB.CreateRound(ctx, round)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB: %v", err)
				}

				// Use the actual ID returned by the database for the update payload
				payload := roundevents.RoundScheduleUpdatePayload{RoundID: round.ID} // <-- CRITICAL FIX: Use the DB-assigned ID
				return &payload, round.ID
			},
			expectedError: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				storedPayload, ok := returnedResult.Success.(*roundevents.RoundStoredPayload)
				if !ok {
					t.Errorf("Expected RoundStoredPayload, got %T", returnedResult.Success)
				}
				if storedPayload.Round.ID == (sharedtypes.RoundID(uuid.Nil)) {
					t.Errorf("Expected a valid round ID in success payload, got zero UUID")
				}
				if storedPayload.Round.Title != "Scheduled Round" {
					t.Errorf("Expected round title 'Scheduled Round', got '%s'", storedPayload.Round.Title)
				}
			},
		},
		{
			name: "Failed to fetch round for rescheduling",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (*roundevents.RoundScheduleUpdatePayload, sharedtypes.RoundID) {
				roundID := sharedtypes.RoundID(uuid.New()) // This round will not be created in the DB
				payload := roundevents.RoundScheduleUpdatePayload{RoundID: roundID}
				return &payload, roundID
			},
			expectedError:            true,
			expectedErrorMessagePart: "not found",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				errorPayload, ok := returnedResult.Failure.(*roundevents.RoundUpdateErrorPayload)
				if !ok {
					t.Errorf("Expected RoundUpdateErrorPayload, got %T", returnedResult.Failure)
				}
				if !strings.Contains(errorPayload.Error, "not found") {
					t.Errorf("Expected error to contain 'not found', got '%s'", errorPayload.Error)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t) // Assuming this provides a clean, real DB connection
			payload, _ := tt.setupTestEnv(deps.Ctx, deps)

			result, err := deps.Service.UpdateScheduledRoundEvents(deps.Ctx, *payload)

			if tt.expectedError {
				// Check for business failure, not Go error
				if err != nil {
					t.Errorf("Expected no Go error (business failure should be in result), but got: %v", err)
				}
				if result.Failure == nil {
					t.Errorf("Expected a failure payload, but got nil")
				}
				if result.Success != nil {
					t.Errorf("Expected nil success payload, but got %v", result.Success)
				}

				if result.Failure != nil && tt.expectedErrorMessagePart != "" {
					errorPayload, ok := result.Failure.(*roundevents.RoundUpdateErrorPayload)
					if ok && !strings.Contains(errorPayload.Error, tt.expectedErrorMessagePart) {
						t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedErrorMessagePart, errorPayload.Error)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
				if result.Failure != nil {
					t.Errorf("Expected nil failure payload, but got %v", result.Failure)
				}
				if result.Success == nil {
					t.Errorf("Expected a success payload, but got nil")
				}
			}

			if tt.validateResult != nil {
				tt.validateResult(t, deps.Ctx, deps, result)
			}
		})
	}
}
