package leaderboardservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_GetLeaderboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	tag1 := sharedtypes.TagNumber(1)
	tag2 := sharedtypes.TagNumber(2)
	ctx := context.Background()
	testRoundID := sharedtypes.RoundID(uuid.New())

	// No-Op implementations for logging, metrics, and tracing
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &leaderboardmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockDBSetup    func(*leaderboarddb.MockLeaderboardDB)
		expectedResult *leaderboardevents.GetLeaderboardResponsePayload
		expectedFail   *leaderboardevents.GetLeaderboardFailedPayload
		expectedError  error
	}{
		{
			name: "Successfully retrieves leaderboard",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					ID: 1,
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{TagNumber: &tag1, UserID: "user1"},
						{TagNumber: &tag2, UserID: "user2"},
					},
					IsActive:     true,
					UpdateSource: leaderboarddbtypes.ServiceUpdateSourceProcessScores,
					UpdateID:     testRoundID,
				}, nil)
			},
			expectedResult: &leaderboardevents.GetLeaderboardResponsePayload{
				Leaderboard: []leaderboardtypes.LeaderboardEntry{
					{
						TagNumber: func() *sharedtypes.TagNumber {
							val := sharedtypes.TagNumber(1)
							return &val
						}(),
						UserID: "user1",
					},
					{
						TagNumber: func() *sharedtypes.TagNumber {
							val := sharedtypes.TagNumber(2)
							return &val
						}(),
						UserID: "user2",
					},
				},
			},
			expectedFail:  nil,
			expectedError: nil,
		},
		{
			name: "Fails to fetch active leaderboard",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(nil, errors.New("database connection error"))
			},
			expectedResult: nil,
			expectedFail: &leaderboardevents.GetLeaderboardFailedPayload{
				Reason: "failed to get active leaderboard",
			},
			expectedError: errors.New("database connection error"),
		},
		// New test case: Empty leaderboard
		{
			name: "Successfully retrieves empty leaderboard",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					ID:              1,
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{},
					IsActive:        true,
					UpdateSource:    leaderboarddbtypes.ServiceUpdateSourceProcessScores,
					UpdateID:        testRoundID,
				}, nil)
			},
			expectedResult: &leaderboardevents.GetLeaderboardResponsePayload{
				Leaderboard: []leaderboardtypes.LeaderboardEntry{},
			},
			expectedFail:  nil,
			expectedError: nil,
		},
		// New test case: Leaderboard with mixed tag numbers
		{
			name: "Successfully retrieves leaderboard with mixed tag numbers",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					ID: 1,
					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
						{TagNumber: &tag1, UserID: "user1"},
						{TagNumber: &tag2, UserID: "user2"}, // Mixed tag number
					},
					IsActive:     true,
					UpdateSource: leaderboarddbtypes.ServiceUpdateSourceProcessScores,
					UpdateID:     testRoundID,
				}, nil)
			},
			expectedResult: &leaderboardevents.GetLeaderboardResponsePayload{
				Leaderboard: []leaderboardtypes.LeaderboardEntry{
					{
						TagNumber: func() *sharedtypes.TagNumber {
							val := sharedtypes.TagNumber(1)
							return &val
						}(),
						UserID: "user1",
					},
					{
						TagNumber: func() *sharedtypes.TagNumber {
							val := sharedtypes.TagNumber(0) // Mixed tag number
							return &val
						}(),
						UserID: "user2",
					},
				},
			},
			expectedFail:  nil,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
			tt.mockDBSetup(mockDB)

			s := &LeaderboardService{
				LeaderboardDB: mockDB,
				logger:        logger,
				metrics:       metrics,
				tracer:        tracer,
				serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			got, err := s.GetLeaderboard(ctx)

			if (err != nil) != (tt.expectedError != nil) {
				t.Errorf("LeaderboardService.GetLeaderboard() error = %v, wantErr %v", err, tt.expectedError)
				return
			} else if err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("LeaderboardService.GetLeaderboard() error = %v, wantErr %v", err, tt.expectedError)
				return
			}

			if tt.expectedResult != nil {
				if got.Success == nil {
					t.Errorf("Expected success payload, got nil")
					return
				}

				successPayload, ok := got.Success.(*leaderboardevents.GetLeaderboardResponsePayload)
				if !ok {
					t.Errorf("Expected success payload type, got %T", got.Success)
					return
				}

				if !reflect.DeepEqual(successPayload, tt.expectedResult) {
					t.Errorf("LeaderboardService.GetLeaderboard() result mismatch: got %v, want %v", successPayload, tt.expectedResult)
				}
			} else {
				if got.Success != nil {
					t.Errorf("Expected nil success payload, got %v", got.Success)
				}
			}

			if tt.expectedFail != nil {
				if got.Failure == nil {
					t.Errorf("Expected failure payload, got nil")
					return
				}

				failurePayload, ok := got.Failure.(*leaderboardevents.GetLeaderboardFailedPayload)
				if !ok {
					t.Errorf("Expected failure payload type, got %T", got.Failure)
					return
				}

				if failurePayload.Reason != tt.expectedFail.Reason {
					t.Errorf("LeaderboardService.GetLeaderboard() failure reason mismatch: got %v, want %v", failurePayload.Reason, tt.expectedFail.Reason)
				}
			} else {
				if got.Failure != nil {
					t.Errorf("Expected nil failure payload, got %v", got.Failure)
				}
			}
		})
	}
}

func TestLeaderboardService_GetTagByUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testUserID := sharedtypes.DiscordID("user1")

	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &leaderboardmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		payload        sharedevents.DiscordTagLookupRequestPayload
		mockDBSetup    func(*leaderboarddb.MockLeaderboardDB)
		expectedResult *sharedevents.DiscordTagLookupResultPayload
		expectedFail   *sharedevents.DiscordTagLookupByUserIDFailedPayload
		expectedError  error
	}{
		{
			name: "Successfully retrieves tag number",
			payload: sharedevents.DiscordTagLookupRequestPayload{
				UserID: testUserID,
			},
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				tagNumber := int(5)
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(&tagNumber, nil)
			},
			expectedResult: &sharedevents.DiscordTagLookupResultPayload{
				TagNumber: func() *sharedtypes.TagNumber {
					val := sharedtypes.TagNumber(5)
					return &val
				}(),
				UserID: testUserID,
				Found:  true,
			},
			expectedFail:  nil,
			expectedError: nil,
		},
		{
			name: "Fails to retrieve tag number due to unexpected DB error",
			payload: sharedevents.DiscordTagLookupRequestPayload{
				UserID: testUserID,
			},
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(nil, errors.New("unexpected DB error"))
			},
			expectedResult: nil,
			expectedFail: &sharedevents.DiscordTagLookupByUserIDFailedPayload{
				Reason: "failed to get tag by UserID: unexpected DB error",
			},
			expectedError: fmt.Errorf("failed to get tag by UserID: %w", errors.New("unexpected DB error")),
		},
		{
			name: "User ID not found in database (sql.ErrNoRows)",
			payload: sharedevents.DiscordTagLookupRequestPayload{
				UserID: testUserID,
			},
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(nil, sql.ErrNoRows)
			},
			expectedResult: &sharedevents.DiscordTagLookupResultPayload{
				TagNumber: nil,
				UserID:    testUserID,
				Found:     false,
			},
			expectedFail:  nil,
			expectedError: nil,
		},
		{
			name: "No active leaderboard found",
			payload: sharedevents.DiscordTagLookupRequestPayload{
				UserID: testUserID,
			},
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(nil, leaderboarddbtypes.ErrNoActiveLeaderboard)
			},
			expectedResult: nil,
			expectedFail: &sharedevents.DiscordTagLookupByUserIDFailedPayload{
				Reason: "No active leaderboard found",
			},
			expectedError: nil,
		},
		{
			name: "Nil tag number returned (should not happen with sql.ErrNoRows handling, but testing robustness)",
			payload: sharedevents.DiscordTagLookupRequestPayload{
				UserID: testUserID,
			},
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(nil, nil)
			},
			expectedResult: &sharedevents.DiscordTagLookupResultPayload{
				TagNumber: nil,
				UserID:    testUserID,
				Found:     false,
			},
			expectedFail:  nil,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
			tt.mockDBSetup(mockDB)

			s := &LeaderboardService{
				LeaderboardDB: mockDB,
				logger:        logger,
				metrics:       metrics,
				tracer:        tracer,
				serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			got, err := s.GetTagByUserID(ctx, testUserID)

			if (err != nil) != (tt.expectedError != nil) {
				t.Errorf("LeaderboardService.GetTagByUserID() error = %v, wantErr %v", err, tt.expectedError)
				return
			} else if err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("LeaderboardService.GetTagByUserID() error = %v, wantErr %v", err, tt.expectedError)
				return
			}

			if tt.expectedResult != nil {
				if got.Success == nil {
					t.Errorf("Expected success payload, got nil")
					return
				}
				successPayload, ok := got.Success.(*sharedevents.DiscordTagLookupResultPayload)
				if !ok {
					t.Errorf("Expected success payload type, got %T", got.Success)
					return
				}

				if !reflect.DeepEqual(successPayload, tt.expectedResult) {
					t.Errorf("LeaderboardService.GetTagByUserID() result mismatch:\ngot  %v\nwant %v", successPayload, tt.expectedResult)
				}
			} else {
				if got.Success != nil {
					t.Errorf("Expected nil success payload, got %v", got.Success)
				}
			}

			if tt.expectedFail != nil {
				if got.Failure == nil {
					t.Errorf("Expected failure payload, got nil")
					return
				}

				failurePayload, ok := got.Failure.(*sharedevents.DiscordTagLookupByUserIDFailedPayload)
				if !ok {
					t.Errorf("Expected failure payload type, got %T", got.Failure)
					return
				}

				if failurePayload.Reason != tt.expectedFail.Reason {
					t.Errorf("LeaderboardService.GetTagByUserID() failure reason mismatch:\ngot  %v\nwant %v", failurePayload.Reason, tt.expectedFail.Reason)
				}
			} else {
				if got.Failure != nil {
					t.Errorf("Expected nil failure payload, got %v", got.Failure)
				}
			}
		})
	}
}

func TestLeaderboardService_RoundGetTagByUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testUserID := sharedtypes.DiscordID("user1")
	testRoundID := sharedtypes.RoundID(uuid.New())

	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &leaderboardmetrics.NoOpMetrics{}

	// Define a dummy request payload to use in tests
	dummyRequestPayload := sharedevents.RoundTagLookupRequestPayload{
		UserID:     testUserID,
		RoundID:    testRoundID,
		Response:   "ACCEPT",                                 // Dummy response
		JoinedLate: func() *bool { b := false; return &b }(), // Dummy joined late
	}

	tests := []struct {
		name           string
		mockDBSetup    func(*leaderboarddb.MockLeaderboardDB)
		requestPayload sharedevents.RoundTagLookupRequestPayload // New field for the input payload
		expectedResult *sharedevents.RoundTagLookupResultPayload // Updated expected result type
		expectedError  error
	}{
		{
			name: "Successfully retrieves tag number",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				tagNumber := 5 // Use int directly as DB mock returns *int
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(&tagNumber, nil)
			},
			requestPayload: dummyRequestPayload, // Use dummy request payload
			expectedResult: &sharedevents.RoundTagLookupResultPayload{ // Updated expected result payload structure
				UserID:  testUserID,
				RoundID: testRoundID,
				TagNumber: func() *sharedtypes.TagNumber { // Correctly define expected TagNumber pointer
					val := sharedtypes.TagNumber(5)
					return &val
				}(),
				Found: true, // Expect Found to be true
				Error: "",   // Expect no service error
				// Echoed original context from the request payload
				OriginalResponse:   dummyRequestPayload.Response,
				OriginalJoinedLate: dummyRequestPayload.JoinedLate,
			},
			expectedError: nil,
		},
		{
			name: "Fails to retrieve tag number (operational error)", // Renamed for clarity
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				// Return a non-sql.ErrNoRows error
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(nil, errors.New("database connection error"))
			},
			requestPayload: dummyRequestPayload, // Use dummy request payload
			expectedResult: &sharedevents.RoundTagLookupResultPayload{ // Expect a result payload with error details
				UserID:    testUserID,
				RoundID:   testRoundID,
				TagNumber: nil,                                            // No tag found on error
				Found:     false,                                          // Expect Found to be false
				Error:     "failed to get tag: database connection error", // Expect service error string
				// Echoed original context from the request payload
				OriginalResponse:   dummyRequestPayload.Response,
				OriginalJoinedLate: dummyRequestPayload.JoinedLate,
			},
			expectedError: errors.New("failed to get tag by UserID: database connection error"), // Expect actual operational error returned
		},
		{
			name: "User ID not found in database (sql.ErrNoRows)", // Renamed for clarity
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				// Return sql.ErrNoRows to simulate "not found"
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(nil, sql.ErrNoRows)
			},
			requestPayload: dummyRequestPayload, // Use dummy request payload
			expectedResult: &sharedevents.RoundTagLookupResultPayload{ // Expect a result payload indicating not found
				UserID:    testUserID,
				RoundID:   testRoundID,
				TagNumber: nil,   // No tag found
				Found:     false, // Expect Found to be false
				Error:     "",    // Expect no service error string
				// Echoed original context from the request payload
				OriginalResponse:   dummyRequestPayload.Response,
				OriginalJoinedLate: dummyRequestPayload.JoinedLate,
			},
			expectedError: nil, // Expect no operational error
		},
		{
			name: "User ID not found in database (string match)", // Test the string match for "not found"
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				// Return an error with "not found" in the string
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(nil, errors.New("user not found in DB"))
			},
			requestPayload: dummyRequestPayload, // Use dummy request payload
			expectedResult: &sharedevents.RoundTagLookupResultPayload{ // Expect a result payload indicating not found
				UserID:    testUserID,
				RoundID:   testRoundID,
				TagNumber: nil,   // No tag found
				Found:     false, // Expect Found to be false
				Error:     "",    // Expect no service error string (as it's treated as not found)
				// Echoed original context from the request payload
				OriginalResponse:   dummyRequestPayload.Response,
				OriginalJoinedLate: dummyRequestPayload.JoinedLate,
			},
			expectedError: nil, // Expect no operational error
		},
		{
			name: "Nil tag number returned from database", // Renamed for clarity
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				// Return nil, nil explicitly
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(nil, nil)
			},
			requestPayload: dummyRequestPayload, // Use dummy request payload
			expectedResult: &sharedevents.RoundTagLookupResultPayload{ // Expect a result payload indicating not found
				UserID:    testUserID,
				RoundID:   testRoundID,
				TagNumber: nil,   // No tag found
				Found:     false, // Expect Found to be false
				Error:     "",    // Expect no service error string
				// Echoed original context from the request payload
				OriginalResponse:   dummyRequestPayload.Response,
				OriginalJoinedLate: dummyRequestPayload.JoinedLate,
			},
			expectedError: nil, // Expect no operational error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
			tt.mockDBSetup(mockDB)

			// Re-define the serviceWrapper mock here to ensure it matches the service method's call
			s := &LeaderboardService{
				LeaderboardDB: mockDB,
				logger:        logger,
				metrics:       metrics,
				tracer:        tracer,
				// Mock serviceWrapper - should accept only ctx, opName, and serviceFunc
				serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			// Call the service method with the updated signature
			got, err := s.RoundGetTagByUserID(ctx, tt.requestPayload)

			// --- Error Checking ---
			if (err != nil) != (tt.expectedError != nil) {
				t.Errorf("LeaderboardService.GetTagByUserID() error mismatch: got = %v, wantErr %v", err, tt.expectedError)
				return // Stop on error mismatch
			}
			if err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("LeaderboardService.GetTagByUserID() error message mismatch: got = %v, want %v", err.Error(), tt.expectedError.Error())
				return // Stop on error message mismatch
			}

			// --- Success Payload Checking ---
			if tt.expectedResult != nil {
				if got.Success == nil {
					t.Errorf("Expected success payload (%T), got nil LeaderboardOperationResult.Success", tt.expectedResult)
					return // Stop if expected success but got nil
				}
				// Assert the returned success payload to the correct type
				successPayload, ok := got.Success.(*sharedevents.RoundTagLookupResultPayload) // <-- Assert to the new shared result payload type
				if !ok {
					t.Errorf("Expected success payload type %T, got %T", tt.expectedResult, got.Success)
					return // Stop on type assertion failure
				}

				// Deep compare the contents of the success payload
				if !reflect.DeepEqual(successPayload, tt.expectedResult) {
					t.Errorf("LeaderboardService.GetTagByUserID() result payload mismatch:\n  got: %#v\n want: %#v", successPayload, tt.expectedResult)
				}
			} else {
				// Expected nil success payload
				if got.Success != nil {
					t.Errorf("Expected nil success payload, got %v", got.Success)
				}
			}
		})
	}
}
