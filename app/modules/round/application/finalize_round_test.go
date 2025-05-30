package roundservice

import (
	"context"
	"errors"
	"fmt"
	"testing"

	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

// Helper to create a pointer to a value
func ptr[T any](v T) *T {
	return &v
}

func TestRoundService_FinalizeRound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)

	testRoundID := sharedtypes.RoundID(uuid.New())
	testFinalizedRound := &roundtypes.Round{
		ID:           testRoundID,
		State:        roundtypes.RoundStateFinalized,
		Participants: []roundtypes.Participant{},
	}
	dbUpdateError := errors.New("db update failed")
	dbGetError := errors.New("db get failed")

	tests := []struct {
		name                    string
		mockDBSetup             func(*rounddb.MockRoundDB)
		mockRoundValidatorSetup func(*roundutil.MockRoundValidator)
		mockEventBusSetup       func(*eventbus.MockEventBus)
		payload                 roundevents.AllScoresSubmittedPayload
		expectedResult          RoundOperationResult
		expectedError           error
	}{
		{
			name: "success finalizing round",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().UpdateRoundState(ctx, testRoundID, roundtypes.RoundStateFinalized).Return(nil)
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(testFinalizedRound, nil)
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
				// No expectations for the RoundValidator
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
			},
			payload: roundevents.AllScoresSubmittedPayload{
				RoundID: testRoundID,
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.RoundFinalizedPayload{
					RoundID:   testRoundID,
					RoundData: *testFinalizedRound,
				},
			},
			expectedError: nil,
		},
		{
			name: "failure updating round state",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().UpdateRoundState(ctx, testRoundID, roundtypes.RoundStateFinalized).Return(dbUpdateError)
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
				// No expectations for the RoundValidator
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
			},
			payload: roundevents.AllScoresSubmittedPayload{
				RoundID: testRoundID,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundFinalizationErrorPayload{
					RoundID: testRoundID,
					Error:   fmt.Sprintf("failed to update round state to finalized: %v", dbUpdateError),
				},
			},
			expectedError: nil, // Service returns failure payload with nil error
		},
		{
			name: "failure fetching round after update",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().UpdateRoundState(ctx, testRoundID, roundtypes.RoundStateFinalized).Return(nil)
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(nil, dbGetError)
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
				// No expectations for the RoundValidator
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
			},
			payload: roundevents.AllScoresSubmittedPayload{
				RoundID: testRoundID,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundFinalizationErrorPayload{
					RoundID: testRoundID,
					Error:   fmt.Sprintf("failed to fetch round data: %v", dbGetError),
				},
			},
			expectedError: nil, // Service returns failure payload with nil error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks for each subtest
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockDB := rounddb.NewMockRoundDB(ctrl)
			mockEventBus := eventbus.NewMockEventBus(ctrl)

			tt.mockDBSetup(mockDB)
			tt.mockRoundValidatorSetup(mockRoundValidator)
			tt.mockEventBusSetup(mockEventBus)

			s := &RoundService{
				RoundDB:        mockDB,
				logger:         logger,
				metrics:        mockMetrics,
				tracer:         tracer,
				roundValidator: mockRoundValidator,
				EventBus:       mockEventBus,
				serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			result, err := s.FinalizeRound(ctx, tt.payload)

			// Validate error
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}

			// Validate result payload
			if tt.expectedResult.Success != nil {
				if result.Success == nil {
					t.Errorf("expected success result, got failure")
				} else if successPayload, ok := result.Success.(*roundevents.RoundFinalizedPayload); !ok {
					t.Errorf("expected result.Success to be of type *roundevents.RoundFinalizedPayload, got %T", result.Success)
				} else if expectedSuccessPayload, ok := tt.expectedResult.Success.(*roundevents.RoundFinalizedPayload); !ok {
					t.Errorf("expected tt.expectedResult.Success to be of type *roundevents.RoundFinalizedPayload, got %T", tt.expectedResult.Success)
				} else if successPayload.RoundID != expectedSuccessPayload.RoundID {
					t.Errorf("expected success RoundID %s, got %s", expectedSuccessPayload.RoundID, successPayload.RoundID)
				}
			} else if tt.expectedResult.Failure != nil {
				if result.Failure == nil {
					t.Errorf("expected failure result, got success")
				} else if failurePayload, ok := result.Failure.(*roundevents.RoundFinalizationErrorPayload); !ok {
					t.Errorf("expected result.Failure to be of type *roundevents.RoundFinalizationErrorPayload, got %T", result.Failure)
				} else if expectedFailurePayload, ok := tt.expectedResult.Failure.(*roundevents.RoundFinalizationErrorPayload); !ok {
					t.Errorf("expected tt.expectedResult.Failure to be of type *roundevents.RoundFinalizationErrorPayload, got %T", tt.expectedResult.Failure)
				} else if failurePayload.Error != expectedFailurePayload.Error {
					t.Errorf("expected failure error %q, got %q", expectedFailurePayload.Error, failurePayload.Error)
				}
			}
		})
	}
}

func TestRoundService_NotifyScoreModule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)

	testRoundID := sharedtypes.RoundID(uuid.New())
	user1ID := sharedtypes.DiscordID("user1")
	user2ID := sharedtypes.DiscordID("user2")
	user3ID := sharedtypes.DiscordID("user3")
	user4ID := sharedtypes.DiscordID("user4")

	tag1 := sharedtypes.TagNumber(10)
	tag2 := sharedtypes.TagNumber(0) // For nil or zero tag
	tag4 := sharedtypes.TagNumber(40)

	score1 := sharedtypes.Score(50)
	score2 := sharedtypes.Score(0) // For nil score
	score3 := sharedtypes.Score(60)

	tests := []struct {
		name           string
		mockDBSetup    func(*rounddb.MockRoundDB)
		payload        roundevents.RoundFinalizedPayload
		expectedResult RoundOperationResult
		expectedError  error
	}{
		{
			name: "success notifying score module with various participants",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// Mock GetRound call that happens in NotifyScoreModule
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(&roundtypes.Round{ID: testRoundID}, nil)
			},
			payload: roundevents.RoundFinalizedPayload{
				RoundID: testRoundID,
				RoundData: roundtypes.Round{
					ID: testRoundID,
					Participants: []roundtypes.Participant{
						{UserID: user1ID, TagNumber: &tag1, Score: &score1},
						{UserID: user2ID, TagNumber: nil, Score: nil},       // Nil tag, nil score
						{UserID: user3ID, TagNumber: &tag2, Score: &score3}, // Zero tag
						{UserID: user4ID, TagNumber: &tag4, Score: nil},     // Valid tag, nil score
					},
				},
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.ProcessRoundScoresRequestPayload{
					RoundID: testRoundID,
					Scores: []roundevents.ParticipantScore{
						{UserID: user1ID, TagNumber: &tag1, Score: score1},
						{UserID: user2ID, TagNumber: &tag2, Score: score2}, // Expect Tag 0, Score 0 for nil inputs
						{UserID: user3ID, TagNumber: &tag2, Score: score3}, // Expect Tag 0 for explicit 0 input
						{UserID: user4ID, TagNumber: &tag4, Score: score2}, // Expect Score 0 for nil input
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "success notifying score module with no participants",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// Mock GetRound call that happens in NotifyScoreModule
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(&roundtypes.Round{ID: testRoundID}, nil)
			},
			payload: roundevents.RoundFinalizedPayload{
				RoundID: testRoundID,
				RoundData: roundtypes.Round{
					ID:           testRoundID,
					Participants: []roundtypes.Participant{}, // Empty slice
				},
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.ProcessRoundScoresRequestPayload{
					RoundID: testRoundID,
					Scores:  []roundevents.ParticipantScore{}, // Expect empty scores slice
				},
			},
			expectedError: nil,
		},
		{
			name: "success notifying score module with nil participants",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// Mock GetRound call that happens in NotifyScoreModule
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(&roundtypes.Round{ID: testRoundID}, nil)
			},
			payload: roundevents.RoundFinalizedPayload{
				RoundID: testRoundID,
				RoundData: roundtypes.Round{
					ID:           testRoundID,
					Participants: nil, // Nil slice
				},
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.ProcessRoundScoresRequestPayload{
					RoundID: testRoundID,
					Scores:  []roundevents.ParticipantScore{}, // Expect empty scores slice
				},
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks for each subtest
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockDB := rounddb.NewMockRoundDB(ctrl)

			tt.mockDBSetup(mockDB)

			s := &RoundService{
				RoundDB:        mockDB,
				logger:         logger,
				metrics:        mockMetrics,
				tracer:         tracer,
				roundValidator: mockRoundValidator,
				serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			result, err := s.NotifyScoreModule(ctx, tt.payload)

			// Validate error
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}

			// Validate result payload
			if tt.expectedResult.Success != nil {
				if result.Success == nil {
					t.Errorf("expected success result, got failure")
				} else if successPayload, ok := result.Success.(*roundevents.ProcessRoundScoresRequestPayload); !ok {
					t.Errorf("expected result.Success to be of type *roundevents.ProcessRoundScoresRequestPayload, got %T", result.Success)
				} else if expectedSuccessPayload, ok := tt.expectedResult.Success.(*roundevents.ProcessRoundScoresRequestPayload); !ok {
					t.Errorf("expected tt.expectedResult.Success to be of type *roundevents.ProcessRoundScoresRequestPayload, got %T", tt.expectedResult.Success)
				} else {
					// Validate RoundID
					if successPayload.RoundID != expectedSuccessPayload.RoundID {
						t.Errorf("expected success RoundID %s, got %s", expectedSuccessPayload.RoundID, successPayload.RoundID)
					}
					// Validate scores length
					if len(successPayload.Scores) != len(expectedSuccessPayload.Scores) {
						t.Errorf("expected %d scores, got %d", len(expectedSuccessPayload.Scores), len(successPayload.Scores))
					}
					// Validate individual scores
					for i, expectedScore := range expectedSuccessPayload.Scores {
						if i < len(successPayload.Scores) {
							actualScore := successPayload.Scores[i]
							if actualScore.UserID != expectedScore.UserID {
								t.Errorf("score %d: expected UserID %s, got %s", i, expectedScore.UserID, actualScore.UserID)
							}
							if *actualScore.TagNumber != *expectedScore.TagNumber {
								t.Errorf("score %d: expected TagNumber %d, got %d", i, *expectedScore.TagNumber, *actualScore.TagNumber)
							}
							if actualScore.Score != expectedScore.Score {
								t.Errorf("score %d: expected Score %d, got %d", i, expectedScore.Score, actualScore.Score)
							}
						}
					}
				}
			} else if tt.expectedResult.Failure != nil {
				if result.Failure == nil {
					t.Errorf("expected failure result, got success")
				}
			}
		})
	}
}
