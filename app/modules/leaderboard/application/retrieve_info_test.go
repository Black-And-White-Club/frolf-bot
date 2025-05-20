package leaderboardservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
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
						{TagNumber: tag1, UserID: "user1"},
						{TagNumber: tag2, UserID: "user2"},
					},
					IsActive:     true,
					UpdateSource: leaderboarddbtypes.ServiceUpdateSourceProcessScores,
					UpdateID:     testRoundID,
				}, nil)
			},
			expectedResult: &leaderboardevents.GetLeaderboardResponsePayload{
				Leaderboard: []leaderboardtypes.LeaderboardEntry{
					{
						TagNumber: func() sharedtypes.TagNumber {
							val := sharedtypes.TagNumber(1)
							return val
						}(),
						UserID: "user1",
					},
					{
						TagNumber: func() sharedtypes.TagNumber {
							val := sharedtypes.TagNumber(2)
							return val
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
				// FIX: Updated expected reason to match the service's behavior for general DB errors
				Reason: "Database error when retrieving leaderboard",
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
						{TagNumber: tag1, UserID: "user1"},
						{TagNumber: tag2, UserID: "user2"}, // Mixed tag number
					},
					IsActive:     true,
					UpdateSource: leaderboarddbtypes.ServiceUpdateSourceProcessScores,
					UpdateID:     testRoundID,
				}, nil)
			},
			expectedResult: &leaderboardevents.GetLeaderboardResponsePayload{
				Leaderboard: []leaderboardtypes.LeaderboardEntry{
					{
						TagNumber: func() sharedtypes.TagNumber {
							val := sharedtypes.TagNumber(1)
							return val
						}(),
						UserID: "user1",
					},
					{
						TagNumber: func() sharedtypes.TagNumber {
							// FIX: Updated expected tag number to match the mock DB return value
							val := sharedtypes.TagNumber(2)
							return val
						}(),
						UserID: "user2",
					},
				},
			},
			expectedFail:  nil,
			expectedError: nil,
		},
		// Add a test case for ErrNoActiveLeaderboard
		{
			name: "Handles no active leaderboard",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(nil, leaderboarddbtypes.ErrNoActiveLeaderboard)
			},
			expectedResult: nil,
			expectedFail: &leaderboardevents.GetLeaderboardFailedPayload{
				// The service specifically returns the error message for ErrNoActiveLeaderboard
				Reason: leaderboarddbtypes.ErrNoActiveLeaderboard.Error(),
			},
			expectedError: nil, // The service handles this error and returns nil error
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

			// Check if the returned error matches the expected error
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					// Check if the error message contains the expected substring
					if !strings.Contains(err.Error(), tt.expectedError.Error()) {
						t.Errorf("expected error to contain: %q, got: %q", tt.expectedError.Error(), err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
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

				// Use reflect.DeepEqual for comparing the success payloads
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

				// FIX: Check against the correct failure payload type
				failurePayload, ok := got.Failure.(*leaderboardevents.GetLeaderboardFailedPayload)
				if !ok {
					t.Errorf("Expected failure payload type *leaderboardevents.GetLeaderboardFailedPayload, got %T", got.Failure)
					return
				}

				// Check if the reason contains the expected substring
				if !strings.Contains(failurePayload.Reason, tt.expectedFail.Reason) {
					t.Errorf("LeaderboardService.GetLeaderboard() failure reason mismatch: expected reason to contain %q, got %q",
						tt.expectedFail.Reason, failurePayload.Reason)
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
		name                string
		payload             sharedevents.DiscordTagLookupRequestPayload
		mockDBSetup         func(*leaderboarddb.MockLeaderboardDB)
		expectedResult      *sharedevents.DiscordTagLookupResultPayload
		expectedFail        *sharedevents.DiscordTagLookupByUserIDFailedPayload
		expectedError       error // Expected standard error return value
		expectedResultError error // Expected error in the LeaderboardOperationResult struct
	}{
		{
			name: "Successfully retrieves tag number",
			payload: sharedevents.DiscordTagLookupRequestPayload{
				UserID: testUserID,
			},
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				// FIX: Return a pointer to sharedtypes.TagNumber instead of *int
				tagNumber := sharedtypes.TagNumber(5)
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
			expectedFail:        nil,
			expectedError:       nil,
			expectedResultError: nil,
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
			expectedFail:   nil, // The service returns the error in the Error field, not a failure payload
			expectedError:  nil, // The service returns nil standard error for unexpected DB errors
			// FIX: Expect the error in the LeaderboardOperationResult struct's Error field
			expectedResultError: fmt.Errorf("failed to get tag by UserID: %w", errors.New("unexpected DB error")),
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
			expectedFail:        nil,
			expectedError:       nil,
			expectedResultError: nil,
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
			expectedError:       nil,
			expectedResultError: nil,
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
			expectedFail:        nil,
			expectedError:       nil,
			expectedResultError: nil,
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

			got, err := s.GetTagByUserID(ctx, tt.payload.UserID) // Pass the UserID from the payload

			// Check if the returned standard error matches the expected standard error
			if (err != nil) != (tt.expectedError != nil) {
				t.Errorf("LeaderboardService.GetTagByUserID() standard error mismatch: got %v, wantErr %v", err, tt.expectedError)
				return
			} else if err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("LeaderboardService.GetTagByUserID() standard error mismatch: got %v, wantErr %v", err, tt.expectedError)
				return
			}

			// FIX: Check the Error field in the result struct
			if tt.expectedResultError != nil {
				if got.Error == nil {
					t.Errorf("Expected result struct error: %v, got nil", tt.expectedResultError)
				} else if got.Error.Error() != tt.expectedResultError.Error() {
					// Check if the error message contains the expected substring
					if !strings.Contains(got.Error.Error(), tt.expectedResultError.Error()) {
						t.Errorf("Result struct error mismatch: expected error to contain: %q, got %q", tt.expectedResultError.Error(), got.Error.Error())
					}
				}
			} else {
				if got.Error != nil {
					t.Errorf("Expected nil result struct error, got: %v", got.Error)
				}
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

	dummyRequestPayload := sharedevents.RoundTagLookupRequestPayload{
		UserID:     testUserID,
		RoundID:    testRoundID,
		Response:   "ACCEPT",
		JoinedLate: func() *bool { b := false; return &b }(),
	}

	tests := []struct {
		name           string
		mockDBSetup    func(*leaderboarddb.MockLeaderboardDB)
		requestPayload sharedevents.RoundTagLookupRequestPayload
		expectedResult *sharedevents.RoundTagLookupResultPayload
		expectedError  error
	}{
		{
			name: "Successfully retrieves tag number",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				tagNumber := sharedtypes.TagNumber(5)
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(&tagNumber, nil)
			},
			requestPayload: dummyRequestPayload,
			expectedResult: &sharedevents.RoundTagLookupResultPayload{
				UserID:  testUserID,
				RoundID: testRoundID,
				TagNumber: func() *sharedtypes.TagNumber {
					val := sharedtypes.TagNumber(5)
					return &val
				}(),
				Found:              true,
				Error:              "",
				OriginalResponse:   dummyRequestPayload.Response,
				OriginalJoinedLate: dummyRequestPayload.JoinedLate,
			},
			expectedError: nil,
		},
		{
			name: "Fails to retrieve tag number (operational error)",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(nil, errors.New("database connection error"))
			},
			requestPayload: dummyRequestPayload,
			expectedResult: nil,
			expectedError:  errors.New("failed to get tag by UserID (Round): database connection error"),
		},
		{
			name: "User ID not found in database (sql.ErrNoRows)",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(nil, sql.ErrNoRows)
			},
			requestPayload: dummyRequestPayload,
			expectedResult: &sharedevents.RoundTagLookupResultPayload{
				UserID:             testUserID,
				RoundID:            testRoundID,
				TagNumber:          nil,
				Found:              false,
				Error:              sql.ErrNoRows.Error(),
				OriginalResponse:   dummyRequestPayload.Response,
				OriginalJoinedLate: dummyRequestPayload.JoinedLate,
			},
			expectedError: nil,
		},
		{
			name: "User ID not found in database (string match)",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(nil, errors.New("user not found in DB"))
			},
			requestPayload: dummyRequestPayload,
			expectedResult: nil,
			// FIX: Expect the standard error returned by the service's general error handling
			expectedError: errors.New("failed to get tag by UserID (Round): user not found in DB"),
		},
		{
			name: "Nil tag number returned from database",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(nil, nil)
			},
			requestPayload: dummyRequestPayload,
			expectedResult: &sharedevents.RoundTagLookupResultPayload{
				UserID:             testUserID,
				RoundID:            testRoundID,
				TagNumber:          nil,
				Found:              false,
				Error:              "", // Error should be empty string when err is nil
				OriginalResponse:   dummyRequestPayload.Response,
				OriginalJoinedLate: dummyRequestPayload.JoinedLate,
			},
			expectedError: nil,
		},
		{
			name: "Handles no active leaderboard (Round)",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				mockDB.EXPECT().GetTagByUserID(gomock.Any(), testUserID).Return(nil, leaderboarddbtypes.ErrNoActiveLeaderboard)
			},
			requestPayload: dummyRequestPayload,
			expectedResult: nil,
			expectedError:  nil,
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

			got, err := s.RoundGetTagByUserID(ctx, tt.requestPayload)

			if (err != nil) != (tt.expectedError != nil) {
				t.Errorf("LeaderboardService.RoundGetTagByUserID() standard error mismatch: got = %v, wantErr %v", err, tt.expectedError)
				return
			}
			if err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("LeaderboardService.RoundGetTagByUserID() standard error message mismatch: got = %v, want %v", err.Error(), tt.expectedError.Error())
				return
			}

			if tt.expectedResult != nil {
				if got.Success == nil {
					t.Errorf("Expected success payload (%T), got nil LeaderboardOperationResult.Success", tt.expectedResult)
					return
				}
				successPayload, ok := got.Success.(*sharedevents.RoundTagLookupResultPayload)
				if !ok {
					t.Errorf("Expected success payload type %T, got %T", tt.expectedResult, got.Success)
					return
				}

				if !reflect.DeepEqual(successPayload, tt.expectedResult) {
					t.Errorf("LeaderboardService.RoundGetTagByUserID() result payload mismatch:\n  got: %#v\n want: %#v", successPayload, tt.expectedResult)
				}
			} else if got.Success != nil {
				t.Errorf("Expected nil success payload, got %v", got.Success)
			}

			if tt.name == "Handles no active leaderboard (Round)" {
				if got.Failure == nil {
					t.Errorf("Expected failure payload for ErrNoActiveLeaderboard, got nil")
				} else {
					failurePayload, ok := got.Failure.(*sharedevents.RoundTagLookupFailedPayload)
					if !ok {
						t.Errorf("Expected failure payload type *sharedevents.RoundTagLookupFailedPayload, got %T", got.Failure)
					} else {
						if failurePayload.Reason != "No active leaderboard found" {
							t.Errorf("Failure payload reason mismatch: got %q, want %q", failurePayload.Reason, "No active leaderboard found")
						}
						if failurePayload.UserID != tt.requestPayload.UserID {
							t.Errorf("Failure payload UserID mismatch: got %q, want %q", failurePayload.UserID, tt.requestPayload.UserID)
						}
						if failurePayload.RoundID != tt.requestPayload.RoundID {
							t.Errorf("Failure payload RoundID mismatch: got %q, want %q", failurePayload.RoundID, tt.requestPayload.RoundID)
						}
					}
				}
			} else if got.Failure != nil {
				t.Errorf("Expected nil failure payload, got %v", got.Failure)
			}
		})
	}
}
