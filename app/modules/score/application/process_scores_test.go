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
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestScoreService_ProcessRoundScores(t *testing.T) {
	ctx := context.Background()
	testMsg := message.NewMessage("test-id", nil)
	testRoundID := sharedtypes.RoundID(123)
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
		event          scoreevents.ProcessRoundScoresRequestPayload
		expectedResult ScoreOperationResult
		expectedError  error
	}{
		{
			name: "Successfully processes round scores",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				mockDB.EXPECT().
					LogScores(gomock.Any(), testRoundID, gomock.Any(), "auto").
					Return(nil)
			},
			event: scoreevents.ProcessRoundScoresRequestPayload{
				RoundID: testRoundID,
				Scores: []scoreevents.ParticipantScore{
					{
						UserID:    testUserID,
						Score:     testScore,
						TagNumber: testTag,
					},
				},
			},
			expectedResult: ScoreOperationResult{
				Success: []scoreevents.ParticipantScore{
					{
						UserID:    testUserID,
						Score:     testScore,
						TagNumber: testTag,
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
			event: scoreevents.ProcessRoundScoresRequestPayload{
				RoundID: testRoundID,
				Scores: []scoreevents.ParticipantScore{
					{
						UserID:    testUserID,
						Score:     testScore,
						TagNumber: testTag,
					},
				},
			},
			expectedResult: ScoreOperationResult{
				Failure: errors.New("database connection failed"),
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
			event: scoreevents.ProcessRoundScoresRequestPayload{
				RoundID: testRoundID,
				Scores: []scoreevents.ParticipantScore{
					{
						UserID:    testUserID,
						Score:     testScore,
						TagNumber: testTag,
					},
				},
			},
			expectedResult: ScoreOperationResult{
				Failure: errors.New("invalid round ID"),
			},
			expectedError: errors.New("invalid round ID"),
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

			gotResult, err := s.ProcessRoundScores(ctx, testMsg, tt.event)

			// Validate result
			if gotResult.Failure != nil && tt.expectedResult.Failure != nil {
				if !reflect.DeepEqual(gotResult.Failure, tt.expectedResult.Failure) {
					t.Errorf("❌ Mismatched error, got: %v, expected: %v", gotResult.Failure, tt.expectedResult.Failure)
				}
			} else if gotResult.Failure != nil || tt.expectedResult.Failure != nil {
				t.Errorf("❌ Mismatched error, got: %v, expected: %v", gotResult.Failure, tt.expectedResult.Failure)
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
