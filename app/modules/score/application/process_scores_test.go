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

func TestScoreService_ProcessRoundScores(t *testing.T) {
	ctx := context.Background()
	testGuildID := sharedtypes.GuildID("guild-1234")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testScore := sharedtypes.Score(10)
	testTag := sharedtypes.TagNumber(1)

	// Use No-Op implementations for logger and metrics for testing purposes.
	logger := loggerfrolfbot.NoOpLogger
	metrics := &scoremetrics.NoOpMetrics{}
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")

	// Define test cases for ProcessRoundScores function.
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
					LogScores(gomock.Any(), testGuildID, testRoundID, gomock.Any(), "auto").
					DoAndReturn(func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, source string) error {
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
					GuildID: testGuildID,
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
					LogScores(gomock.Any(), testGuildID, testRoundID, gomock.Any(), "auto").
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
				Failure: &scoreevents.ProcessRoundScoresFailurePayload{
					GuildID: testGuildID,
					RoundID: testRoundID,
					Error:   "database connection failed",
				},
			},
			expectedError: nil, // Corrected: The service returns nil error for this case
		},
		{
			name: "Fails due to invalid round ID",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				mockDB.EXPECT().
					LogScores(gomock.Any(), testGuildID, testRoundID, gomock.Any(), "auto").
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
				Failure: &scoreevents.ProcessRoundScoresFailurePayload{
					GuildID: testGuildID,
					RoundID: testRoundID,
					Error:   "invalid round ID",
				},
			},
			expectedError: nil, // Corrected: The service returns nil error for this case
		},
		{
			name: "Fails with empty score list",
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				// No DB expectations because the error for empty list should occur before DB call.
			},
			scores: []sharedtypes.ScoreInfo{},
			expectedResult: ScoreOperationResult{
				Failure: &scoreevents.ProcessRoundScoresFailurePayload{
					GuildID: testGuildID,
					RoundID: testRoundID, // RoundID will be passed through to the failure payload
					Error:   "cannot process empty score list",
				},
			},
			expectedError: nil, // Corrected: The service returns nil error for this case
		},
		{
			name: "Handles extreme score values (expects validation error)", // Renamed for clarity
			mockDBSetup: func(mockDB *scoredb.MockScoreDB) {
				// No DB expectations because an error is expected from ProcessScoresForStorage before LogScores is called.
			},
			scores: []sharedtypes.ScoreInfo{
				{
					UserID: sharedtypes.DiscordID("user1"),
					Score:  sharedtypes.Score(150), // This score is causing the "invalid score value" error.
				},
				{
					UserID: sharedtypes.DiscordID("user2"),
					Score:  sharedtypes.Score(-150),
				},
				{
					UserID: sharedtypes.DiscordID("user3"),
					Score:  sharedtypes.Score(5),
				},
			},
			expectedResult: ScoreOperationResult{
				// Expect a failure payload matching the one returned by ProcessScoresForStorage.
				Failure: &scoreevents.ProcessRoundScoresFailurePayload{
					GuildID: testGuildID,
					RoundID: testRoundID, // RoundID will be passed through to the failure payload
					Error:   "invalid score value: 150 for user user1. Score must be between -36 and 72",
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

			// Initialize service with No-Op implementations for dependencies.
			s := &ScoreService{
				ScoreDB: mockDB,
				logger:  logger,
				metrics: metrics,
				tracer:  tracer,
				// The serviceWrapper is mocked to directly call the serviceFunc for simplicity in tests.
				serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (ScoreOperationResult, error)) (ScoreOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			// Set up mock expectations specific to the current test case.
			tt.mockDBSetup(mockDB)

			// Call the function under test.
			gotResult, err := s.ProcessRoundScores(ctx, testGuildID, testRoundID, tt.scores)

			// Validate the returned result.
			if (gotResult.Success != nil && tt.expectedResult.Success == nil) || (gotResult.Success == nil && tt.expectedResult.Success != nil) {
				t.Errorf("Mismatched result success, got: %v, expected: %v", gotResult.Success, tt.expectedResult.Success)
			} else if gotResult.Success != nil && tt.expectedResult.Success != nil {
				successGot, okGot := gotResult.Success.(*scoreevents.ProcessRoundScoresSuccessPayload)
				successExpected, okExpected := tt.expectedResult.Success.(*scoreevents.ProcessRoundScoresSuccessPayload)
				if okGot && okExpected {
					if successGot.GuildID != successExpected.GuildID {
						t.Errorf("Mismatched GuildID, got: %v, expected: %v", successGot.GuildID, successExpected.GuildID)
					}
					if successGot.RoundID != successExpected.RoundID {
						t.Errorf("Mismatched RoundID, got: %v, expected: %v", successGot.RoundID, successExpected.RoundID)
					}

					// Compare TagMappings slice.
					if len(successGot.TagMappings) != len(successExpected.TagMappings) {
						t.Errorf("Mismatched TagMappings length, got: %v, expected: %v", len(successGot.TagMappings), len(successExpected.TagMappings))
					} else {
						for i := range successGot.TagMappings {
							if successGot.TagMappings[i] != successExpected.TagMappings[i] {
								t.Errorf("Mismatched TagMapping at index %d, got: %v, expected: %v", i, successGot.TagMappings[i], successExpected.TagMappings[i])
							}
						}
					}
				}
			}

			// Validate the returned error.
			if (gotResult.Failure != nil && tt.expectedResult.Failure == nil) || (gotResult.Failure == nil && tt.expectedResult.Failure != nil) {
				t.Errorf("Mismatched result failure, got: %v, expected: %v", gotResult.Failure, tt.expectedResult.Failure)
			} else if gotResult.Failure != nil && tt.expectedResult.Failure != nil {
				failureGot, okGot := gotResult.Failure.(*scoreevents.ProcessRoundScoresFailurePayload)
				failureExpected, okExpected := tt.expectedResult.Failure.(*scoreevents.ProcessRoundScoresFailurePayload)
				if okGot && okExpected {
					if failureGot.GuildID != failureExpected.GuildID {
						t.Errorf("Mismatched GuildID, got: %v, expected: %v", failureGot.GuildID, failureExpected.GuildID)
					}
					if failureGot.RoundID != failureExpected.RoundID {
						t.Errorf("Mismatched RoundID, got: %v, expected: %v", failureGot.RoundID, failureExpected.RoundID)
					}
					if failureGot.Error != failureExpected.Error {
						t.Errorf("Mismatched error message in failure payload, got: %v, expected: %v", failureGot.Error, failureExpected.Error)
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
