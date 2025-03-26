package scoreservice

import (
	"context"
	"errors"
	"reflect"
	"testing"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/score"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	scoredbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestScoreService_CorrectScore(t *testing.T) {
	ctx := context.Background()
	testMsg := message.NewMessage("test-id", nil)
	testRoundID := sharedtypes.RoundID(123)
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testScore := sharedtypes.Score(10)
	testTag := sharedtypes.TagNumber(1)
	invalidTag := sharedtypes.TagNumber(-1)

	// Use No-Op implementations
	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &scoremetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

	// Define test cases
	tests := []struct {
		name           string
		mockDBSetup    func(*scoredb.MockScoreDB)
		newScore       sharedtypes.Score
		tagNumber      *sharedtypes.TagNumber
		expectedResult ScoreOperationResult
		expectedError  error
	}{
		{
			name: "Successfully corrects score",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				mockDB.EXPECT().
					UpdateOrAddScore(gomock.Any(), &scoredbtypes.Score{
						UserID:    testUserID,
						RoundID:   testRoundID,
						Score:     testScore,
						TagNumber: nil,
						Source:    "manual",
					}).
					Return(nil)
			},
			newScore:  testScore,
			tagNumber: nil,
			expectedResult: ScoreOperationResult{
				Success: &scoredbtypes.Score{
					UserID:    testUserID,
					RoundID:   testRoundID,
					Score:     testScore,
					TagNumber: nil,
					Source:    "manual",
				},
			},
			expectedError: nil,
		},
		{
			name: "Successfully corrects score with tag number",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				mockDB.EXPECT().
					UpdateOrAddScore(gomock.Any(), &scoredbtypes.Score{
						UserID:    testUserID,
						RoundID:   testRoundID,
						Score:     testScore,
						TagNumber: &testTag,
						Source:    "manual",
					}).
					Return(nil)
			},
			newScore:  testScore,
			tagNumber: &testTag,
			expectedResult: ScoreOperationResult{
				Success: &scoredbtypes.Score{
					UserID:    testUserID,
					RoundID:   testRoundID,
					Score:     testScore,
					TagNumber: &testTag,
					Source:    "manual",
				},
			},
			expectedError: nil,
		},
		{
			name: "Fails due to database error",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				mockDB.EXPECT().
					UpdateOrAddScore(gomock.Any(), &scoredbtypes.Score{
						UserID:    testUserID,
						RoundID:   testRoundID,
						Score:     testScore,
						TagNumber: nil,
						Source:    "manual",
					}).
					Return(errors.New("database connection failed"))
			},
			newScore:  testScore,
			tagNumber: nil,
			expectedResult: ScoreOperationResult{
				Failure: errors.New("database connection failed"),
			},
			expectedError: errors.New("database connection failed"),
		},
		{
			name: "Fails due to invalid round ID",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				mockDB.EXPECT().
					UpdateOrAddScore(gomock.Any(), gomock.Any()).
					Return(errors.New("invalid round ID"))
			},
			newScore:  testScore,
			tagNumber: nil,
			expectedResult: ScoreOperationResult{
				Failure: errors.New("invalid round ID"),
			},
			expectedError: errors.New("invalid round ID"),
		},
		{
			name: "Fails due to invalid user ID",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				mockDB.EXPECT().
					UpdateOrAddScore(gomock.Any(), gomock.Any()).
					Return(errors.New("invalid user ID"))
			},
			newScore:  testScore,
			tagNumber: nil,
			expectedResult: ScoreOperationResult{
				Failure: errors.New("invalid user ID"),
			},
			expectedError: errors.New("invalid user ID"),
		},
		{
			name: "Fails due to invalid tag number",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				mockDB.EXPECT().
					UpdateOrAddScore(gomock.Any(), gomock.Any()).
					Return(errors.New("invalid tag number"))
			},
			newScore:  testScore,
			tagNumber: &invalidTag,
			expectedResult: ScoreOperationResult{
				Failure: errors.New("invalid tag number"),
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
				serviceWrapper: func(msg *message.Message, operationName string, roundID sharedtypes.RoundID, serviceFunc func() (ScoreOperationResult, error)) (ScoreOperationResult, error) {
					return serviceFunc()
				},
			}

			tt.mockDBSetup(mockDB)

			event := scoreevents.ScoreUpdateRequestPayload{
				RoundID:   testRoundID,
				UserID:    testUserID,
				Score:     tt.newScore,
				TagNumber: tt.tagNumber,
			}

			gotResult, err := s.CorrectScore(ctx, testMsg, event)

			// Validate result
			if gotResult.Error != nil && tt.expectedResult.Error != nil {
				if !errors.Is(gotResult.Error, tt.expectedResult.Error) {
					t.Errorf("❌ Mismatched error, got: %v, expected: %v", gotResult.Error, tt.expectedResult.Error)
				}
			} else if gotResult.Error != nil || tt.expectedResult.Error != nil {
				t.Errorf("❌ Mismatched error, got: %v, expected: %v", gotResult.Error, tt.expectedResult.Error)
			}
			if gotResult.Success != nil && tt.expectedResult.Success != nil {
				if !reflect.DeepEqual(gotResult.Success, tt.expectedResult.Success) {
					t.Errorf("❌ Mismatched result, got: %v, expected: %v", gotResult.Success, tt.expectedResult.Success)
				}
			} else if gotResult.Success != nil || tt.expectedResult.Success != nil {
				t.Errorf("❌ Mismatched result, got: %v, expected: %v", gotResult.Success, tt.expectedResult.Success)
			}
			if gotResult.Failure != nil && tt.expectedResult.Failure != nil {
				if !reflect.DeepEqual(gotResult.Failure, tt.expectedResult.Failure) {
					t.Errorf("❌ Mismatched failure, got: %v, expected: %v", gotResult.Failure, tt.expectedResult.Failure)
				}
			} else if gotResult.Failure != nil || tt.expectedResult.Failure != nil {
				t.Errorf("❌ Mismatched failure, got: %v, expected: %v", gotResult.Failure, tt.expectedResult.Failure)
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
