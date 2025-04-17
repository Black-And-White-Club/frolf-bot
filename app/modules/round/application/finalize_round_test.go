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
	mockDB := rounddb.NewMockRoundDB(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)
	mockEventBus := eventbus.NewMockEventBus(ctrl)

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
				Success: roundevents.RoundFinalizedPayload{
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
				Failure: roundevents.RoundFinalizationErrorPayload{
					RoundID: testRoundID,
					Error:   fmt.Sprintf("failed to update round state to finalized: %v", dbUpdateError),
				},
			},
			expectedError: fmt.Errorf("failed to update round state: %w", dbUpdateError),
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
				Failure: roundevents.RoundFinalizationErrorPayload{
					RoundID: testRoundID,
					Error:   fmt.Sprintf("failed to fetch round data: %v", dbGetError),
				},
			},
			expectedError: fmt.Errorf("failed to fetch round %s: %w", testRoundID, dbGetError),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			_, err := s.FinalizeRound(ctx, tt.payload)

			// Validate error
			if tt.expectedError != nil {
				if err == nil || err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}

			// Validate result
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error, got: %v", err)
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
		payload        roundevents.RoundFinalizedPayload
		expectedResult RoundOperationResult
		expectedError  error
	}{
		{
			name: "success notifying score module with various participants",
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
				Success: roundevents.ProcessRoundScoresRequestPayload{
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
			payload: roundevents.RoundFinalizedPayload{
				RoundID: testRoundID,
				RoundData: roundtypes.Round{
					ID:           testRoundID,
					Participants: []roundtypes.Participant{}, // Empty slice
				},
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.ProcessRoundScoresRequestPayload{
					RoundID: testRoundID,
					Scores:  []roundevents.ParticipantScore{}, // Expect empty scores slice
				},
			},
			expectedError: nil,
		},
		{
			name: "success notifying score module with nil participants",
			payload: roundevents.RoundFinalizedPayload{
				RoundID: testRoundID,
				RoundData: roundtypes.Round{
					ID:           testRoundID,
					Participants: nil, // Nil slice
				},
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.ProcessRoundScoresRequestPayload{
					RoundID: testRoundID,
					Scores:  []roundevents.ParticipantScore{}, // Expect empty scores slice
				},
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &RoundService{
				logger:         logger,
				metrics:        mockMetrics,
				tracer:         tracer,
				roundValidator: mockRoundValidator,
				serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			_, err := s.NotifyScoreModule(ctx, tt.payload)

			// Validate error
			if tt.expectedError != nil {
				if err == nil || err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}

			// Validate result
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}
