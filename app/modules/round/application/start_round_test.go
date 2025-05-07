package roundservice

import (
	"context"
	"errors"
	"reflect"
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
	testStartRoundID        = sharedtypes.RoundID(uuid.New())
	testRoundTitle          = roundtypes.Title("Test Round")
	testStartLocation       = roundtypes.Location("Test Location")
	testStartRoundTime      = sharedtypes.StartTime(time.Now())
	testStartEventMessageID = "12345"
)

var (
	testStartParticipant1 = roundtypes.Participant{
		UserID:    sharedtypes.DiscordID("user1"),
		TagNumber: nil,
		Response:  roundtypes.ResponseAccept,
	}
	testStartParticipant2 = roundtypes.Participant{
		UserID:    sharedtypes.DiscordID("user2"),
		TagNumber: nil,
		Response:  roundtypes.ResponseTentative,
	}
)

func TestRoundService_ProcessRoundStart(t *testing.T) {
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
		payload        roundevents.RoundStartedPayload
		expectedResult RoundOperationResult
		expectedError  error
	}{
		{
			name: "successful processing",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(ctx, testStartRoundID).Return(&roundtypes.Round{
					ID:             testStartRoundID,
					Title:          testRoundTitle,
					Location:       &testStartLocation,
					StartTime:      &testStartRoundTime,
					State:          roundtypes.RoundStateUpcoming,
					Participants:   []roundtypes.Participant{testStartParticipant1, testStartParticipant2},
					EventMessageID: testStartEventMessageID,
				}, nil)
				mockDB.EXPECT().UpdateRound(ctx, testStartRoundID, &roundtypes.Round{
					ID:             testStartRoundID,
					Title:          testRoundTitle,
					Location:       &testStartLocation,
					StartTime:      &testStartRoundTime,
					State:          roundtypes.RoundStateInProgress,
					Participants:   []roundtypes.Participant{testStartParticipant1, testStartParticipant2},
					EventMessageID: testStartEventMessageID,
				}).Return(nil)
			},
			payload: roundevents.RoundStartedPayload{
				RoundID: testStartRoundID,
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.DiscordRoundStartPayload{
					RoundID:   testStartRoundID,
					Title:     testRoundTitle,
					Location:  &testStartLocation,
					StartTime: &testStartRoundTime,
					Participants: []roundevents.RoundParticipant{
						{
							UserID:    sharedtypes.DiscordID("user1"),
							TagNumber: nil,
							Response:  roundtypes.ResponseAccept,
							Score:     nil,
						},
						{
							UserID:    sharedtypes.DiscordID("user2"),
							TagNumber: nil,
							Response:  roundtypes.ResponseTentative,
							Score:     nil,
						},
					},
					EventMessageID: testStartEventMessageID,
				},
			},
			expectedError: nil,
		},
		{
			name: "error getting round",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(ctx, testStartRoundID).Return(&roundtypes.Round{}, errors.New("database error"))
			},
			payload: roundevents.RoundStartedPayload{
				RoundID: testStartRoundID,
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: testStartRoundID,
					Error:   "database error",
				},
			},
			expectedError: errors.New("failed to get round from database: database error"),
		},
		{
			name: "error updating round",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(ctx, testStartRoundID).Return(&roundtypes.Round{
					ID:             testStartRoundID,
					Title:          testRoundTitle,
					Location:       &testStartLocation,
					StartTime:      &testStartRoundTime,
					State:          roundtypes.RoundStateUpcoming,
					Participants:   []roundtypes.Participant{testStartParticipant1, testStartParticipant2},
					EventMessageID: testStartEventMessageID,
				}, nil)
				mockDB.EXPECT().UpdateRound(ctx, testStartRoundID, &roundtypes.Round{
					ID:             testStartRoundID,
					Title:          testRoundTitle,
					Location:       &testStartLocation,
					StartTime:      &testStartRoundTime,
					State:          roundtypes.RoundStateInProgress,
					Participants:   []roundtypes.Participant{testStartParticipant1, testStartParticipant2},
					EventMessageID: testStartEventMessageID,
				}).Return(errors.New("database error"))
			},
			payload: roundevents.RoundStartedPayload{
				RoundID: testStartRoundID,
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: testStartRoundID,
					Error:   "database error",
				},
			},
			expectedError: errors.New("failed to update round: database error"),
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

			result, err := s.ProcessRoundStart(ctx, tt.payload)
			if (err != nil) != (tt.expectedError != nil) {
				t.Fatalf("expected error %v, got %v", tt.expectedError, err)
			}

			if err != nil && tt.expectedError != nil {
				if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message %q, got %q", tt.expectedError.Error(), err.Error())
				}
			}

			if !reflect.DeepEqual(result.Success, tt.expectedResult.Success) {
				t.Errorf("expected result Success %v, got %v", tt.expectedResult.Success, result.Success)
			}

			if !reflect.DeepEqual(result.Failure, tt.expectedResult.Failure) {
				t.Errorf("expected result Failure %v, got %v", tt.expectedResult.Failure, result.Failure)
			}
		})
	}
}
