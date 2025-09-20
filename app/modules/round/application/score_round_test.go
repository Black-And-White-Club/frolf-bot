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
	guildID := sharedtypes.GuildID("guild-123")
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
				GuildID:     guildID,
				RoundID:     testScoreRoundID,
				Participant: testParticipant,
				Score:       &testScore,
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.ScoreUpdateValidatedPayload{
					GuildID: guildID,
					ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
						GuildID:     guildID,
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
				GuildID:     guildID,
				RoundID:     sharedtypes.RoundID(uuid.Nil),
				Participant: testParticipant,
				Score:       &testScore,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundScoreUpdateErrorPayload{
					GuildID: guildID,
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayload{
						GuildID:     guildID,
						RoundID:     sharedtypes.RoundID(uuid.Nil),
						Participant: testParticipant,
						Score:       &testScore,
					},
					Error: "validation errors: round ID cannot be zero",
				},
			},
			expectedError: nil, // Changed from error to nil
		},
		{
			name: "empty participant",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// No DB interactions expected for validation
			},
			payload: roundevents.ScoreUpdateRequestPayload{
				GuildID:     guildID,
				RoundID:     testScoreRoundID,
				Participant: "",
				Score:       &testScore,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundScoreUpdateErrorPayload{
					GuildID: guildID,
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayload{
						GuildID:     guildID,
						RoundID:     testScoreRoundID,
						Participant: "",
						Score:       &testScore,
					},
					Error: "validation errors: participant Discord ID cannot be empty",
				},
			},
			expectedError: nil, // Changed from error to nil
		},
		{
			name: "nil score",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// No DB interactions expected for validation
			},
			payload: roundevents.ScoreUpdateRequestPayload{
				GuildID:     guildID,
				RoundID:     testScoreRoundID,
				Participant: testParticipant,
				Score:       nil,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundScoreUpdateErrorPayload{
					GuildID: guildID,
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayload{
						GuildID:     guildID,
						RoundID:     testScoreRoundID,
						Participant: testParticipant,
						Score:       nil,
					},
					Error: "validation errors: score cannot be empty",
				},
			},
			expectedError: nil, // Changed from error to nil
		},
		{
			name: "multiple validation errors",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// No DB interactions expected for validation
			},
			payload: roundevents.ScoreUpdateRequestPayload{
				GuildID:     guildID,
				RoundID:     sharedtypes.RoundID(uuid.Nil),
				Participant: "",
				Score:       nil,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundScoreUpdateErrorPayload{
					GuildID: guildID,
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayload{
						GuildID:     guildID,
						RoundID:     sharedtypes.RoundID(uuid.Nil),
						Participant: "",
						Score:       nil,
					},
					Error: "validation errors: round ID cannot be zero; participant Discord ID cannot be empty; score cannot be empty",
				},
			},
			expectedError: nil,
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

			result, err := s.ValidateScoreUpdateRequest(ctx, tt.payload)

			// Check error expectation
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			// Check result structure
			if tt.expectedResult.Success != nil {
				if result.Success == nil {
					t.Errorf("expected success result, got failure")
				} else if successPayload, ok := result.Success.(*roundevents.ScoreUpdateValidatedPayload); !ok {
					t.Errorf("expected result.Success to be of type *roundevents.ScoreUpdateValidatedPayload, got %T", result.Success)
				} else if expectedSuccessPayload, ok := tt.expectedResult.Success.(*roundevents.ScoreUpdateValidatedPayload); !ok {
					t.Errorf("expected tt.expectedResult.Success to be of type *roundevents.ScoreUpdateValidatedPayload, got %T", tt.expectedResult.Success)
				} else {
					// Compare the payload contents
					if successPayload.ScoreUpdateRequestPayload.RoundID != expectedSuccessPayload.ScoreUpdateRequestPayload.RoundID {
						t.Errorf("expected RoundID %v, got %v", expectedSuccessPayload.ScoreUpdateRequestPayload.RoundID, successPayload.ScoreUpdateRequestPayload.RoundID)
					}
				}
			}

			if tt.expectedResult.Failure != nil {
				if result.Failure == nil {
					t.Errorf("expected failure result, got success")
				} else if failurePayload, ok := result.Failure.(*roundevents.RoundScoreUpdateErrorPayload); !ok {
					t.Errorf("expected result.Failure to be of type *roundevents.RoundScoreUpdateErrorPayload, got %T", result.Failure)
				} else if expectedFailurePayload, ok := tt.expectedResult.Failure.(*roundevents.RoundScoreUpdateErrorPayload); !ok {
					t.Errorf("expected tt.expectedResult.Failure to be of type *roundevents.RoundScoreUpdateErrorPayload, got %T", tt.expectedResult.Failure)
				} else {
					// Compare the error message
					if failurePayload.Error != expectedFailurePayload.Error {
						t.Errorf("expected error message %q, got %q", expectedFailurePayload.Error, failurePayload.Error)
					}
				}
			}
		})
	}
}

func TestRoundService_UpdateParticipantScore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-123")
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
				mockDB.EXPECT().UpdateParticipantScore(ctx, guildID, testScoreRoundID, testParticipant, testScore).Return(nil)
				mockDB.EXPECT().GetParticipants(ctx, guildID, testScoreRoundID).Return([]roundtypes.Participant{
					{UserID: testParticipant, Score: &testScore},
				}, nil)
				mockDB.EXPECT().GetRound(ctx, guildID, testScoreRoundID).Return(&roundtypes.Round{
					EventMessageID: testDiscordMessageID,
				}, nil)
			},
			payload: roundevents.ScoreUpdateValidatedPayload{
				GuildID: guildID,
				ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
					GuildID:     guildID,
					RoundID:     testScoreRoundID,
					Participant: testParticipant,
					Score:       &testScore,
				},
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.ParticipantScoreUpdatedPayload{
					GuildID:        guildID,
					RoundID:        testScoreRoundID,
					Participant:    testParticipant,
					Score:          testScore,
					EventMessageID: testDiscordMessageID,
					Participants: []roundtypes.Participant{
						{UserID: testParticipant, Score: &testScore},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "error updating score",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().UpdateParticipantScore(ctx, guildID, testScoreRoundID, testParticipant, testScore).Return(errors.New("database error"))
			},
			payload: roundevents.ScoreUpdateValidatedPayload{
				GuildID: guildID,
				ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
					GuildID:     guildID,
					RoundID:     testScoreRoundID,
					Participant: testParticipant,
					Score:       &testScore,
				},
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundScoreUpdateErrorPayload{
					GuildID: guildID,
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayload{
						GuildID:     guildID,
						RoundID:     testScoreRoundID,
						Participant: testParticipant,
						Score:       &testScore,
					},
					Error: "Failed to update score in database: database error",
				},
			},
			expectedError: nil, // Changed from error to nil
		},
		{
			name: "error getting participants after update",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().UpdateParticipantScore(ctx, guildID, testScoreRoundID, testParticipant, testScore).Return(nil)
				mockDB.EXPECT().GetParticipants(ctx, guildID, testScoreRoundID).Return(nil, errors.New("participants fetch error"))
			},
			payload: roundevents.ScoreUpdateValidatedPayload{
				GuildID: guildID,
				ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
					GuildID:     guildID,
					RoundID:     testScoreRoundID,
					Participant: testParticipant,
					Score:       &testScore,
				},
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundErrorPayload{
					GuildID: guildID,
					RoundID: testScoreRoundID,
					Error:   "Failed to retrieve updated participants list after score update: participants fetch error",
				},
			},
			expectedError: nil, // Changed from error to nil
		},
		{
			name: "error getting round",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().UpdateParticipantScore(ctx, guildID, testScoreRoundID, testParticipant, testScore).Return(nil)
				mockDB.EXPECT().GetParticipants(ctx, guildID, testScoreRoundID).Return([]roundtypes.Participant{
					{UserID: testParticipant, Score: &testScore},
				}, nil)
				mockDB.EXPECT().GetRound(ctx, guildID, testScoreRoundID).Return(&roundtypes.Round{}, errors.New("database error"))
			},
			payload: roundevents.ScoreUpdateValidatedPayload{
				GuildID: guildID,
				ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
					GuildID:     guildID,
					RoundID:     testScoreRoundID,
					Participant: testParticipant,
					Score:       &testScore,
				},
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundErrorPayload{
					GuildID: guildID,
					RoundID: testScoreRoundID,
					Error:   "Failed to retrieve round details for event payload: database error",
				},
			},
			expectedError: nil, // Changed from error to nil
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

			// Check error expectation
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			// Check result structure
			if tt.expectedResult.Success != nil {
				if result.Success == nil {
					t.Errorf("expected success result, got failure")
				}
			}

			if tt.expectedResult.Failure != nil {
				if result.Failure == nil {
					t.Errorf("expected failure result, got success")
				}
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
				guildID := sharedtypes.GuildID("guild-123")
				mockDB.EXPECT().GetParticipants(gomock.Any(), guildID, testScoreRoundID).Return([]roundtypes.Participant{
					{
						UserID: sharedtypes.DiscordID("user1"),
						Score:  &testScore,
					},
					{
						UserID: sharedtypes.DiscordID("user2"),
						Score:  &testScore,
					},
				}, nil).Times(2)
			},
			payload: roundevents.ParticipantScoreUpdatedPayload{
				GuildID:        sharedtypes.GuildID("guild-123"),
				RoundID:        testScoreRoundID,
				Participant:    testParticipant,
				Score:          testScore,
				EventMessageID: testDiscordMessageID,
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.AllScoresSubmittedPayload{ // Changed to pointer
					GuildID:        sharedtypes.GuildID("guild-123"),
					RoundID:        testScoreRoundID,
					EventMessageID: testDiscordMessageID,
					Participants: []roundtypes.Participant{
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
				guildID := sharedtypes.GuildID("guild-123")
				mockDB.EXPECT().GetParticipants(gomock.Any(), guildID, testScoreRoundID).Return([]roundtypes.Participant{
					{
						UserID: sharedtypes.DiscordID("user1"),
						Score:  &testScore,
					},
					{
						UserID: sharedtypes.DiscordID("user2"),
						Score:  nil,
					},
				}, nil).Times(2)
			},
			payload: roundevents.ParticipantScoreUpdatedPayload{
				GuildID:        sharedtypes.GuildID("guild-123"),
				RoundID:        testScoreRoundID,
				Participant:    testParticipant,
				Score:          testScore,
				EventMessageID: testDiscordMessageID,
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.NotAllScoresSubmittedPayload{ // Changed to pointer
					GuildID:        sharedtypes.GuildID("guild-123"),
					RoundID:        testScoreRoundID,
					Participant:    testParticipant,
					Score:          testScore,
					EventMessageID: testDiscordMessageID,
					Participants: []roundtypes.Participant{
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
				guildID := sharedtypes.GuildID("guild-123")
				mockDB.EXPECT().GetParticipants(gomock.Any(), guildID, testScoreRoundID).Return(nil, errors.New("database error from checkIfAllScoresSubmitted")).Times(1)
				// No second GetParticipants call expected if the first one fails and returns early
			},
			payload: roundevents.ParticipantScoreUpdatedPayload{
				GuildID:        sharedtypes.GuildID("guild-123"),
				RoundID:        testScoreRoundID,
				Participant:    testParticipant,
				Score:          testScore,
				EventMessageID: testDiscordMessageID,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundErrorPayload{ // Changed to pointer
					GuildID: sharedtypes.GuildID("guild-123"),
					RoundID: testScoreRoundID,
					Error:   "database error from checkIfAllScoresSubmitted",
				},
			},
			expectedError: nil, // Changed from error to nil
		},
		{
			name: "error getting participants for success payload (GetParticipants fails after checkIfAllScoresSubmitted)",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				guildID := sharedtypes.GuildID("guild-123")
				// First call to GetParticipants (inside checkIfAllScoresSubmitted) succeeds
				mockDB.EXPECT().GetParticipants(gomock.Any(), guildID, testScoreRoundID).Return([]roundtypes.Participant{
					{UserID: sharedtypes.DiscordID("user1"), Score: &testScore},
				}, nil).Times(1)
				// Second call to GetParticipants (for the success payload) fails
				mockDB.EXPECT().GetParticipants(gomock.Any(), guildID, testScoreRoundID).Return(nil, errors.New("database error from main func GetParticipants")).Times(1)
			},
			payload: roundevents.ParticipantScoreUpdatedPayload{
				GuildID:        sharedtypes.GuildID("guild-123"),
				RoundID:        testScoreRoundID,
				Participant:    testParticipant,
				Score:          testScore,
				EventMessageID: testDiscordMessageID,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundErrorPayload{ // Changed to pointer
					GuildID: sharedtypes.GuildID("guild-123"),
					RoundID: testScoreRoundID,
					Error:   "Failed to retrieve updated participants list for score check: database error from main func GetParticipants",
				},
			},
			expectedError: nil, // Changed from error to nil
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

			// Check error expectation
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			// Check result structure
			if tt.expectedResult.Success != nil && result.Success == nil {
				t.Errorf("expected success result, got failure")
			}
			if tt.expectedResult.Failure != nil && result.Failure == nil {
				t.Errorf("expected failure result, got success")
			}
		})
	}
}
