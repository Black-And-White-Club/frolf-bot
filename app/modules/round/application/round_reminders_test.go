package roundservice

import (
	"context"
	"errors"
	"testing"
	"time"

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
	testReminderRoundID    = sharedtypes.RoundID(uuid.New())
	testReminderRoundTitle = roundtypes.Title("Test Round")
	testReminderDesc       = roundtypes.Description("Test Description")
	testReminderLocation   = roundtypes.Location("Test Location")
	testReminderStartTime  = sharedtypes.StartTime(time.Now())
	testReminderType       = "Test Reminder Type"
	testEventMessageID     = testReminderRoundID
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
	mockLogger := &lokifrolfbot.NoOpLogger{}
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockTracer := tempofrolfbot.NewNoOpTracer()
	mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)
	mockEventBus := eventbus.NewMockEventBus(ctrl)

	tests := []struct {
		name           string
		mockDBSetup    func(*rounddb.MockRoundDB)
		payload        roundevents.DiscordReminderPayload
		expectedResult RoundOperationResult
		expectedError  error
	}{
		{
			name: "successful processing with participants",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetParticipants(ctx, testReminderRoundID).Return([]roundtypes.Participant{testParticipant1, testParticipant2}, nil)
			},
			payload: roundevents.DiscordReminderPayload{
				RoundID:        testReminderRoundID,
				RoundTitle:     testReminderRoundTitle,
				StartTime:      &testReminderStartTime,
				Location:       &testReminderLocation,
				ReminderType:   testReminderType,
				EventMessageID: testEventMessageID,
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.DiscordReminderPayload{
					RoundID:        testReminderRoundID,
					RoundTitle:     testReminderRoundTitle,
					StartTime:      &testReminderStartTime,
					Location:       &testReminderLocation,
					UserIDs:        []sharedtypes.DiscordID{testParticipant1.UserID, testParticipant2.UserID},
					ReminderType:   testReminderType,
					EventMessageID: testEventMessageID,
				},
			},
			expectedError: nil,
		},
		{
			name: "successful processing with no participants",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetParticipants(ctx, testReminderRoundID).Return([]roundtypes.Participant{testParticipant3}, nil)
			},
			payload: roundevents.DiscordReminderPayload{
				RoundID:        testReminderRoundID,
				RoundTitle:     testReminderRoundTitle,
				StartTime:      &testReminderStartTime,
				Location:       &testReminderLocation,
				ReminderType:   testReminderType,
				EventMessageID: testEventMessageID,
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
				mockDB.EXPECT().GetParticipants(ctx, testReminderRoundID).Return([]roundtypes.Participant{}, errors.New("database error"))
			},
			payload: roundevents.DiscordReminderPayload{
				RoundID:        testReminderRoundID,
				RoundTitle:     testReminderRoundTitle,
				StartTime:      &testReminderStartTime,
				Location:       &testReminderLocation,
				ReminderType:   testReminderType,
				EventMessageID: testEventMessageID,
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: testReminderRoundID,
					Error:   "database error",
				},
			},
			expectedError: errors.New("failed to get round: database error"),
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

			_, err := s.ProcessRoundReminder(ctx, tt.payload)
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
