package scoreservice

import (
	"context"
	"errors"
	"reflect"
	"testing"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories/mocks"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestScoreService_ProcessRoundScores(t *testing.T) {
	ctx := context.Background()
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testScore := sharedtypes.Score(10)
	testTag := sharedtypes.TagNumber(1)

	// Use No-Op implementations
	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &scoremetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

	// Define test cases
	tests := []struct {
		name           string
		mockDBSetup    func(*scoredb.MockScoreDB)
		scores         []sharedtypes.ScoreInfo
		expectedResult ScoreOperationResult
		expectedError  error
	}{
		{
			name: "Successfully processes round scores",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				mockDB.EXPECT().
					LogScores(gomock.Any(), testRoundID, gomock.Any(), "auto").
					DoAndReturn(func(ctx context.Context, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, source string) error {
						return nil
					})
			},
			scores: []sharedtypes.ScoreInfo{
				{
					UserID:    testUserID,
					Score:     testScore,
					TagNumber: &testTag,
				},
			},
			expectedResult: ScoreOperationResult{
				Success: &scoreevents.ProcessRoundScoresSuccessPayload{
					RoundID: testRoundID,
					TagMappings: []sharedtypes.TagMapping{
						{
							DiscordID: testUserID,
							TagNumber: testTag,
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "Fails due to database error",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				mockDB.EXPECT().
					LogScores(gomock.Any(), testRoundID, gomock.Any(), "auto").
					Return(errors.New("database connection failed"))
			},
			scores: []sharedtypes.ScoreInfo{
				{
					UserID:    testUserID,
					Score:     testScore,
					TagNumber: &testTag,
				},
			},
			expectedResult: ScoreOperationResult{
				Error: errors.New("database connection failed"),
			},
			expectedError: errors.New("database connection failed"),
		},
		{
			name: "Fails due to invalid round ID",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				mockDB.EXPECT().
					LogScores(gomock.Any(), testRoundID, gomock.Any(), "auto").
					Return(errors.New("invalid round ID"))
			},
			scores: []sharedtypes.ScoreInfo{
				{
					UserID:    testUserID,
					Score:     testScore,
					TagNumber: &testTag,
				},
			},
			expectedResult: ScoreOperationResult{
				Error: errors.New("invalid round ID"),
			},
			expectedError: errors.New("invalid round ID"),
		},
		{
			name: "Fails with empty score list",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				// No DB expectations because we should fail before calling DB
			},
			scores: []sharedtypes.ScoreInfo{},
			expectedResult: ScoreOperationResult{
				Error: errors.New("cannot process empty score list"),
			},
			expectedError: errors.New("cannot process empty score list"),
		},
		{
			name: "Handles extreme score values",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				mockDB.EXPECT().
					LogScores(gomock.Any(), testRoundID, gomock.Any(), "auto").
					DoAndReturn(func(ctx context.Context, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, source string) error {
						return nil
					})
			},
			scores: []sharedtypes.ScoreInfo{
				{
					UserID: sharedtypes.DiscordID("user1"),
					Score:  sharedtypes.Score(150), // Extremely high score
				},
				{
					UserID: sharedtypes.DiscordID("user2"),
					Score:  sharedtypes.Score(-150), // Extremely low score
				},
				{
					UserID: sharedtypes.DiscordID("user3"),
					Score:  sharedtypes.Score(5), // Normal score
				},
			},
			expectedResult: ScoreOperationResult{
				Success: &scoreevents.ProcessRoundScoresSuccessPayload{
					RoundID:     testRoundID,
					TagMappings: []sharedtypes.TagMapping{},
				},
			},
			expectedError: nil,
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

			gotResult, err := s.ProcessRoundScores(ctx, testRoundID, tt.scores)

			// Validate result
			if !reflect.DeepEqual(gotResult, tt.expectedResult) {
				t.Errorf("❌ Mismatched result, got: %v, expected: %v", gotResult, tt.expectedResult)
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
