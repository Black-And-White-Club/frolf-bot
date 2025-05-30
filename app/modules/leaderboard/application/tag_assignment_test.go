package leaderboardservice

// import (
// 	"context"
// 	"errors"
// 	"fmt"
// 	"testing"

// 	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
// 	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
// 	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
// 	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
// 	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
// 	leaderboarddbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
// 	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
// 	"github.com/google/uuid"
// 	"go.opentelemetry.io/otel/trace/noop"
// 	"go.uber.org/mock/gomock"
// )

// func TestLeaderboardService_TagAssignmentRequested(t *testing.T) {
// 	tagNumber := sharedtypes.TagNumber(42)
// 	testRoundID := sharedtypes.RoundID(uuid.New())
// 	testAssignmentID := sharedtypes.RoundID(uuid.New())

// 	tests := []struct {
// 		name           string
// 		mockDBSetup    func(*leaderboarddb.MockLeaderboardDB)
// 		payload        leaderboardevents.TagAssignmentRequestedPayload
// 		expectedResult interface{}
// 		expectedFail   *leaderboardevents.TagAssignmentFailedPayload
// 		expectedError  error
// 	}{
// 		{
// 			name: "Successfully assigns tag to user",
// 			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
// 				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{}, nil)
// 				mockDB.EXPECT().AssignTag(
// 					gomock.Any(),
// 					sharedtypes.DiscordID("test_user_id"),
// 					sharedtypes.TagNumber(42),
// 					"create_user",
// 					testRoundID,
// 					sharedtypes.DiscordID("test_user_id"),
// 				).Return(testAssignmentID, nil)
// 			},
// 			payload: leaderboardevents.TagAssignmentRequestedPayload{
// 				UserID:     sharedtypes.DiscordID("test_user_id"),
// 				TagNumber:  &tagNumber,
// 				Source:     "create_user",
// 				UpdateType: "",
// 				UpdateID:   testRoundID,
// 			},
// 			expectedResult: &leaderboardevents.TagAssignedPayload{
// 				UserID:       sharedtypes.DiscordID("test_user_id"),
// 				TagNumber:    &tagNumber,
// 				AssignmentID: testAssignmentID,
// 				Source:       "create_user",
// 			},
// 			expectedFail:  nil,
// 			expectedError: nil,
// 		},
// 		{
// 			name: "Triggers tag swap when user with tag claims another taken tag",
// 			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
// 				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
// 					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
// 						{UserID: "requestor", TagNumber: 1},
// 						{UserID: "target", TagNumber: 2},
// 					},
// 				}, nil)
// 			},
// 			payload: leaderboardevents.TagAssignmentRequestedPayload{
// 				UserID:     sharedtypes.DiscordID("requestor"),
// 				TagNumber:  ptrTag(2),
// 				Source:     "create_user",
// 				UpdateType: "",
// 				UpdateID:   testRoundID,
// 			},
// 			expectedResult: &leaderboardevents.TagSwapRequestedPayload{
// 				RequestorID: "requestor",
// 				TargetID:    "target",
// 			},
// 			expectedFail:  nil,
// 			expectedError: nil,
// 		},
// 		{
// 			name: "Fails to assign tag to user due to database error",
// 			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
// 				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{}, nil)
// 				mockDB.EXPECT().AssignTag(
// 					gomock.Any(),
// 					sharedtypes.DiscordID("test_user_id"),
// 					sharedtypes.TagNumber(42),
// 					"create_user",
// 					testRoundID,
// 					sharedtypes.DiscordID("test_user_id"),
// 				).Return(sharedtypes.RoundID(uuid.Nil), errors.New("database error"))
// 			},
// 			payload: leaderboardevents.TagAssignmentRequestedPayload{
// 				UserID:     sharedtypes.DiscordID("test_user_id"),
// 				TagNumber:  &tagNumber,
// 				Source:     "create_user",
// 				UpdateType: "",
// 				UpdateID:   testRoundID,
// 			},
// 			expectedResult: nil,
// 			expectedFail: &leaderboardevents.TagAssignmentFailedPayload{
// 				UserID:     sharedtypes.DiscordID("test_user_id"),
// 				TagNumber:  &tagNumber,
// 				Source:     "create_user",
// 				UpdateType: "",
// 				Reason:     "Database error during tag assignment: database error",
// 			},
// 			expectedError: fmt.Errorf("failed to assign tag in database: %w", errors.New("database error")),
// 		},
// 		{
// 			name: "Fails to assign tag to user due to invalid input",
// 			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
// 				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{}, nil)
// 			},
// 			payload: leaderboardevents.TagAssignmentRequestedPayload{
// 				UserID:     sharedtypes.DiscordID("test_user_id"),
// 				TagNumber:  ptrTag(-1),
// 				Source:     "create_user",
// 				UpdateType: "",
// 				UpdateID:   testRoundID,
// 			},
// 			expectedResult: nil,
// 			expectedFail: &leaderboardevents.TagAssignmentFailedPayload{
// 				UserID:     sharedtypes.DiscordID("test_user_id"),
// 				TagNumber:  nil,
// 				Source:     "create_user",
// 				UpdateType: "",
// 				Reason:     "invalid input: tag number cannot be negative",
// 			},
// 			expectedError: nil,
// 		},
// 		{
// 			name: "Fails to assign tag to user due to GetActiveLeaderboard error",
// 			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
// 				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(nil, errors.New("database error"))
// 			},
// 			payload: leaderboardevents.TagAssignmentRequestedPayload{
// 				UserID:     sharedtypes.DiscordID("test_user_id"),
// 				TagNumber:  &tagNumber,
// 				Source:     "create_user",
// 				UpdateType: "",
// 				UpdateID:   testRoundID,
// 			},
// 			expectedResult: nil,
// 			expectedFail: &leaderboardevents.TagAssignmentFailedPayload{
// 				UserID:     sharedtypes.DiscordID("test_user_id"),
// 				TagNumber:  &tagNumber,
// 				Source:     "create_user",
// 				UpdateType: "",
// 				Reason:     "Failed to get active leaderboard: database error",
// 			},
// 			expectedError: fmt.Errorf("failed to get active leaderboard: %w", errors.New("database error")),
// 		},
// 		{
// 			name: "Successfully updates tag for existing user",
// 			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
// 				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
// 					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
// 						{UserID: "test_user_id", TagNumber: 10},
// 					},
// 				}, nil)
// 				mockDB.EXPECT().AssignTag(
// 					gomock.Any(),
// 					sharedtypes.DiscordID("test_user_id"),
// 					sharedtypes.TagNumber(42),
// 					"create_user",
// 					testRoundID,
// 					sharedtypes.DiscordID("test_user_id"),
// 				).Return(testAssignmentID, nil)
// 			},
// 			payload: leaderboardevents.TagAssignmentRequestedPayload{
// 				UserID:     sharedtypes.DiscordID("test_user_id"),
// 				TagNumber:  &tagNumber,
// 				Source:     "create_user",
// 				UpdateType: "update",
// 				UpdateID:   testRoundID,
// 			},
// 			expectedResult: &leaderboardevents.TagAssignedPayload{
// 				UserID:       sharedtypes.DiscordID("test_user_id"),
// 				TagNumber:    &tagNumber,
// 				AssignmentID: testAssignmentID,
// 				Source:       "create_user",
// 			},
// 			expectedFail:  nil,
// 			expectedError: nil,
// 		},
// 		{
// 			name: "Fails to update tag for non-existent user",
// 			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
// 				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{}, nil)
// 			},
// 			payload: leaderboardevents.TagAssignmentRequestedPayload{
// 				UserID:     sharedtypes.DiscordID("non_existent_user"),
// 				TagNumber:  &tagNumber,
// 				Source:     "create_user",
// 				UpdateType: "update",
// 				UpdateID:   testRoundID,
// 			},
// 			expectedResult: nil,
// 			expectedFail: &leaderboardevents.TagAssignmentFailedPayload{
// 				UserID:     sharedtypes.DiscordID("non_existent_user"),
// 				TagNumber:  &tagNumber,
// 				Source:     "create_user",
// 				UpdateType: "update",
// 				Reason:     "Cannot update tag for user that doesn't exist in leaderboard",
// 			},
// 			expectedError: fmt.Errorf("cannot update tag for non-existent user"),
// 		},
// 		{
// 			name: "Triggers tag swap during update when new tag conflicts with another user",
// 			mockDBSetup: func(mockDB *leaderboarddb.MockLeaderboardDB) {
// 				mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
// 					LeaderboardData: []leaderboardtypes.LeaderboardEntry{
// 						{UserID: "requestor", TagNumber: 1},
// 						{UserID: "target", TagNumber: 2},
// 					},
// 				}, nil)
// 			},
// 			payload: leaderboardevents.TagAssignmentRequestedPayload{
// 				UserID:     sharedtypes.DiscordID("requestor"),
// 				TagNumber:  ptrTag(2),
// 				Source:     "create_user",
// 				UpdateType: "update",
// 				UpdateID:   testRoundID,
// 			},
// 			expectedResult: &leaderboardevents.TagSwapRequestedPayload{
// 				RequestorID: "requestor",
// 				TargetID:    "target",
// 			},
// 			expectedFail:  nil,
// 			expectedError: nil,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			ctrl := gomock.NewController(t)
// 			defer ctrl.Finish()

// 			mockDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
// 			tt.mockDBSetup(mockDB)

// 			logger := loggerfrolfbot.NoOpLogger
// 			tracerProvider := noop.NewTracerProvider()
// 			tracer := tracerProvider.Tracer("test")
// 			metrics := &leaderboardmetrics.NoOpMetrics{}

// 			s := &LeaderboardService{
// 				LeaderboardDB: mockDB,
// 				logger:        logger,
// 				metrics:       metrics,
// 				tracer:        tracer,
// 				serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
// 					return serviceFunc(ctx)
// 				},
// 			}

// 			ctx := context.Background()

// 			got, err := s.TagAssignmentRequested(ctx, tt.payload)

// 			// Validate success case (now supports TagAssignedPayload and TagSwapRequestedPayload)
// 			if tt.expectedResult != nil {
// 				if got.Success == nil {
// 					t.Errorf("❌ Expected success payload, got nil")
// 				} else {
// 					switch expected := tt.expectedResult.(type) {
// 					case *leaderboardevents.TagAssignedPayload:
// 						successPayload, ok := got.Success.(*leaderboardevents.TagAssignedPayload)
// 						if !ok {
// 							t.Errorf("❌ Expected Success to be *TagAssignedPayload, but got %T", got.Success)
// 						} else if successPayload.UserID != expected.UserID {
// 							t.Errorf("❌ Mismatched User ID, got: %v, expected: %v", successPayload.UserID, expected.UserID)
// 						} else if (successPayload.TagNumber == nil && expected.TagNumber != nil) ||
// 							(successPayload.TagNumber != nil && expected.TagNumber == nil) ||
// 							(successPayload.TagNumber != nil && expected.TagNumber != nil && *successPayload.TagNumber != *expected.TagNumber) {
// 							t.Errorf("❌ Mismatched Tag Number, got: %v, expected: %v", successPayload.TagNumber, expected.TagNumber)
// 						} else if successPayload.AssignmentID != expected.AssignmentID {
// 							t.Errorf("❌ Mismatched Assignment ID, got: %v, expected: %v", successPayload.AssignmentID, expected.AssignmentID)
// 						} else if successPayload.Source != expected.Source {
// 							t.Errorf("❌ Mismatched Source, got: %v, expected: %v", successPayload.Source, expected.Source)
// 						}
// 					case *leaderboardevents.TagSwapRequestedPayload:
// 						swapPayload, ok := got.Success.(*leaderboardevents.TagSwapRequestedPayload)
// 						if !ok {
// 							t.Errorf("❌ Expected Success to be *TagSwapRequestedPayload, but got %T", got.Success)
// 						} else if swapPayload.RequestorID != expected.RequestorID || swapPayload.TargetID != expected.TargetID {
// 							t.Errorf("❌ Mismatched swap IDs, got: %v/%v, expected: %v/%v", swapPayload.RequestorID, swapPayload.TargetID, expected.RequestorID, expected.TargetID)
// 						}
// 					default:
// 						t.Errorf("❌ Unexpected Success payload type: %T", got.Success)
// 					}
// 				}
// 			} else if got.Success != nil {
// 				t.Errorf("❌ Unexpected success payload: %v", got.Success)
// 			}

// 			// Validate failure case
// 			if tt.expectedFail != nil {
// 				if got.Failure == nil {
// 					t.Errorf("❌ Expected failure payload, got nil")
// 				} else {
// 					failurePayload, ok := got.Failure.(*leaderboardevents.TagAssignmentFailedPayload)
// 					if !ok {
// 						t.Errorf("❌ Expected Failure to be *TagAssignmentFailedPayload, but got %T", got.Failure)
// 					} else if failurePayload.UserID != tt.expectedFail.UserID {
// 						t.Errorf("❌ Mismatched User ID, got: %v, expected: %v", failurePayload.UserID, tt.expectedFail.UserID)
// 					} else if (failurePayload.TagNumber == nil && tt.expectedFail.TagNumber != nil) ||
// 						(failurePayload.TagNumber != nil && tt.expectedFail.TagNumber == nil) ||
// 						(failurePayload.TagNumber != nil && tt.expectedFail.TagNumber != nil && *failurePayload.TagNumber != *tt.expectedFail.TagNumber) {
// 						t.Errorf("❌ Mismatched Tag Number, got: %v, expected: %v", failurePayload.TagNumber, tt.expectedFail.TagNumber)
// 					} else if failurePayload.Source != tt.expectedFail.Source {
// 						t.Errorf("❌ Mismatched Source, got: %v, expected: %v", failurePayload.Source, tt.expectedFail.Source)
// 					} else if failurePayload.UpdateType != tt.expectedFail.UpdateType {
// 						t.Errorf("❌ Mismatched Update Type, got: %v, expected: %v", failurePayload.UpdateType, tt.expectedFail.UpdateType)
// 					} else if failurePayload.Reason != tt.expectedFail.Reason {
// 						t.Errorf("❌ Mismatched Reason, got: %v, expected: %v", failurePayload.Reason, tt.expectedFail.Reason)
// 					}
// 				}
// 			} else if got.Failure != nil {
// 				t.Errorf("❌ Unexpected failure payload: %v", got.Failure)
// 			}

// 			// Validate error presence
// 			if tt.expectedError != nil {
// 				if err == nil {
// 					t.Errorf("❌ Expected an error but got nil")
// 				} else if err.Error() != tt.expectedError.Error() {
// 					t.Errorf("❌ Mismatched error reason, got: %v, expected: %v", err.Error(), tt.expectedError.Error())
// 				}
// 			} else if err != nil {
// 				t.Errorf("❌ Unexpected error: %v", err)
// 			}
// 		})
// 	}
// }
