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
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(&testRound, nil)
			},
			expectedResult: RoundOperationResult{
				Success: testRound,
			},
			expectedError: nil,
		},
		{
			name: "error retrieving round",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(&roundtypes.Round{}, errors.New("database error"))
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundErrorPayload{
					RoundID: testRoundID,
					Error:   "database error",
				},
			},
			expectedError: errors.New("failed to retrieve round: database error"),
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

			_, err := s.GetRound(ctx, testRoundID)
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
