package roundservice

import (
	"context"
	"errors"
	"testing"
	"time"

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
	testReminderRoundID    = sharedtypes.RoundID(uuid.New())
	testReminderRoundTitle = roundtypes.Title("Test Round")
	testReminderLocation   = roundtypes.Location("Test Location")
	testReminderStartTime  = sharedtypes.StartTime(time.Now())
	testReminderType       = "Test Reminder Type"
)

var (
	testParticipant1 = roundtypes.Participant{
		UserID:    sharedtypes.DiscordID("user1"),
		TagNumber: nil,
		Response:  roundtypes.ResponseAccept,
	}
	testParticipant2 = roundtypes.Participant{
		UserID:    sharedtypes.DiscordID("user2"),
		TagNumber: nil,
		Response:  roundtypes.ResponseTentative,
	}
	testParticipant3 = roundtypes.Participant{
		UserID:    sharedtypes.DiscordID("user3"),
		TagNumber: nil,
		Response:  roundtypes.ResponseDecline,
	}
)

func TestRoundService_ProcessRoundReminder(t *testing.T) {
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
	testDiscordMessageID := "12345"

	tests := []struct {
		name           string
		mockDBSetup    func(*rounddb.MockRoundDB)
		payload        roundevents.DiscordReminderPayloadV1
		expectedResult RoundOperationResult
		expectedError  error
	}{
		{
			name: "successful processing with participants",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				guildID := sharedtypes.GuildID("guild-123")
				mockDB.EXPECT().GetParticipants(ctx, guildID, testReminderRoundID).Return([]roundtypes.Participant{testParticipant1, testParticipant2}, nil)
			},
			payload: roundevents.DiscordReminderPayloadV1{
				RoundID:        testReminderRoundID,
				GuildID:        sharedtypes.GuildID("guild-123"),
				RoundTitle:     testReminderRoundTitle,
				StartTime:      &testReminderStartTime,
				Location:       &testReminderLocation,
				ReminderType:   testReminderType,
				EventMessageID: testDiscordMessageID,
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.DiscordReminderPayloadV1{
					RoundID:        testReminderRoundID,
					RoundTitle:     testReminderRoundTitle,
					StartTime:      &testReminderStartTime,
					Location:       &testReminderLocation,
					UserIDs:        []sharedtypes.DiscordID{testParticipant1.UserID, testParticipant2.UserID},
					ReminderType:   testReminderType,
					EventMessageID: testDiscordMessageID,
				},
			},
			expectedError: nil,
		},
		{
			name: "successful processing with no participants",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				guildID := sharedtypes.GuildID("guild-123")
				mockDB.EXPECT().GetParticipants(ctx, guildID, testReminderRoundID).Return([]roundtypes.Participant{testParticipant3}, nil)
			},
			payload: roundevents.DiscordReminderPayloadV1{
				RoundID:        testReminderRoundID,
				GuildID:        sharedtypes.GuildID("guild-123"),
				RoundTitle:     testReminderRoundTitle,
				StartTime:      &testReminderStartTime,
				Location:       &testReminderLocation,
				ReminderType:   testReminderType,
				EventMessageID: testDiscordMessageID,
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.RoundReminderProcessedPayload{
					RoundID: testReminderRoundID,
				},
			},
			expectedError: nil,
		},
		{
			name: "error retrieving participants",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				guildID := sharedtypes.GuildID("guild-123")
				mockDB.EXPECT().GetParticipants(ctx, guildID, testReminderRoundID).Return([]roundtypes.Participant{}, errors.New("database error"))
			},
			payload: roundevents.DiscordReminderPayloadV1{
				RoundID:        testReminderRoundID,
				GuildID:        sharedtypes.GuildID("guild-123"),
				RoundTitle:     testReminderRoundTitle,
				StartTime:      &testReminderStartTime,
				Location:       &testReminderLocation,
				ReminderType:   testReminderType,
				EventMessageID: testDiscordMessageID,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{ // Add pointer here
					RoundID: testReminderRoundID,
					Error:   "database error",
				},
			},
			expectedError: nil, // Change this to nil since you're handling errors in Failure
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

			result, err := s.ProcessRoundReminder(ctx, tt.payload)

			// Check error expectations
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			// Check result expectations
			if tt.expectedResult.Success != nil {
				if result.Success == nil {
					t.Errorf("expected success result, got nil")
				}
				// Add detailed comparison of the DiscordReminderPayload if needed
			}

			if tt.expectedResult.Failure != nil {
				if result.Failure == nil {
					t.Errorf("expected failure result, got nil")
				} else {
					expectedFailure := tt.expectedResult.Failure.(*roundevents.RoundErrorPayloadV1)
					actualFailure := result.Failure.(*roundevents.RoundErrorPayloadV1)
					if expectedFailure.RoundID != actualFailure.RoundID {
						t.Errorf("expected RoundID: %v, got: %v", expectedFailure.RoundID, actualFailure.RoundID)
					}
					if expectedFailure.Error != actualFailure.Error {
						t.Errorf("expected Error: %v, got: %v", expectedFailure.Error, actualFailure.Error)
					}
				}
			}
		})
	}
}
