package scoreservice

import (
	"context"
	"errors"
	"testing"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestScoreService_CorrectScore(t *testing.T) {
	ctx := context.Background()
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testScore := sharedtypes.Score(10)
	testTag := sharedtypes.TagNumber(1)
	invalidTag := sharedtypes.TagNumber(-1)

	// Use No-Op implementations
	logger := loggerfrolfbot.NoOpLogger
	metrics := &scoremetrics.NoOpMetrics{}
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")

	// Define test cases
	tests := []struct {
		name           string
		mockDBSetup    func(*scoredb.MockScoreDB)
		userID         sharedtypes.DiscordID
		score          sharedtypes.Score
		tagNumber      *sharedtypes.TagNumber
		expectedResult ScoreOperationResult
		expectedError  error
	}{
		{
			name: "Successfully corrects score",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				mockDB.EXPECT().
					UpdateOrAddScore(gomock.Any(), testRoundID, sharedtypes.ScoreInfo{
						UserID:    testUserID,
						Score:     testScore,
						TagNumber: nil,
					}).
					Return(nil)
			},
			userID:    testUserID,
			score:     testScore,
			tagNumber: nil,
			expectedResult: ScoreOperationResult{
				Success: &scoreevents.ScoreUpdateSuccessPayload{
					RoundID: testRoundID,
					UserID:  testUserID,
					Score:   testScore,
				},
			},
			expectedError: nil,
		},
		{
			name: "Successfully corrects score with tag number",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				mockDB.EXPECT().
					UpdateOrAddScore(gomock.Any(), testRoundID, sharedtypes.ScoreInfo{
						UserID:    testUserID,
						Score:     testScore,
						TagNumber: &testTag,
					}).
					Return(nil)
			},
			userID:    testUserID,
			score:     testScore,
			tagNumber: &testTag,
			expectedResult: ScoreOperationResult{
				Success: &scoreevents.ScoreUpdateSuccessPayload{
					RoundID: testRoundID,
					UserID:  testUserID,
					Score:   testScore,
				},
			},
			expectedError: nil,
		},
		{
			name: "Fails due to database error",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				mockDB.EXPECT().
					UpdateOrAddScore(gomock.Any(), testRoundID, sharedtypes.ScoreInfo{
						UserID:    testUserID,
						Score:     testScore,
						TagNumber: nil,
					}).
					Return(errors.New("database connection failed"))
			},
			userID:    testUserID,
			score:     testScore,
			tagNumber: nil,
			expectedResult: ScoreOperationResult{
				Failure: &scoreevents.ScoreUpdateFailurePayload{
					RoundID: testRoundID,
					UserID:  testUserID,
					Error:   "database connection failed",
				},
				Error: errors.New("database connection failed"),
			},
			expectedError: errors.New("database connection failed"),
		},
		{
			name: "Fails due to invalid tag number",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				mockDB.EXPECT().
					UpdateOrAddScore(gomock.Any(), testRoundID, sharedtypes.ScoreInfo{
						UserID:    testUserID,
						Score:     testScore,
						TagNumber: &invalidTag,
					}).
					Return(errors.New("invalid tag number"))
			},
			userID:    testUserID,
			score:     testScore,
			tagNumber: &invalidTag,
			expectedResult: ScoreOperationResult{
				Failure: &scoreevents.ScoreUpdateFailurePayload{
					RoundID: testRoundID,
					UserID:  testUserID,
					Error:   "invalid tag number",
				},
				Error: errors.New("invalid tag number"),
			},
			expectedError: errors.New("invalid tag number"),
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := scoredb.NewMockScoreDB(ctrl)

			// Initialize service with No-Op implementations
			s := &ScoreService{
				ScoreDB: mockDB,
				logger:  logger,
				metrics: metrics,
				tracer:  tracer,
				serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (ScoreOperationResult, error)) (ScoreOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			tt.mockDBSetup(mockDB)

			gotResult, err := s.CorrectScore(ctx, testRoundID, tt.userID, tt.score, tt.tagNumber)

			// Validate result
			if (gotResult.Success != nil && tt.expectedResult.Success == nil) || (gotResult.Success == nil && tt.expectedResult.Success != nil) {
				t.Errorf("❌ Mismatched result success, got: %v, expected: %v", gotResult.Success, tt.expectedResult.Success)
			} else if gotResult.Success != nil && tt.expectedResult.Success != nil {
				successGot, okGot := gotResult.Success.(*scoreevents.ScoreUpdateSuccessPayload)
				successExpected, okExpected := tt.expectedResult.Success.(*scoreevents.ScoreUpdateSuccessPayload)
				if okGot && okExpected && *successGot != *successExpected {
					t.Errorf("❌ Mismatched success payload, got: %v, expected: %v", successGot, successExpected)
				}
			}

			if (gotResult.Failure != nil && tt.expectedResult.Failure == nil) || (gotResult.Failure == nil && tt.expectedResult.Failure != nil) {
				t.Errorf("❌ Mismatched result failure, got: %v, expected: %v", gotResult.Failure, tt.expectedResult.Failure)
			} else if gotResult.Failure != nil && tt.expectedResult.Failure != nil {
				failureGot, okGot := gotResult.Failure.(*scoreevents.ScoreUpdateFailurePayload)
				failureExpected, okExpected := tt.expectedResult.Failure.(*scoreevents.ScoreUpdateFailurePayload)
				if okGot && okExpected && *failureGot != *failureExpected {
					t.Errorf("❌ Mismatched failure payload, got: %v, expected: %v", failureGot, failureExpected)
				}
			}

			// Validate error
			if (err != nil) != (tt.expectedError != nil) {
				t.Errorf("❌ Unexpected error: %v", err)
			} else if err != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("❌ Mismatched error message, got: %v, expected: %v", err.Error(), tt.expectedError.Error())
			}
		})
	}
}
