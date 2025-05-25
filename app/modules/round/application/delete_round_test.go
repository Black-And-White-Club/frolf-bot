package roundservice

import (
	"context"
	"errors"
	"fmt" // Import fmt for error string matching
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
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockRoundValidator := &roundutil.MockRoundValidator{}

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
				// No specific DB mocks needed for validation
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
				// No specific validator mocks needed for validation
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
				// No specific event bus mocks needed for validation
			},
			payload: roundevents.RoundDeleteRequestPayload{
				RoundID:              sharedtypes.RoundID(uuid.New()),
				RequestingUserUserID: "Test User",
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.RoundDeleteValidatedPayload{
					RoundDeleteRequestPayload: roundevents.RoundDeleteRequestPayload{
						RoundID:              sharedtypes.RoundID(uuid.New()), // Note: This RoundID should match the payload's RoundID for exact comparison
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
			expectedError: fmt.Errorf("ValidateRoundDeleteRequest operation failed: %w", errors.New("round ID cannot be zero")),
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
						RoundID:              sharedtypes.RoundID(uuid.New()), // Note: This RoundID should match the payload's RoundID for exact comparison
						RequestingUserUserID: "",
					},
					Error: "requesting user's Discord ID cannot be empty",
				},
			},
			expectedError: fmt.Errorf("ValidateRoundDeleteRequest operation failed: %w", errors.New("requesting user's Discord ID cannot be empty")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks for each subtest
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockDB := rounddb.NewMockRoundDB(ctrl)
			mockEventBus := eventbus.NewMockEventBus(ctrl)

			tt.mockDBSetup(mockDB)
			tt.mockRoundValidatorSetup(mockRoundValidator) // This mock is not used in ValidateRoundDeleteRequest, but kept for consistency
			tt.mockEventBusSetup(mockEventBus)

			s := &RoundService{
				RoundDB:        mockDB,
				logger:         logger,
				metrics:        mockMetrics,
				tracer:         tracer,
				roundValidator: mockRoundValidator,
				EventBus:       mockEventBus,
				serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (RoundOperationResult, error) {
					result, err := serviceFunc(ctx)
					if err != nil {
						// Mimic the actual serviceWrapper's error wrapping
						return result, fmt.Errorf("%s operation failed: %w", operationName, err)
					}
					return result, nil
				},
			}

			_, err := s.ValidateRoundDeleteRequest(ctx, tt.payload)

			// Validate error presence and message
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else {
				if err != nil {
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
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockRoundValidator := &roundutil.MockRoundValidator{}

	id := sharedtypes.RoundID(uuid.New()) // Use a consistent UUID for tests

	tests := []struct {
		name              string
		mockDBSetup       func(*rounddb.MockRoundDB)
		mockEventBusSetup func(*eventbus.MockEventBus)
		payload           roundevents.RoundDeleteAuthorizedPayload
		expectedResult    RoundOperationResult
		expectedError     error
	}{
		{
			name: "delete round successfully",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// No GetEventMessageID call in service, so no mock needed
				mockDB.EXPECT().DeleteRound(ctx, gomock.Eq(id)).Return(nil)
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
				mockEventBus.EXPECT().CancelScheduledMessage(ctx, gomock.Eq(id)).Return(nil)
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
			name: "delete round fails - delete round error",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// No GetEventMessageID call in service, so no mock needed
				mockDB.EXPECT().DeleteRound(ctx, gomock.Eq(id)).Return(errors.New("delete round error"))
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
				// CancelScheduledMessage should not be called if DeleteRound fails
			},
			payload: roundevents.RoundDeleteAuthorizedPayload{
				RoundID: id,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundDeleteErrorPayload{
					RoundDeleteRequest: &roundevents.RoundDeleteRequestPayload{
						RoundID: id,
						// RequestingUserUserID is not part of DeleteRoundAuthorizedPayload, so it will be empty
						RequestingUserUserID: "",
					},
					Error: fmt.Sprintf("failed to delete round from database: %v", errors.New("delete round error")),
				},
			},
			expectedError: fmt.Errorf("failed to delete round %s from DB: %w", id.String(), errors.New("delete round error")),
		},
		{
			name: "delete round succeeds despite cancel scheduled message error",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// No GetEventMessageID call in service, so no mock needed
				mockDB.EXPECT().DeleteRound(ctx, gomock.Eq(id)).Return(nil)
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
				mockEventBus.EXPECT().CancelScheduledMessage(ctx, gomock.Eq(id)).Return(errors.New("cancel scheduled message error"))
			},
			payload: roundevents.RoundDeleteAuthorizedPayload{
				RoundID: id,
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.RoundDeletedPayload{
					RoundID: id,
				},
			},
			expectedError: nil, // The service logs a warning but returns nil error for the overall operation
		},
		{
			name: "delete round fails - nil UUID provided",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// No DB calls expected if UUID is nil
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
				// No EventBus calls expected if UUID is nil
			},
			payload: roundevents.RoundDeleteAuthorizedPayload{
				RoundID: sharedtypes.RoundID(uuid.Nil),
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundDeleteErrorPayload{
					RoundDeleteRequest: &roundevents.RoundDeleteRequestPayload{
						RoundID: sharedtypes.RoundID(uuid.Nil),
					},
					Error: "round ID cannot be nil",
				},
			},
			expectedError: errors.New("cannot delete round: nil UUID provided"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks for each subtest
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockDB := rounddb.NewMockRoundDB(ctrl)
			mockEventBus := eventbus.NewMockEventBus(ctrl)

			tt.mockDBSetup(mockDB)
			tt.mockEventBusSetup(mockEventBus)

			s := &RoundService{
				RoundDB:        mockDB,
				logger:         logger,
				metrics:        mockMetrics,
				tracer:         tracer,
				roundValidator: mockRoundValidator, // This mock is not used in DeleteRound, but kept for consistency
				EventBus:       mockEventBus,
				serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (RoundOperationResult, error) {
					result, err := serviceFunc(ctx)
					if err != nil {
						// Mimic the actual serviceWrapper's error wrapping
						return result, fmt.Errorf("%s operation failed: %w", operationName, err)
					}
					return result, nil
				},
			}

			result, err := s.DeleteRound(ctx, tt.payload)

			// Validate error presence and message
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}

			// Validate result payload (success/failure)
			if tt.expectedResult.Success != nil {
				// Type assertion for Success payload
				if result.Success == nil {
					t.Errorf("expected success result, got failure")
				} else if successPayload, ok := result.Success.(*roundevents.RoundDeletedPayload); !ok {
					t.Errorf("expected result.Success to be of type *roundevents.RoundDeletedPayload, got %T", result.Success)
				} else if expectedSuccessPayload, ok := tt.expectedResult.Success.(*roundevents.RoundDeletedPayload); !ok {
					t.Errorf("expected tt.expectedResult.Success to be of type *roundevents.RoundDeletedPayload, got %T", tt.expectedResult.Success)
				} else if successPayload.RoundID != expectedSuccessPayload.RoundID {
					t.Errorf("expected success RoundID %s, got %s", expectedSuccessPayload.RoundID, successPayload.RoundID)
				}
			} else if tt.expectedResult.Failure != nil {
				// Type assertion for Failure payload
				if result.Failure == nil {
					t.Errorf("expected failure result, got success")
				} else if failurePayload, ok := result.Failure.(*roundevents.RoundDeleteErrorPayload); !ok {
					t.Errorf("expected result.Failure to be of type *roundevents.RoundDeleteErrorPayload, got %T", result.Failure)
				} else if expectedFailurePayload, ok := tt.expectedResult.Failure.(*roundevents.RoundDeleteErrorPayload); !ok {
					t.Errorf("expected tt.expectedResult.Failure to be of type *roundevents.RoundDeleteErrorPayload, got %T", tt.expectedResult.Failure)
				} else if failurePayload.Error != expectedFailurePayload.Error {
					t.Errorf("expected failure error %q, got %q", expectedFailurePayload.Error, failurePayload.Error)
				}
			}
		})
	}
}
