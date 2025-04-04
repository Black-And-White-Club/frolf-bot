package roundservice

import (
	"context"
	"errors"
	"testing"

	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/round"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/mocks"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

var (
	testScoreRoundID = sharedtypes.RoundID(uuid.New())
	testParticipant  = sharedtypes.DiscordID("user1")
	testScore        = sharedtypes.Score(10)
)

func TestRoundService_ValidateScoreUpdateRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockDB := rounddb.NewMockRoundDB(ctrl)
	mockLogger := &lokifrolfbot.NoOpLogger{}
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockTracer := tempofrolfbot.NewNoOpTracer()
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
				logger:         mockLogger,
				metrics:        mockMetrics,
				tracer:         mockTracer,
				roundValidator: mockRoundValidator,
				EventBus:       mockEventBus,
				serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func() (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc()
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
	mockLogger := &lokifrolfbot.NoOpLogger{}
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockTracer := tempofrolfbot.NewNoOpTracer()
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
				mockDB.EXPECT().GetRound(ctx, testScoreRoundID).Return(&roundtypes.Round{
					EventMessageID: testScoreRoundID,
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
				Success: roundevents.ParticipantScoreUpdatedPayload{
					RoundID:        testScoreRoundID,
					Participant:    testParticipant,
					Score:          testScore,
					EventMessageID: &testScoreRoundID,
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
					Error: "database error",
				},
			},
			expectedError: errors.New("database error"),
		},
		{
			name: "error getting round",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().UpdateParticipantScore(ctx, testScoreRoundID, testParticipant, testScore).Return(nil)
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
					Error:   "database error",
				},
			},
			expectedError: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockDBSetup(mockDB)

			s := &RoundService{
				RoundDB:        mockDB,
				logger:         mockLogger,
				metrics:        mockMetrics,
				tracer:         mockTracer,
				roundValidator: mockRoundValidator,
				EventBus:       mockEventBus,
				serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func() (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc()
				},
			}

			_, err := s.UpdateParticipantScore(ctx, tt.payload)
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
				}, nil).Times(1)
			},
			payload: roundevents.ParticipantScoreUpdatedPayload{
				RoundID:        testScoreRoundID,
				Participant:    testParticipant,
				Score:          testScore,
				EventMessageID: &testScoreRoundID,
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.AllScoresSubmittedPayload{
					RoundID:        testScoreRoundID,
					EventMessageID: &testScoreRoundID,
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
				}, nil).Times(1)
			},
			payload: roundevents.ParticipantScoreUpdatedPayload{
				RoundID:        testScoreRoundID,
				Participant:    testParticipant,
				Score:          testScore,
				EventMessageID: &testScoreRoundID,
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.NotAllScoresSubmittedPayload{
					RoundID:        testScoreRoundID,
					Participant:    testParticipant,
					Score:          testScore,
					EventMessageID: testScoreRoundID,
				},
			},
			expectedError: nil,
		},
		{
			name: "error getting round",
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
				}, errors.New("database error")).Times(1)
			},
			payload: roundevents.ParticipantScoreUpdatedPayload{
				RoundID:        testScoreRoundID,
				Participant:    testParticipant,
				Score:          testScore,
				EventMessageID: &testScoreRoundID,
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: testScoreRoundID,
					Error:   "database error",
				},
			},
			expectedError: errors.New("database error"),
		},
		{
			name: "error getting participants",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetParticipants(gomock.Any(), testScoreRoundID).Return([]roundtypes.Participant{}, errors.New("database error")).Times(1)
			},
			payload: roundevents.ParticipantScoreUpdatedPayload{
				RoundID:        testScoreRoundID,
				Participant:    testParticipant,
				Score:          testScore,
				EventMessageID: &testScoreRoundID,
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: testScoreRoundID,
					Error:   "database error",
				},
			},
			expectedError: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := rounddb.NewMockRoundDB(ctrl)
			mockLogger := &lokifrolfbot.NoOpLogger{}
			mockMetrics := &roundmetrics.NoOpMetrics{}
			mockTracer := tempofrolfbot.NewNoOpTracer()
			mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)
			mockEventBus := eventbus.NewMockEventBus(ctrl)

			tt.mockDBSetup(mockDB)

			s := &RoundService{
				RoundDB:        mockDB,
				logger:         mockLogger,
				metrics:        mockMetrics,
				tracer:         mockTracer,
				roundValidator: mockRoundValidator,
				EventBus:       mockEventBus,
				serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func() (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc()
				},
			}

			ctx := context.Background()
			_, err := s.CheckAllScoresSubmitted(ctx, tt.payload)
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
