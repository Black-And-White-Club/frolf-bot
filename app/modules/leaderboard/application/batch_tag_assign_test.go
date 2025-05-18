package leaderboardservice

import (
	"context"
	"errors"
	"strings" // Import strings for error message comparison
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared" // Import shared events
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_BatchTagAssignmentRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Mock dependencies
	mockDB := leaderboarddb.NewMockLeaderboardDB(ctrl)

	// No-Op implementations for logging, metrics, and tracing
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &leaderboardmetrics.NoOpMetrics{}

	// Helper function to convert sharedevents.TagAssignmentInfo to leaderboarddbtypes.TagAssignment
	convertAssignmentsForDB := func(assignments []sharedevents.TagAssignmentInfo) []leaderboarddbtypes.TagAssignment {
		dbAssignments := make([]leaderboarddbtypes.TagAssignment, 0, len(assignments))
		for _, assignment := range assignments {
			// Assuming the service filters invalid tags BEFORE calling the DB
			if assignment.TagNumber >= 0 {
				dbAssignments = append(dbAssignments, leaderboarddbtypes.TagAssignment{
					UserID:    assignment.UserID,
					TagNumber: assignment.TagNumber,
				})
			}
		}
		return dbAssignments
	}

	tests := []struct {
		name                   string
		mockDBSetup            func(*leaderboarddb.MockLeaderboardDB)
		payload                sharedevents.BatchTagAssignmentRequestedPayload    // Use shared events payload
		expectedSuccessPayload *leaderboardevents.BatchTagAssignedPayload         // Use concrete success payload type
		expectedFailurePayload *leaderboardevents.BatchTagAssignmentFailedPayload // Use concrete failure payload type
		expectedError          error
	}{
		{
			name: "Successfully assigns tags",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				// Expected assignments after filtering (none filtered in this case)
				expectedDBAssignments := convertAssignmentsForDB([]sharedevents.TagAssignmentInfo{
					{UserID: "user1", TagNumber: 1},
					{UserID: "user2", TagNumber: 2},
				})
				mockDB.EXPECT().
					BatchAssignTags(gomock.Any(), expectedDBAssignments, leaderboarddbtypes.ServiceUpdateSourceAdminBatch, gomock.Any(), sharedtypes.DiscordID("test_user_id")).
					Return(nil)
			},
			payload: sharedevents.BatchTagAssignmentRequestedPayload{ // Use shared events payload
				BatchID:          "batch-1",
				RequestingUserID: "test_user_id",
				Assignments: []sharedevents.TagAssignmentInfo{ // Use shared events TagAssignmentInfo
					{UserID: "user1", TagNumber: 1},
					{UserID: "user2", TagNumber: 2},
				},
			},
			expectedSuccessPayload: &leaderboardevents.BatchTagAssignedPayload{ // Use concrete success payload type
				RequestingUserID: "test_user_id",
				BatchID:          "batch-1",
				AssignmentCount:  2, // Count of attempted assignments
				Assignments: []leaderboardevents.TagAssignmentInfo{ // Outgoing is leaderboardevents
					{UserID: "user1", TagNumber: 1},
					{UserID: "user2", TagNumber: 2},
				},
			},
			expectedError: nil,
		},
		{
			name: "Invalid tag assignment",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				// Expected assignments after filtering (only user2 remains)
				expectedDBAssignments := convertAssignmentsForDB([]sharedevents.TagAssignmentInfo{
					{UserID: "user2", TagNumber: 2}, // Only valid assignment passed to DB
				})
				mockDB.EXPECT().
					BatchAssignTags(gomock.Any(), expectedDBAssignments, leaderboarddbtypes.ServiceUpdateSourceAdminBatch, gomock.Any(), sharedtypes.DiscordID("test_user_id")).
					Return(nil)
			},
			payload: sharedevents.BatchTagAssignmentRequestedPayload{ // Use shared events payload
				BatchID:          "batch-2",
				RequestingUserID: "test_user_id",
				Assignments: []sharedevents.TagAssignmentInfo{ // Use shared events TagAssignmentInfo
					{UserID: "user1", TagNumber: -1}, // Invalid tag number
					{UserID: "user2", TagNumber: 2},
				},
			},
			expectedSuccessPayload: &leaderboardevents.BatchTagAssignedPayload{ // Outgoing is leaderboardevents
				RequestingUserID: "test_user_id",
				BatchID:          "batch-2",
				AssignmentCount:  2, // Count of attempted assignments (service includes all attempted in success payload)
				Assignments: []leaderboardevents.TagAssignmentInfo{ // Outgoing is leaderboardevents
					{UserID: "user1", TagNumber: -1}, // Service includes attempted assignments in success payload
					{UserID: "user2", TagNumber: 2},
				},
			},
			expectedError: nil,
		},
		{
			name: "Database error on batch assignment",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				// Expected assignments after filtering (none filtered)
				expectedDBAssignments := convertAssignmentsForDB([]sharedevents.TagAssignmentInfo{
					{UserID: "user1", TagNumber: 1},
				})
				mockDB.EXPECT().
					BatchAssignTags(gomock.Any(), expectedDBAssignments, leaderboarddbtypes.ServiceUpdateSourceAdminBatch, gomock.Any(), sharedtypes.DiscordID("test_user_id")).
					Return(errors.New("database error"))
			},
			payload: sharedevents.BatchTagAssignmentRequestedPayload{ // Use shared events payload
				BatchID:          "batch-3",
				RequestingUserID: "test_user_id",
				Assignments: []sharedevents.TagAssignmentInfo{ // Use shared events TagAssignmentInfo
					{UserID: "user1", TagNumber: 1},
				},
			},
			expectedFailurePayload: &leaderboardevents.BatchTagAssignmentFailedPayload{ // Outgoing is leaderboardevents
				RequestingUserID: "test_user_id",
				BatchID:          "batch-3",
				Reason:           "database error", // Expecting the exact error message from the mock
			},
			expectedError: errors.New("database error"), // Expecting the error to be returned
		},
		{
			name: "No valid assignments in batch",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				// BatchAssignTags should NOT be called if no valid assignments
				mockDB.EXPECT().BatchAssignTags(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			payload: sharedevents.BatchTagAssignmentRequestedPayload{ // Use shared events payload
				BatchID:          "batch-4",
				RequestingUserID: "test_user_id",
				Assignments: []sharedevents.TagAssignmentInfo{ // Use shared events TagAssignmentInfo
					{UserID: "user1", TagNumber: -1}, // Invalid tag number
					{UserID: "user2", TagNumber: -5}, // Invalid tag number
				},
			},
			expectedSuccessPayload: &leaderboardevents.BatchTagAssignedPayload{ // Outgoing is leaderboardevents
				RequestingUserID: "test_user_id",
				BatchID:          "batch-4",
				AssignmentCount:  0,                                       // Count of attempted assignments (0 valid processed)
				Assignments:      []leaderboardevents.TagAssignmentInfo{}, // Empty slice
			},
			expectedError: nil,
		},
		{
			name: "Empty assignments list",
			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
				// BatchAssignTags should NOT be called for empty list
				mockDB.EXPECT().BatchAssignTags(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			payload: sharedevents.BatchTagAssignmentRequestedPayload{ // Use shared events payload
				BatchID:          "batch-5",
				RequestingUserID: "test_user_id",
				Assignments:      []sharedevents.TagAssignmentInfo{}, // Empty list
			},
			expectedSuccessPayload: &leaderboardevents.BatchTagAssignedPayload{ // Outgoing is leaderboardevents
				RequestingUserID: "test_user_id",
				BatchID:          "batch-5",
				AssignmentCount:  0,                                       // Count of attempted assignments (0 valid processed)
				Assignments:      []leaderboardevents.TagAssignmentInfo{}, // Empty slice
			},
			expectedError: nil,
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockDBSetup(mockDB)

			// Initialize service with No-Op implementations and a wrapper that just calls the function
			s := &LeaderboardService{
				LeaderboardDB: mockDB,
				logger:        logger,
				metrics:       metrics,
				tracer:        tracer,
				serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			got, err := s.BatchTagAssignmentRequested(ctx, tt.payload)

			// Validate error presence
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
				// Validate the result based on whether it's a success or failure
				if tt.expectedSuccessPayload != nil {
					if got.Success == nil {
						t.Errorf("expected success result, got: nil")
					} else {
						// Type assert the received success payload
						successPayload, ok := got.Success.(*leaderboardevents.BatchTagAssignedPayload)
						if !ok {
							t.Errorf("expected result to be *leaderboardevents.BatchTagAssignedPayload, got: %T", got.Success)
						} else {
							// Deep compare the success payloads
							if successPayload.RequestingUserID != tt.expectedSuccessPayload.RequestingUserID {
								t.Errorf("Success payload RequestingUserID mismatch: expected %q, got %q",
									tt.expectedSuccessPayload.RequestingUserID, successPayload.RequestingUserID)
							}
							if successPayload.BatchID != tt.expectedSuccessPayload.BatchID {
								t.Errorf("Success payload BatchID mismatch: expected %q, got %q",
									tt.expectedSuccessPayload.BatchID, successPayload.BatchID)
							}
							if successPayload.AssignmentCount != tt.expectedSuccessPayload.AssignmentCount {
								t.Errorf("Success payload AssignmentCount mismatch: expected %d, got %d",
									tt.expectedSuccessPayload.AssignmentCount, successPayload.AssignmentCount)
							}
							// Compare assignments - order might not be guaranteed, so compare as sets/maps if necessary
							// For now, assuming order is preserved for simplicity in this unit test.
							if len(successPayload.Assignments) != len(tt.expectedSuccessPayload.Assignments) {
								t.Errorf("Success payload Assignments count mismatch: expected %d, got %d",
									len(tt.expectedSuccessPayload.Assignments), len(successPayload.Assignments))
							} else {
								for i := range successPayload.Assignments {
									if successPayload.Assignments[i].UserID != tt.expectedSuccessPayload.Assignments[i].UserID ||
										successPayload.Assignments[i].TagNumber != tt.expectedSuccessPayload.Assignments[i].TagNumber {
										t.Errorf("Success payload Assignment mismatch at index %d: expected %+v, got %+v",
											i, tt.expectedSuccessPayload.Assignments[i], successPayload.Assignments[i])
										break // Stop checking assignments after the first mismatch
									}
								}
							}
						}
					}
				} else if tt.expectedFailurePayload != nil {
					if got.Failure == nil {
						t.Errorf("expected failure result, got: nil")
					} else {
						// Type assert the received failure payload
						failurePayload, ok := got.Failure.(*leaderboardevents.BatchTagAssignmentFailedPayload)
						if !ok {
							t.Errorf("expected result to be *leaderboardevents.BatchTagAssignmentFailedPayload, got: %T", got.Failure)
						} else {
							// Compare failure payloads
							if failurePayload.RequestingUserID != tt.expectedFailurePayload.RequestingUserID {
								t.Errorf("Failure payload RequestingUserID mismatch: expected %q, got %q",
									tt.expectedFailurePayload.RequestingUserID, failurePayload.RequestingUserID)
							}
							if failurePayload.BatchID != tt.expectedFailurePayload.BatchID {
								t.Errorf("Failure payload BatchID mismatch: expected %q, got %q",
									tt.expectedFailurePayload.BatchID, failurePayload.BatchID)
							}
							// Check if the reason contains the expected substring
							if !strings.Contains(failurePayload.Reason, tt.expectedFailurePayload.Reason) {
								t.Errorf("Failure payload Reason mismatch: expected reason to contain %q, got %q",
									tt.expectedFailurePayload.Reason, failurePayload.Reason)
							}
						}
					}
				} else {
					// This case should not be reached if expectedSuccessPayload or expectedFailurePayload is properly defined
					t.Errorf("test case misconfigured: neither expectedSuccessPayload nor expectedFailurePayload is set")
				}
			}
		})
	}
}
