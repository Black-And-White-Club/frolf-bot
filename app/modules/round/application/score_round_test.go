package roundservice

import (
	"context"
	"errors"
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

var (
	testScoreRoundID     = sharedtypes.RoundID(uuid.New())
	testParticipant      = sharedtypes.DiscordID("user1")
	testScore            = sharedtypes.Score(10)
	testDiscordMessageID = "12345"
)

func TestRoundService_ValidateScoreUpdateRequest(t *testing.T) {
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

	tests := []struct {
		name           string
		mockDBSetup    func(*rounddb.MockRoundDB)
		payload        roundevents.ScoreUpdateRequestPayload
		expectedResult RoundOperationResult
		expectedError  error
	}{
		{
			name: "successful validation",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// No DB interactions expected for validation
			},
			payload: roundevents.ScoreUpdateRequestPayload{
				RoundID:     testScoreRoundID,
				Participant: testParticipant,
				Score:       &testScore,
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.ScoreUpdateValidatedPayload{
					ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
						RoundID:     testScoreRoundID,
						Participant: testParticipant,
						Score:       &testScore,
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "invalid round ID",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// No DB interactions expected for validation
			},
			payload: roundevents.ScoreUpdateRequestPayload{
				RoundID:     sharedtypes.RoundID(uuid.Nil),
				Participant: testParticipant,
				Score:       &testScore,
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundScoreUpdateErrorPayload{
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayload{
						RoundID:     sharedtypes.RoundID(uuid.Nil),
						Participant: testParticipant,
						Score:       &testScore,
					},
					Error: "validation errors: round ID cannot be zero",
				},
			},
			expectedError: errors.New("validation errors: round ID cannot be zero"),
		},
		{
			name: "empty participant",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// No DB interactions expected for validation
			},
			payload: roundevents.ScoreUpdateRequestPayload{
				RoundID:     testScoreRoundID,
				Participant: "",
				Score:       &testScore,
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundScoreUpdateErrorPayload{
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayload{
						RoundID:     testScoreRoundID,
						Participant: "",
						Score:       &testScore,
					},
					Error: "validation errors: participant Discord ID cannot be empty",
				},
			},
			expectedError: errors.New("validation errors: participant Discord ID cannot be empty"),
		},
		{
			name: "nil score",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// No DB interactions expected for validation
			},
			payload: roundevents.ScoreUpdateRequestPayload{
				RoundID:     testScoreRoundID,
				Participant: testParticipant,
				Score:       nil,
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundScoreUpdateErrorPayload{
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayload{
						RoundID:     testScoreRoundID,
						Participant: testParticipant,
						Score:       nil,
					},
					Error: "validation errors: score cannot be empty",
				},
			},
			expectedError: errors.New("validation errors: score cannot be empty"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockDBSetup(mockDB)

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

			_, err := s.ValidateScoreUpdateRequest(ctx, tt.payload)
			if (err != nil) && (tt.expectedError == nil || err.Error() != tt.expectedError.Error()) {
				t.Fatalf("expected error %v, got %v", tt.expectedError, err)
			}

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

func TestRoundService_UpdateParticipantScore(t *testing.T) {
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

	tests := []struct {
		name           string
		mockDBSetup    func(*rounddb.MockRoundDB)
		payload        roundevents.ScoreUpdateValidatedPayload
		expectedResult RoundOperationResult
		expectedError  error
	}{
		{
			name: "successful update",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().UpdateParticipantScore(ctx, testScoreRoundID, testParticipant, testScore).Return(nil)
				// Expect GetParticipants to be called after updating score
				mockDB.EXPECT().GetParticipants(ctx, testScoreRoundID).Return([]roundtypes.Participant{
					{UserID: testParticipant, Score: &testScore},
				}, nil)
				mockDB.EXPECT().GetRound(ctx, testScoreRoundID).Return(&roundtypes.Round{
					EventMessageID: testDiscordMessageID,
				}, nil)
			},
			payload: roundevents.ScoreUpdateValidatedPayload{
				ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
					RoundID:     testScoreRoundID,
					Participant: testParticipant,
					Score:       &testScore,
				},
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.ParticipantScoreUpdatedPayload{ // Note: this is a pointer in the actual code
					RoundID:        testScoreRoundID,
					Participant:    testParticipant,
					Score:          testScore,
					EventMessageID: testDiscordMessageID,
					Participants: []roundtypes.Participant{ // Expect participants in the success payload
						{UserID: testParticipant, Score: &testScore},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "error updating score",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().UpdateParticipantScore(ctx, testScoreRoundID, testParticipant, testScore).Return(errors.New("database error"))
			},
			payload: roundevents.ScoreUpdateValidatedPayload{
				ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
					RoundID:     testScoreRoundID,
					Participant: testParticipant,
					Score:       &testScore,
				},
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundScoreUpdateErrorPayload{
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayload{
						RoundID:     testScoreRoundID,
						Participant: testParticipant,
						Score:       &testScore,
					},
					Error: "Failed to update score in database: database error", // Updated expected error message
				},
			},
			expectedError: errors.New("failed to update participant score in DB: database error"), // Updated expected error message
		},
		{
			name: "error getting participants after update", // New test case for GetParticipants error
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().UpdateParticipantScore(ctx, testScoreRoundID, testParticipant, testScore).Return(nil)
				mockDB.EXPECT().GetParticipants(ctx, testScoreRoundID).Return(nil, errors.New("participants fetch error"))
			},
			payload: roundevents.ScoreUpdateValidatedPayload{
				ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
					RoundID:     testScoreRoundID,
					Participant: testParticipant,
					Score:       &testScore,
				},
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: testScoreRoundID,
					Error:   "Failed to retrieve updated participants list after score update: participants fetch error",
				},
			},
			expectedError: errors.New("failed to get updated participants list: participants fetch error"),
		},
		{
			name: "error getting round",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().UpdateParticipantScore(ctx, testScoreRoundID, testParticipant, testScore).Return(nil)
				mockDB.EXPECT().GetParticipants(ctx, testScoreRoundID).Return([]roundtypes.Participant{
					{UserID: testParticipant, Score: &testScore},
				}, nil) // Mock GetParticipants as it's called before GetRound
				mockDB.EXPECT().GetRound(ctx, testScoreRoundID).Return(&roundtypes.Round{}, errors.New("database error"))
			},
			payload: roundevents.ScoreUpdateValidatedPayload{
				ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
					RoundID:     testScoreRoundID,
					Participant: testParticipant,
					Score:       &testScore,
				},
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: testScoreRoundID,
					Error:   "Failed to retrieve round details for event payload: database error", // Updated expected error message
				},
			},
			expectedError: errors.New("failed to get round details for event payload: database error"), // Updated expected error message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockDBSetup(mockDB)

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

			result, err := s.UpdateParticipantScore(ctx, tt.payload)

			// Compare errors
			if (err != nil) != (tt.expectedError != nil) {
				t.Fatalf("expected error presence %v, got %v (error: %v)", tt.expectedError != nil, err != nil, err)
			}
			if err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
			}

			// Compare results (simplified for success/failure check, full comparison can be more complex)
			if tt.expectedResult.Success != nil && result.Success == nil {
				t.Errorf("expected success result, got failure")
			}
			if tt.expectedResult.Failure != nil && result.Failure == nil {
				t.Errorf("expected failure result, got success")
			}
			// Add more detailed comparison for success/failure payloads if needed
		})
	}
}

func TestRoundService_CheckAllScoresSubmitted(t *testing.T) {
	tests := []struct {
		name           string
		mockDBSetup    func(*rounddb.MockRoundDB)
		payload        roundevents.ParticipantScoreUpdatedPayload
		expectedResult RoundOperationResult
		expectedError  error
	}{
		{
			name: "all scores submitted",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetParticipants(gomock.Any(), testScoreRoundID).Return([]roundtypes.Participant{
					{
						UserID: sharedtypes.DiscordID("user1"),
						Score:  &testScore,
					},
					{
						UserID: sharedtypes.DiscordID("user2"),
						Score:  &testScore,
					},
				}, nil).Times(2) // Called twice: once by checkIfAllScoresSubmitted, once directly
			},
			payload: roundevents.ParticipantScoreUpdatedPayload{
				RoundID:        testScoreRoundID,
				Participant:    testParticipant,
				Score:          testScore,
				EventMessageID: testDiscordMessageID,
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.AllScoresSubmittedPayload{
					RoundID:        testScoreRoundID,
					EventMessageID: testDiscordMessageID,
					Participants: []roundtypes.Participant{ // Expected participants in the payload
						{UserID: sharedtypes.DiscordID("user1"), Score: &testScore},
						{UserID: sharedtypes.DiscordID("user2"), Score: &testScore},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "not all scores submitted",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetParticipants(gomock.Any(), testScoreRoundID).Return([]roundtypes.Participant{
					{
						UserID: sharedtypes.DiscordID("user1"),
						Score:  &testScore,
					},
					{
						UserID: sharedtypes.DiscordID("user2"),
						Score:  nil,
					},
				}, nil).Times(2) // Called twice
			},
			payload: roundevents.ParticipantScoreUpdatedPayload{
				RoundID:        testScoreRoundID,
				Participant:    testParticipant,
				Score:          testScore,
				EventMessageID: testDiscordMessageID,
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.NotAllScoresSubmittedPayload{
					RoundID:        testScoreRoundID,
					Participant:    testParticipant,
					Score:          testScore,
					EventMessageID: testDiscordMessageID, // Include EventMessageID
					Participants: []roundtypes.Participant{ // Expected participants in the payload
						{UserID: sharedtypes.DiscordID("user1"), Score: &testScore},
						{UserID: sharedtypes.DiscordID("user2"), Score: nil},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "error checking if all scores submitted (GetParticipants fails in checkIfAllScoresSubmitted)",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// This mock is for the first call to GetParticipants inside checkIfAllScoresSubmitted
				mockDB.EXPECT().GetParticipants(gomock.Any(), testScoreRoundID).Return(nil, errors.New("database error from checkIfAllScoresSubmitted")).Times(1)
				// No second GetParticipants call expected if the first one fails and returns early
			},
			payload: roundevents.ParticipantScoreUpdatedPayload{
				RoundID:        testScoreRoundID,
				Participant:    testParticipant,
				Score:          testScore,
				EventMessageID: testDiscordMessageID,
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: testScoreRoundID,
					Error:   "database error from checkIfAllScoresSubmitted", // This is the error from checkIfAllScoresSubmitted
				},
			},
			expectedError: errors.New("failed to check if all scores have been submitted: database error from checkIfAllScoresSubmitted"), // Wrapped error
		},
		{
			name: "error getting participants for success payload (GetParticipants fails after checkIfAllScoresSubmitted)",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// First call to GetParticipants (inside checkIfAllScoresSubmitted) succeeds
				mockDB.EXPECT().GetParticipants(gomock.Any(), testScoreRoundID).Return([]roundtypes.Participant{
					{UserID: sharedtypes.DiscordID("user1"), Score: &testScore},
				}, nil).Times(1)
				// Second call to GetParticipants (for the success payload) fails
				mockDB.EXPECT().GetParticipants(gomock.Any(), testScoreRoundID).Return(nil, errors.New("database error from main func GetParticipants")).Times(1)
			},
			payload: roundevents.ParticipantScoreUpdatedPayload{
				RoundID:        testScoreRoundID,
				Participant:    testParticipant,
				Score:          testScore,
				EventMessageID: testDiscordMessageID,
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: testScoreRoundID,
					Error:   "Failed to retrieve updated participants list for score check: database error from main func GetParticipants",
				},
			},
			expectedError: errors.New("failed to get updated participants list for score check: database error from main func GetParticipants"), // Wrapped error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := rounddb.NewMockRoundDB(ctrl)
			logger := loggerfrolfbot.NoOpLogger
			tracerProvider := noop.NewTracerProvider()
			tracer := tracerProvider.Tracer("test")
			mockMetrics := &roundmetrics.NoOpMetrics{}
			mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)
			mockEventBus := eventbus.NewMockEventBus(ctrl)

			tt.mockDBSetup(mockDB)

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

			ctx := context.Background()
			result, err := s.CheckAllScoresSubmitted(ctx, tt.payload)

			// Compare errors
			if (err != nil) != (tt.expectedError != nil) {
				t.Fatalf("expected error presence %v, got %v (error: %v)", tt.expectedError != nil, err != nil, err)
			}
			if err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
			}

			// Compare results (simplified for success/failure check, full comparison can be more complex)
			if tt.expectedResult.Success != nil && result.Success == nil {
				t.Errorf("expected success result, got failure")
			}
			if tt.expectedResult.Failure != nil && result.Failure == nil {
				t.Errorf("expected failure result, got success")
			}
		})
	}
}
