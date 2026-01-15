package scoreservice

import (
	"context"
	"errors"
	"testing"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
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
	testGuildID := sharedtypes.GuildID("guild-1234")
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
		expectedError  error // This should be nil if the service returns a Failure payload
	}{
		{
			name: "Successfully corrects score",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				mockDB.EXPECT().
					GetScoresForRound(gomock.Any(), testGuildID, testRoundID).
					Return([]sharedtypes.ScoreInfo{}, nil)
				mockDB.EXPECT().
					UpdateOrAddScore(gomock.Any(), testGuildID, testRoundID, sharedtypes.ScoreInfo{
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
				Success: &sharedevents.ScoreUpdatedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  testUserID,
					Score:   testScore,
				},
			},
			expectedError: nil,
		},
		{
			name: "Preserves existing tag when none provided",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				existingTag := sharedtypes.TagNumber(7)
				mockDB.EXPECT().
					GetScoresForRound(gomock.Any(), testGuildID, testRoundID).
					Return([]sharedtypes.ScoreInfo{{UserID: testUserID, Score: 12, TagNumber: &existingTag}}, nil)
				mockDB.EXPECT().
					UpdateOrAddScore(gomock.Any(), testGuildID, testRoundID, sharedtypes.ScoreInfo{
						UserID:    testUserID,
						Score:     testScore,
						TagNumber: &existingTag,
					}).
					Return(nil)
			},
			userID:    testUserID,
			score:     testScore,
			tagNumber: nil,
			expectedResult: ScoreOperationResult{
				Success: &sharedevents.ScoreUpdatedPayloadV1{
					GuildID: testGuildID,
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
					UpdateOrAddScore(gomock.Any(), testGuildID, testRoundID, sharedtypes.ScoreInfo{
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
				Success: &sharedevents.ScoreUpdatedPayloadV1{
					GuildID: testGuildID,
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
					GetScoresForRound(gomock.Any(), testGuildID, testRoundID).
					Return([]sharedtypes.ScoreInfo{}, nil)
				mockDB.EXPECT().
					UpdateOrAddScore(gomock.Any(), testGuildID, testRoundID, sharedtypes.ScoreInfo{
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
				Failure: &sharedevents.ScoreUpdateFailedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  testUserID,
					Reason:  "database connection failed",
				},
			},
			expectedError: nil, // Corrected: The service returns nil error for this case
		},
		{
			name: "Fails due to invalid tag number",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				mockDB.EXPECT().
					UpdateOrAddScore(gomock.Any(), testGuildID, testRoundID, sharedtypes.ScoreInfo{
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
				Failure: &sharedevents.ScoreUpdateFailedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  testUserID,
					Reason:  "invalid tag number",
				},
			},
			expectedError: nil, // Corrected: The service returns nil error for this case
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
					// In test, simply call the service function directly without wrapping
					return serviceFunc(ctx)
				},
			}

			tt.mockDBSetup(mockDB)

			gotResult, err := s.CorrectScore(ctx, testGuildID, testRoundID, tt.userID, tt.score, tt.tagNumber)

			// Validate result
			if (gotResult.Success != nil && tt.expectedResult.Success == nil) || (gotResult.Success == nil && tt.expectedResult.Success != nil) {
				t.Errorf("Mismatched result success, got: %v, expected: %v", gotResult.Success, tt.expectedResult.Success)
			} else if gotResult.Success != nil && tt.expectedResult.Success != nil {
				successGot, okGot := gotResult.Success.(*sharedevents.ScoreUpdatedPayloadV1)
				successExpected, okExpected := tt.expectedResult.Success.(*sharedevents.ScoreUpdatedPayloadV1)
				if okGot && okExpected {
					if successGot.GuildID != successExpected.GuildID {
						t.Errorf("Mismatched GuildID, got: %v, expected: %v", successGot.GuildID, successExpected.GuildID)
					}
					if successGot.RoundID != successExpected.RoundID {
						t.Errorf("Mismatched RoundID, got: %v, expected: %v", successGot.RoundID, successExpected.RoundID)
					}
					if successGot.UserID != successExpected.UserID {
						t.Errorf("Mismatched UserID, got: %v, expected: %v", successGot.UserID, successExpected.UserID)
					}
					if successGot.Score != successExpected.Score {
						t.Errorf("Mismatched Score, got: %v, expected: %v", successGot.Score, successExpected.Score)
					}
				}
			}

			if (gotResult.Failure != nil && tt.expectedResult.Failure == nil) || (gotResult.Failure == nil && tt.expectedResult.Failure != nil) {
				t.Errorf("Mismatched result failure, got: %v, expected: %v", gotResult.Failure, tt.expectedResult.Failure)
			} else if gotResult.Failure != nil && tt.expectedResult.Failure != nil {
				failureGot, okGot := gotResult.Failure.(*sharedevents.ScoreUpdateFailedPayloadV1)
				failureExpected, okExpected := tt.expectedResult.Failure.(*sharedevents.ScoreUpdateFailedPayloadV1)
				if okGot && okExpected {
					if failureGot.GuildID != failureExpected.GuildID {
						t.Errorf("Mismatched GuildID, got: %v, expected: %v", failureGot.GuildID, failureExpected.GuildID)
					}
					if failureGot.RoundID != failureExpected.RoundID {
						t.Errorf("Mismatched RoundID, got: %v, expected: %v", failureGot.RoundID, failureExpected.RoundID)
					}
					if failureGot.UserID != failureExpected.UserID {
						t.Errorf("Mismatched UserID, got: %v, expected: %v", failureGot.UserID, failureExpected.UserID)
					}
					if failureGot.Reason != failureExpected.Reason {
						t.Errorf("Mismatched error message, got: %v, expected: %v", failureGot.Reason, failureExpected.Reason)
					}
				}
			}

			// Validate error
			if (err != nil) != (tt.expectedError != nil) {
				t.Errorf("Unexpected error: %v", err)
			} else if err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("Mismatched error message, got: %v, expected: %v", err.Error(), tt.expectedError.Error())
			}
		})
	}
}
