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
	testRoundID   = sharedtypes.RoundID(uuid.New())
	testTitle     = roundtypes.Title("Test Round")
	testDesc      = roundtypes.Description("Test Description")
	testLoc       = roundtypes.Location("Test Location")
	testEventType = roundtypes.EventType("Test Event Type")
	testStartTime = sharedtypes.StartTime(time.Now())
	testCreatorID = sharedtypes.DiscordID("Test User")
	testState     = roundtypes.RoundState("Test State")
)

var testRound = roundtypes.Round{
	ID:           testRoundID,
	Title:        testTitle,
	Description:  &testDesc,
	Location:     &testLoc,
	EventType:    &testEventType,
	StartTime:    &testStartTime,
	Finalized:    false,
	CreatedBy:    testCreatorID,
	State:        testState,
	Participants: []roundtypes.Participant{},
}

func TestRoundService_GetRound(t *testing.T) {
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
		expectedResult RoundOperationResult
		expectedError  error
	}{
		{
			name: "successful retrieval",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				guildID := sharedtypes.GuildID("guild-123")
				mockDB.EXPECT().GetRound(ctx, guildID, testRoundID).Return(&testRound, nil)
			},
			expectedResult: RoundOperationResult{
				Success: &testRound,
			},
			expectedError: nil,
		},
		{
			name: "error retrieving round",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				guildID := sharedtypes.GuildID("guild-123")
				mockDB.EXPECT().GetRound(ctx, guildID, testRoundID).Return(&roundtypes.Round{}, errors.New("database error"))
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					RoundID: testRoundID,
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

			guildID := sharedtypes.GuildID("guild-123")
			result, err := s.GetRound(ctx, guildID, testRoundID)

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
				} else {
					// Compare the actual round data
					expectedRound := tt.expectedResult.Success.(*roundtypes.Round)
					actualRound := result.Success.(*roundtypes.Round)
					if expectedRound.ID != actualRound.ID {
						t.Errorf("expected RoundID: %v, got: %v", expectedRound.ID, actualRound.ID)
					}
					// Add more field comparisons as needed
				}
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
