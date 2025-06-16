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
				round := &roundtypes.Round{
					ID:             testStartRoundID,
					Title:          testRoundTitle,
					Location:       &testStartLocation,
					StartTime:      &testStartRoundTime,
					State:          roundtypes.RoundStateUpcoming,
					Participants:   []roundtypes.Participant{testStartParticipant1, testStartParticipant2},
					EventMessageID: testStartEventMessageID,
				}

				mockDB.EXPECT().GetRound(ctx, testStartRoundID).Return(round, nil)

				// ✅ Fixed: Implementation calls UpdateRoundState, not UpdateRound
				mockDB.EXPECT().UpdateRoundState(ctx, testStartRoundID, roundtypes.RoundStateInProgress).Return(nil)
			},
			payload: roundevents.RoundStartedPayload{
				RoundID:   testStartRoundID,
				Title:     testRoundTitle,
				Location:  &testStartLocation,
				StartTime: &testStartRoundTime,
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.DiscordRoundStartPayload{
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
				Failure: &roundevents.RoundErrorPayload{
					RoundID: testStartRoundID,
					Error:   "database error",
				},
			},
			expectedError: nil,
		},
		{
			name: "error updating round",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				round := &roundtypes.Round{
					ID:             testStartRoundID,
					Title:          testRoundTitle,
					Location:       &testStartLocation,
					StartTime:      &testStartRoundTime,
					State:          roundtypes.RoundStateUpcoming,
					Participants:   []roundtypes.Participant{testStartParticipant1, testStartParticipant2},
					EventMessageID: testStartEventMessageID,
				}

				mockDB.EXPECT().GetRound(ctx, testStartRoundID).Return(round, nil)
				// ✅ Fixed: Implementation calls UpdateRoundState, not UpdateRound
				mockDB.EXPECT().UpdateRoundState(ctx, testStartRoundID, roundtypes.RoundStateInProgress).Return(errors.New("database error"))
			},
			payload: roundevents.RoundStartedPayload{
				RoundID: testStartRoundID,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundErrorPayload{
					RoundID: testStartRoundID,
					Error:   "database error",
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

			result, err := s.ProcessRoundStart(ctx, tt.payload)

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
				} else {
					if expectedPayload, ok := tt.expectedResult.Success.(*roundevents.DiscordRoundStartPayload); ok {
						if actualPayload, ok := result.Success.(*roundevents.DiscordRoundStartPayload); ok {
							if actualPayload.RoundID != expectedPayload.RoundID {
								t.Errorf("expected RoundID %v, got %v", expectedPayload.RoundID, actualPayload.RoundID)
							}
							if actualPayload.Title != expectedPayload.Title {
								t.Errorf("expected Title %v, got %v", expectedPayload.Title, actualPayload.Title)
							}
							if (actualPayload.Location == nil) != (expectedPayload.Location == nil) {
								t.Errorf("expected Location nil status %v, got %v", expectedPayload.Location == nil, actualPayload.Location == nil)
							} else if actualPayload.Location != nil && expectedPayload.Location != nil && *actualPayload.Location != *expectedPayload.Location {
								t.Errorf("expected Location %v, got %v", *expectedPayload.Location, *actualPayload.Location)
							}
							if (actualPayload.StartTime == nil) != (expectedPayload.StartTime == nil) {
								t.Errorf("expected StartTime nil status %v, got %v", expectedPayload.StartTime == nil, actualPayload.StartTime == nil)
							} else if actualPayload.StartTime != nil && expectedPayload.StartTime != nil && !actualPayload.StartTime.AsTime().Equal(expectedPayload.StartTime.AsTime()) {
								t.Errorf("expected StartTime %v, got %v", expectedPayload.StartTime, actualPayload.StartTime)
							}
							if actualPayload.EventMessageID != expectedPayload.EventMessageID {
								t.Errorf("expected EventMessageID %v, got %v", expectedPayload.EventMessageID, actualPayload.EventMessageID)
							}
							if !reflect.DeepEqual(actualPayload.Participants, expectedPayload.Participants) {
								t.Errorf("expected Participants %v, got %v", expectedPayload.Participants, actualPayload.Participants)
							}
						} else {
							t.Errorf("expected result.Success to be *roundevents.DiscordRoundStartPayload, got %T", result.Success)
						}
					}
				}
			}

			if tt.expectedResult.Failure != nil {
				if result.Failure == nil {
					t.Errorf("expected failure result, got success")
				} else {
					if expectedPayload, ok := tt.expectedResult.Failure.(*roundevents.RoundErrorPayload); ok {
						if actualPayload, ok := result.Failure.(*roundevents.RoundErrorPayload); ok {
							if actualPayload.RoundID != expectedPayload.RoundID {
								t.Errorf("expected Failure RoundID %v, got %v", expectedPayload.RoundID, actualPayload.RoundID)
							}
							if actualPayload.Error != expectedPayload.Error {
								t.Errorf("expected Failure Error %v, got %v", expectedPayload.Error, actualPayload.Error)
							}
						} else {
							t.Errorf("expected result.Failure to be *roundevents.RoundErrorPayload, got %T", result.Failure)
						}
					}
				}
			}
		})
	}
}
