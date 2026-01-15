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
		payload        roundevents.ScoreUpdateRequestPayloadV1
		expectedResult RoundOperationResult
		expectedError  error
	}{
		{
			name: "successful validation",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// No DB interactions expected for validation
			},
			payload: roundevents.ScoreUpdateRequestPayloadV1{
				GuildID: guildID,
				RoundID: testScoreRoundID,
				UserID:  testParticipant,
				Score:   &testScore,
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.ScoreUpdateValidatedPayloadV1{
					GuildID: guildID,
					ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayloadV1{
						GuildID: guildID,
						RoundID: testScoreRoundID,
						UserID:  testParticipant,
						Score:   &testScore,
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
			payload: roundevents.ScoreUpdateRequestPayloadV1{
				GuildID: guildID,
				RoundID: sharedtypes.RoundID(uuid.Nil),
				UserID:  testParticipant,
				Score:   &testScore,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundScoreUpdateErrorPayloadV1{
					GuildID: guildID,
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayloadV1{
						GuildID: guildID,
						RoundID: sharedtypes.RoundID(uuid.Nil),
						UserID:  testParticipant,
						Score:   &testScore,
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
			payload: roundevents.ScoreUpdateRequestPayloadV1{
				GuildID: guildID,
				RoundID: testScoreRoundID,
				UserID:  "",
				Score:   &testScore,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundScoreUpdateErrorPayloadV1{
					GuildID: guildID,
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayloadV1{
						GuildID: guildID,
						RoundID: testScoreRoundID,
						UserID:  "",
						Score:   &testScore,
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
			payload: roundevents.ScoreUpdateRequestPayloadV1{
				GuildID: guildID,
				RoundID: testScoreRoundID,
				UserID:  testParticipant,
				Score:   nil,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundScoreUpdateErrorPayloadV1{
					GuildID: guildID,
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayloadV1{
						GuildID: guildID,
						RoundID: testScoreRoundID,
						UserID:  testParticipant,
						Score:   nil,
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
			payload: roundevents.ScoreUpdateRequestPayloadV1{
				GuildID: guildID,
				RoundID: sharedtypes.RoundID(uuid.Nil),
				UserID:  "",
				Score:   nil,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundScoreUpdateErrorPayloadV1{
					GuildID: guildID,
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayloadV1{
						GuildID: guildID,
						RoundID: sharedtypes.RoundID(uuid.Nil),
						UserID:  "",
						Score:   nil,
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
				} else if successPayload, ok := result.Success.(*roundevents.ScoreUpdateValidatedPayloadV1); !ok {
					t.Errorf("expected result.Success to be of type *roundevents.ScoreUpdateValidatedPayloadV1, got %T", result.Success)
				} else if expectedSuccessPayload, ok := tt.expectedResult.Success.(*roundevents.ScoreUpdateValidatedPayloadV1); !ok {
					t.Errorf("expected tt.expectedResult.Success to be of type *roundevents.ScoreUpdateValidatedPayloadV1, got %T", tt.expectedResult.Success)
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
				} else if failurePayload, ok := result.Failure.(*roundevents.RoundScoreUpdateErrorPayloadV1); !ok {
					t.Errorf("expected result.Failure to be of type *roundevents.RoundScoreUpdateErrorPayloadV1, got %T", result.Failure)
				} else if expectedFailurePayload, ok := tt.expectedResult.Failure.(*roundevents.RoundScoreUpdateErrorPayloadV1); !ok {
					t.Errorf("expected tt.expectedResult.Failure to be of type *roundevents.RoundScoreUpdateErrorPayloadV1, got %T", tt.expectedResult.Failure)
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
		payload        roundevents.ScoreUpdateValidatedPayloadV1
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
			payload: roundevents.ScoreUpdateValidatedPayloadV1{
				GuildID: guildID,
				ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayloadV1{
					GuildID: guildID,
					RoundID: testScoreRoundID,
					UserID:  testParticipant,
					Score:   &testScore,
				},
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.ParticipantScoreUpdatedPayloadV1{
					GuildID:        guildID,
					RoundID:        testScoreRoundID,
					UserID:         testParticipant,
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
			payload: roundevents.ScoreUpdateValidatedPayloadV1{
				GuildID: guildID,
				ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayloadV1{
					GuildID: guildID,
					RoundID: testScoreRoundID,
					UserID:  testParticipant,
					Score:   &testScore,
				},
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundScoreUpdateErrorPayloadV1{
					GuildID: guildID,
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayloadV1{
						GuildID: guildID,
						RoundID: testScoreRoundID,
						UserID:  testParticipant,
						Score:   &testScore,
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
			payload: roundevents.ScoreUpdateValidatedPayloadV1{
				GuildID: guildID,
				ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayloadV1{
					GuildID:   guildID,
					RoundID:   testScoreRoundID,
					UserID:    testParticipant,
					Score:     &testScore,
					ChannelID: "test-channel",
					MessageID: "test-message",
				},
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
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
			payload: roundevents.ScoreUpdateValidatedPayloadV1{
				GuildID: guildID,
				ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayloadV1{
					GuildID:   guildID,
					RoundID:   testScoreRoundID,
					UserID:    testParticipant,
					Score:     &testScore,
					ChannelID: "test-channel",
					MessageID: "test-message",
				},
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
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
		payload        roundevents.ParticipantScoreUpdatedPayloadV1
		expectedResult RoundOperationResult
		expectedError  error
	}{
		{
			name: "all scores submitted",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				guildID := sharedtypes.GuildID("guild-123")
				// Round fetch is called once to build AllScoresSubmitted payload
				mockDB.EXPECT().GetRound(gomock.Any(), guildID, testScoreRoundID).Return(&roundtypes.Round{
					ID:      testScoreRoundID,
					GuildID: guildID,
					Participants: []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: &testScore},
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept, Score: &testScore},
					},
				}, nil)
			},
			payload: roundevents.ParticipantScoreUpdatedPayloadV1{
				GuildID:        sharedtypes.GuildID("guild-123"),
				RoundID:        testScoreRoundID,
				UserID:         testParticipant,
				Score:          testScore,
				ChannelID:      "test-channel",
				EventMessageID: testDiscordMessageID,
				Participants: []roundtypes.Participant{
					{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: &testScore},
					{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept, Score: &testScore},
				},
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.AllScoresSubmittedPayloadV1{
					GuildID:        sharedtypes.GuildID("guild-123"),
					RoundID:        testScoreRoundID,
					EventMessageID: testDiscordMessageID,
					RoundData: roundtypes.Round{
						ID:      testScoreRoundID,
						GuildID: sharedtypes.GuildID("guild-123"),
						Participants: []roundtypes.Participant{
							{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: &testScore},
							{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept, Score: &testScore},
						},
					},
					Participants: []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: &testScore},
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept, Score: &testScore},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "not all scores submitted",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// No DB calls expected when payload includes participants
			},
			payload: roundevents.ParticipantScoreUpdatedPayloadV1{
				GuildID:        sharedtypes.GuildID("guild-123"),
				RoundID:        testScoreRoundID,
				UserID:         testParticipant,
				Score:          testScore,
				ChannelID:      "test-channel",
				EventMessageID: testDiscordMessageID,
				Participants: []roundtypes.Participant{
					{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: &testScore},
					{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept, Score: nil},
				},
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.ScoresPartiallySubmittedPayloadV1{ // Changed to pointer
					GuildID:        sharedtypes.GuildID("guild-123"),
					RoundID:        testScoreRoundID,
					UserID:         testParticipant,
					Score:          testScore,
					EventMessageID: testDiscordMessageID,
					Participants: []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: &testScore},
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept, Score: nil},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "declined participant without score does not block finalization",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				guildID := sharedtypes.GuildID("guild-123")
				mockDB.EXPECT().GetRound(gomock.Any(), guildID, testScoreRoundID).Return(&roundtypes.Round{
					ID:      testScoreRoundID,
					GuildID: guildID,
					Participants: []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: &testScore},
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseDecline, Score: nil},
					},
				}, nil)
			},
			payload: roundevents.ParticipantScoreUpdatedPayloadV1{
				GuildID:        sharedtypes.GuildID("guild-123"),
				RoundID:        testScoreRoundID,
				UserID:         testParticipant,
				Score:          testScore,
				ChannelID:      "test-channel",
				EventMessageID: testDiscordMessageID,
				Participants: []roundtypes.Participant{
					{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: &testScore},
					{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseDecline, Score: nil},
				},
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.AllScoresSubmittedPayloadV1{
					GuildID:        sharedtypes.GuildID("guild-123"),
					RoundID:        testScoreRoundID,
					EventMessageID: testDiscordMessageID,
					RoundData: roundtypes.Round{
						ID:      testScoreRoundID,
						GuildID: sharedtypes.GuildID("guild-123"),
						Participants: []roundtypes.Participant{
							{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: &testScore},
							{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseDecline, Score: nil},
						},
					},
					Participants: []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: &testScore},
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseDecline, Score: nil},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "tentative participant without score blocks finalization",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {},
			payload: roundevents.ParticipantScoreUpdatedPayloadV1{
				GuildID:        sharedtypes.GuildID("guild-123"),
				RoundID:        testScoreRoundID,
				UserID:         testParticipant,
				Score:          testScore,
				ChannelID:      "test-channel",
				EventMessageID: testDiscordMessageID,
				Participants: []roundtypes.Participant{
					{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: &testScore},
					{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseTentative, Score: nil},
				},
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.ScoresPartiallySubmittedPayloadV1{
					GuildID:        sharedtypes.GuildID("guild-123"),
					RoundID:        testScoreRoundID,
					UserID:         testParticipant,
					Score:          testScore,
					EventMessageID: testDiscordMessageID,
					Participants: []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user1"), Response: roundtypes.ResponseAccept, Score: &testScore},
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseTentative, Score: nil},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "error checking if all scores submitted (GetParticipants fails in checkIfAllScoresSubmitted)",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// When all participants have scores the service will call GetRound; simulate GetRound failure
				guildID := sharedtypes.GuildID("guild-123")
				mockDB.EXPECT().GetRound(gomock.Any(), guildID, testScoreRoundID).Return(nil, errors.New("database error from checkIfAllScoresSubmitted"))
			},
			payload: roundevents.ParticipantScoreUpdatedPayloadV1{
				GuildID:        sharedtypes.GuildID("guild-123"),
				RoundID:        testScoreRoundID,
				UserID:         testParticipant,
				Score:          testScore,
				ChannelID:      "test-channel",
				EventMessageID: testDiscordMessageID,
				Participants:   []roundtypes.Participant{{UserID: sharedtypes.DiscordID("user1"), Score: &testScore}},
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
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
				// When all participants are present the service will call GetRound to fetch round data; simulate GetRound failure
				mockDB.EXPECT().GetRound(gomock.Any(), guildID, testScoreRoundID).Return(nil, errors.New("database error from main func GetRound"))
			},
			payload: roundevents.ParticipantScoreUpdatedPayloadV1{
				GuildID:        sharedtypes.GuildID("guild-123"),
				RoundID:        testScoreRoundID,
				UserID:         testParticipant,
				Score:          testScore,
				ChannelID:      "test-channel",
				EventMessageID: testDiscordMessageID,
				Participants:   []roundtypes.Participant{{UserID: sharedtypes.DiscordID("user1"), Score: &testScore}},
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					GuildID: sharedtypes.GuildID("guild-123"),
					RoundID: testScoreRoundID,
					Error:   "database error from main func GetRound",
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
