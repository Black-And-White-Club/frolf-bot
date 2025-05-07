package roundservice

import (
	"context"
	"errors"
	"testing"

	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestRoundService_ValidateRoundDeleteRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Mock dependencies
	mockDB := rounddb.NewMockRoundDB(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockRoundValidator := &roundutil.MockRoundValidator{}
	mockEventBus := &eventbus.MockEventBus{}

	tests := []struct {
		name                    string
		mockDBSetup             func(*rounddb.MockRoundDB)
		mockRoundValidatorSetup func(*roundutil.MockRoundValidator)
		mockEventBusSetup       func(*eventbus.MockEventBus)
		payload                 roundevents.RoundDeleteRequestPayload
		expectedResult          RoundOperationResult
		expectedError           error
	}{
		{
			name: "valid round delete request",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
			},
			payload: roundevents.RoundDeleteRequestPayload{
				RoundID:              sharedtypes.RoundID(uuid.New()),
				RequestingUserUserID: "Test User",
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.RoundDeleteValidatedPayload{
					RoundDeleteRequestPayload: roundevents.RoundDeleteRequestPayload{
						RoundID:              sharedtypes.RoundID(uuid.New()),
						RequestingUserUserID: "Test User",
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "invalid round delete request - zero round ID",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
			},
			payload: roundevents.RoundDeleteRequestPayload{
				RoundID:              sharedtypes.RoundID(uuid.Nil),
				RequestingUserUserID: "Test User",
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundDeleteErrorPayload{
					RoundDeleteRequest: &roundevents.RoundDeleteRequestPayload{
						RoundID:              sharedtypes.RoundID(uuid.Nil),
						RequestingUserUserID: "Test User",
					},
					Error: "round ID cannot be zero",
				},
			},
			expectedError: errors.New("round ID cannot be zero"),
		},
		{
			name: "invalid round delete request - empty requesting user ID",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
			},
			payload: roundevents.RoundDeleteRequestPayload{
				RoundID:              sharedtypes.RoundID(uuid.New()),
				RequestingUserUserID: "",
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundDeleteErrorPayload{
					RoundDeleteRequest: &roundevents.RoundDeleteRequestPayload{
						RoundID:              sharedtypes.RoundID(uuid.New()),
						RequestingUserUserID: "",
					},
					Error: "requesting user's Discord ID cannot be empty",
				},
			},
			expectedError: errors.New("requesting user's Discord ID cannot be empty"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockDBSetup(mockDB)
			tt.mockRoundValidatorSetup(mockRoundValidator)
			tt.mockEventBusSetup(mockEventBus)

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

			_, err := s.ValidateRoundDeleteRequest(ctx, tt.payload)

			// Validate error presence
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
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

			}
		})
	}
}

func TestRoundService_DeleteRound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Mock dependencies
	mockDB := rounddb.NewMockRoundDB(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockRoundValidator := &roundutil.MockRoundValidator{}
	mockEventBus := eventbus.NewMockEventBus(ctrl)

	id := sharedtypes.RoundID(uuid.New())

	tests := []struct {
		name                    string
		mockDBSetup             func(*rounddb.MockRoundDB)
		mockRoundValidatorSetup func(*roundutil.MockRoundValidator)
		mockEventBusSetup       func(*eventbus.MockEventBus)
		payload                 roundevents.RoundDeleteAuthorizedPayload
		expectedResult          RoundOperationResult
		expectedError           error
	}{
		{
			name: "delete round successfully",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetEventMessageID(ctx, gomock.Any()).Return(&id, nil)
				mockDB.EXPECT().DeleteRound(ctx, gomock.Any()).Return(nil)
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
				mockEventBus.EXPECT().CancelScheduledMessage(ctx, id).Return(nil)
			},
			payload: roundevents.RoundDeleteAuthorizedPayload{
				RoundID: id,
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.RoundDeletedPayload{
					RoundID: id,
				},
			},
			expectedError: nil,
		},
		{
			name: "delete round fails - get event message ID error",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetEventMessageID(ctx, gomock.Any()).Return(nil, errors.New("get event message ID error"))
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
			},
			payload: roundevents.RoundDeleteAuthorizedPayload{
				RoundID: id,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundDeleteErrorPayload{
					RoundDeleteRequest: &roundevents.RoundDeleteRequestPayload{
						RoundID:              id,
						RequestingUserUserID: "",
					},
					Error: "get event message ID error",
				},
			},
			expectedError: errors.New("failed to retrieve EventMessageID for round"),
		},
		{
			name: "delete round fails - delete round error",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetEventMessageID(ctx, gomock.Any()).Return(&id, nil)
				mockDB.EXPECT().DeleteRound(ctx, gomock.Any()).Return(errors.New("delete round error"))
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
			},
			payload: roundevents.RoundDeleteAuthorizedPayload{
				RoundID: id,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundDeleteErrorPayload{
					RoundDeleteRequest: &roundevents.RoundDeleteRequestPayload{
						RoundID:              id,
						RequestingUserUserID: "",
					},
					Error: "delete round error",
				},
			},
			expectedError: errors.New("failed to delete round"),
		},
		{
			name: "delete round fails - cancel scheduled message error",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetEventMessageID(ctx, gomock.Any()).Return(&id, nil)
				mockDB.EXPECT().DeleteRound(ctx, gomock.Any()).Return(nil)
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
				mockEventBus.EXPECT().CancelScheduledMessage(ctx, id).Return(errors.New("cancel scheduled message error"))
			},
			payload: roundevents.RoundDeleteAuthorizedPayload{
				RoundID: id,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundDeleteErrorPayload{
					RoundDeleteRequest: &roundevents.RoundDeleteRequestPayload{
						RoundID:              id,
						RequestingUserUserID: "",
					},
					Error: "cancel scheduled message error",
				},
			},
			expectedError: errors.New("failed to cancel scheduled messages"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockDBSetup(mockDB)
			tt.mockRoundValidatorSetup(mockRoundValidator)
			tt.mockEventBusSetup(mockEventBus)

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

			_, err := s.DeleteRound(ctx, tt.payload)

			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
		})
	}
}
