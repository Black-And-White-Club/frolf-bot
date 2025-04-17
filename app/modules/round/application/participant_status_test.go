package roundservice

import (
	"context"
	"errors"
	"fmt"
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

var joinedLateFalse = false

func TestRoundService_CheckParticipantStatus(t *testing.T) {
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

	testRoundID := sharedtypes.RoundID(uuid.New())
	user1ID := sharedtypes.DiscordID("user1")

	tests := []struct {
		name                    string
		mockDBSetup             func(*rounddb.MockRoundDB)
		mockRoundValidatorSetup func(*roundutil.MockRoundValidator)
		mockEventBusSetup       func(*eventbus.MockEventBus)
		payload                 roundevents.ParticipantJoinRequestPayload
		expectedResult          RoundOperationResult
		expectedError           error
	}{
		{
			name: "new participant joining",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// Return nil participant (user not found/not participating yet)
				mockDB.EXPECT().GetParticipant(ctx, testRoundID, user1ID).Return(nil, nil)
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
				// No expectations for the RoundValidator
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
				// No expectations for the EventBus
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:  testRoundID,
				UserID:   user1ID,
				Response: roundtypes.ResponseAccept,
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.ParticipantJoinValidationRequestPayload{
					RoundID:  testRoundID,
					UserID:   user1ID,
					Response: roundtypes.ResponseAccept,
				},
			},
			expectedError: nil,
		},
		{
			name: "participant changing status",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// Return existing participant with different status
				mockDB.EXPECT().GetParticipant(ctx, testRoundID, user1ID).Return(
					&roundtypes.Participant{
						UserID:   user1ID,
						Response: roundtypes.ResponseTentative,
					},
					nil,
				)
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
				// No expectations for the RoundValidator
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
				// No expectations for the EventBus
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:  testRoundID,
				UserID:   user1ID,
				Response: roundtypes.ResponseDecline,
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.ParticipantJoinValidationRequestPayload{
					RoundID:  testRoundID,
					UserID:   user1ID,
					Response: roundtypes.ResponseDecline,
				},
			},
			expectedError: nil,
		},
		{
			name: "toggle participant status (same status clicked)",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// Return existing participant with same status as requested
				mockDB.EXPECT().GetParticipant(ctx, testRoundID, user1ID).Return(
					&roundtypes.Participant{
						UserID:   user1ID,
						Response: roundtypes.ResponseDecline,
					},
					nil,
				)
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
				// No expectations for the RoundValidator
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
				// No expectations for the EventBus
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:  testRoundID,
				UserID:   user1ID,
				Response: roundtypes.ResponseDecline,
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.ParticipantRemovalRequestPayload{
					RoundID: testRoundID,
					UserID:  user1ID,
				},
			},
			expectedError: nil,
		},
		{
			name: "failure checking participant status",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetParticipant(ctx, testRoundID, user1ID).Return(nil, errors.New("db error"))
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
				// No expectations for the RoundValidator
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
				// No expectations for the EventBus
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:  testRoundID,
				UserID:   user1ID,
				Response: roundtypes.ResponseAccept,
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.ParticipantStatusCheckErrorPayload{
					RoundID: testRoundID,
					UserID:  user1ID,
					Error:   "failed to get participant status: db error",
				},
			},
			expectedError: errors.New("failed to get participant status: db error"),
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

			result, err := s.CheckParticipantStatus(ctx, tt.payload)

			// Validate error
			if tt.expectedError != nil {
				if err == nil || err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}

			// Validate result
			if err == nil {
				// For success cases, check specific payload types and values
				switch expected := tt.expectedResult.Success.(type) {
				case roundevents.ParticipantJoinValidationRequestPayload:
					if actual, ok := result.Success.(roundevents.ParticipantJoinValidationRequestPayload); ok {
						if actual.RoundID != expected.RoundID || actual.UserID != expected.UserID || actual.Response != expected.Response {
							t.Errorf("expected validation payload: %+v, got: %+v", expected, actual)
						}
					} else {
						t.Errorf("expected ParticipantJoinValidationRequestPayload, got: %T", result.Success)
					}
				case roundevents.ParticipantRemovalRequestPayload:
					if actual, ok := result.Success.(roundevents.ParticipantRemovalRequestPayload); ok {
						if actual.RoundID != expected.RoundID || actual.UserID != expected.UserID {
							t.Errorf("expected removal payload: %+v, got: %+v", expected, actual)
						}
					} else {
						t.Errorf("expected ParticipantRemovalRequestPayload, got: %T", result.Success)
					}
				}
			} else {
				// For failure cases, check error payload
				if expected, ok := tt.expectedResult.Failure.(roundevents.ParticipantStatusCheckErrorPayload); ok {
					if actual, ok := result.Failure.(roundevents.ParticipantStatusCheckErrorPayload); ok {
						if actual.RoundID != expected.RoundID || actual.UserID != expected.UserID || actual.Error != expected.Error {
							t.Errorf("expected error payload: %+v, got: %+v", expected, actual)
						}
					} else {
						t.Errorf("expected ParticipantStatusCheckErrorPayload, got: %T", result.Failure)
					}
				}
			}
		})
	}
}

func TestRoundService_ValidateParticipantJoinRequest(t *testing.T) {
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

	testRoundID := sharedtypes.RoundID(uuid.New())
	user1ID := sharedtypes.DiscordID("user1")

	tests := []struct {
		name                    string
		mockDBSetup             func(*rounddb.MockRoundDB)
		mockRoundValidatorSetup func(*roundutil.MockRoundValidator)
		mockEventBusSetup       func(*eventbus.MockEventBus)
		payload                 roundevents.ParticipantJoinRequestPayload
		expectedResult          RoundOperationResult
		expectedError           error
	}{
		{
			name: "success validating participant join request",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(&roundtypes.Round{ID: testRoundID, State: roundtypes.RoundStateUpcoming}, nil)
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
				// No expectations for the RoundValidator
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
				// No expectations for the EventBus
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:  testRoundID,
				UserID:   user1ID,
				Response: roundtypes.ResponseAccept,
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.ParticipantJoinRequestPayload{
					RoundID:    testRoundID,
					UserID:     user1ID,
					Response:   roundtypes.ResponseAccept,
					JoinedLate: &joinedLateFalse,
				},
			},
			expectedError: nil,
		},
		{
			name: "failure validating participant join request",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(nil, errors.New("db error"))
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
				// No expectations for the RoundValidator
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
				// No expectations for the EventBus
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:  testRoundID,
				UserID:   user1ID,
				Response: roundtypes.ResponseAccept,
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.RoundParticipantJoinErrorPayload{
					ParticipantJoinRequest: &roundevents.ParticipantJoinRequestPayload{
						RoundID:  testRoundID,
						UserID:   user1ID,
						Response: roundtypes.ResponseAccept,
					},
					Error: "failed to fetch round details: db error",
				},
			},
			expectedError: errors.New("failed to fetch round details: db error"),
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

			_, err := s.ValidateParticipantJoinRequest(ctx, tt.payload)

			// Validate error
			if tt.expectedError != nil {
				if err == nil || err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}

			// Validate result
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

func TestRoundService_ParticipantRemoval(t *testing.T) {
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

	testRoundID := sharedtypes.RoundID(uuid.New())
	user1ID := sharedtypes.DiscordID("user1")

	tests := []struct {
		name                    string
		mockDBSetup             func(*rounddb.MockRoundDB)
		mockRoundValidatorSetup func(*roundutil.MockRoundValidator)
		mockEventBusSetup       func(*eventbus.MockEventBus)
		payload                 roundevents.ParticipantRemovalRequestPayload
		expectedResult          RoundOperationResult
		expectedError           error
	}{
		{
			name: "success removing participant",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(&roundtypes.Round{ID: testRoundID, EventMessageID: sharedtypes.RoundID(uuid.New())}, nil)
				mockDB.EXPECT().GetParticipant(ctx, testRoundID, user1ID).Return(&roundtypes.Participant{UserID: user1ID, Response: roundtypes.ResponseAccept}, nil)
				mockDB.EXPECT().RemoveParticipant(ctx, testRoundID, user1ID).Return(nil)
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
				// No expectations for the RoundValidator
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
				// No expectations for the EventBus
			},
			payload: roundevents.ParticipantRemovalRequestPayload{
				RoundID: testRoundID,
				UserID:  user1ID,
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.ParticipantRemovedPayload{
					RoundID:        testRoundID,
					UserID:         user1ID,
					Response:       roundtypes.ResponseAccept,
					EventMessageID: sharedtypes.RoundID(uuid.New()),
				},
			},
			expectedError: nil,
		},
		{
			name: "failure removing participant",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(nil, errors.New("db error"))
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
				// No expectations for the RoundValidator
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
				// No expectations for the EventBus
			},
			payload: roundevents.ParticipantRemovalRequestPayload{
				RoundID: testRoundID,
				UserID:  user1ID,
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.ParticipantRemovalErrorPayload{
					RoundID: testRoundID,
					UserID:  user1ID,
					Error:   "failed to fetch round details: db error",
				},
			},
			expectedError: errors.New("failed to fetch round details: db error"),
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

			_, err := s.ParticipantRemoval(ctx, tt.payload)

			// Validate error
			if tt.expectedError != nil {
				if err == nil || err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}

			// Validate result
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

func TestRoundService_UpdateParticipantStatus(t *testing.T) {
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

	testRoundID := sharedtypes.RoundID(uuid.New())
	user1ID := sharedtypes.DiscordID("user1")

	tests := []struct {
		name                    string
		mockDBSetup             func(*rounddb.MockRoundDB)
		mockRoundValidatorSetup func(*roundutil.MockRoundValidator)
		mockEventBusSetup       func(*eventbus.MockEventBus)
		payload                 roundevents.ParticipantJoinRequestPayload
		expectedResult          RoundOperationResult
		expectedError           error
	}{
		{
			name: "success updating participant status",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(gomock.Any(), testRoundID).Return(&roundtypes.Round{ID: testRoundID, EventMessageID: sharedtypes.RoundID(uuid.New())}, nil)
				mockDB.EXPECT().UpdateParticipant(gomock.Any(), testRoundID, roundtypes.Participant{UserID: user1ID, Response: roundtypes.ResponseDecline}).Return([]roundtypes.Participant{
					{UserID: user1ID, Response: roundtypes.ResponseDecline},
				}, nil)
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
				// No expectations for the RoundValidator
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
				// No expectations for the EventBus
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:  testRoundID,
				UserID:   user1ID,
				Response: roundtypes.ResponseDecline,
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.ParticipantJoinedPayload{
					RoundID:               testRoundID,
					AcceptedParticipants:  []roundtypes.Participant{},
					DeclinedParticipants:  []roundtypes.Participant{{UserID: user1ID, Response: roundtypes.ResponseDecline}},
					TentativeParticipants: []roundtypes.Participant{},
					EventMessageID:        sharedtypes.RoundID(uuid.New()),
					JoinedLate:            &joinedLateFalse,
				},
			},
			expectedError: nil,
		},
		{
			name: "failure updating participant status",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(nil, errors.New("db error"))
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
				// No expectations for the RoundValidator
			},
			mockEventBusSetup: func(mockEventBus *eventbus.MockEventBus) {
				// No expectations for the EventBus
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:  testRoundID,
				UserID:   user1ID,
				Response: roundtypes.ResponseDecline,
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.ParticipantUpdateErrorPayload{
					RoundID: testRoundID,
					UserID:  user1ID,
					Error:   "failed to fetch round details: db error",
				},
			},
			expectedError: errors.New("failed to fetch round details: db error"),
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

			result, err := s.UpdateParticipantStatus(ctx, tt.payload)

			// Validate error
			if tt.expectedError != nil {
				if err == nil || err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}

			// Validate result
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			// Add this validation logic to your test:
			if tt.expectedError == nil {
				// Only validate success case details when no error is expected
				successPayload, ok := result.Success.(roundevents.ParticipantJoinedPayload)
				if !ok {
					t.Errorf("expected success payload of type ParticipantJoinedPayload, got %T", result.Success)
					return
				}

				// Validate important fields
				if successPayload.RoundID != tt.payload.RoundID {
					t.Errorf("expected RoundID: %v, got: %v", tt.payload.RoundID, successPayload.RoundID)
				}

				// Check that we have the correct participant in the declined list
				foundParticipant := false
				for _, p := range successPayload.DeclinedParticipants {
					if p.UserID == tt.payload.UserID && p.Response == tt.payload.Response {
						foundParticipant = true
						break
					}
				}

				if !foundParticipant {
					t.Errorf("expected to find participant with UserID: %v and Response: %v in declined list",
						tt.payload.UserID, tt.payload.Response)
				}

				// Validate JoinedLate is correctly set
				if (successPayload.JoinedLate == nil && tt.payload.JoinedLate != nil) ||
					(successPayload.JoinedLate != nil && tt.payload.JoinedLate == nil) ||
					(successPayload.JoinedLate != nil && tt.payload.JoinedLate != nil &&
						*successPayload.JoinedLate != *tt.payload.JoinedLate) {
					t.Errorf("expected JoinedLate: %v, got: %v",
						formatBoolPtr(tt.payload.JoinedLate),
						formatBoolPtr(successPayload.JoinedLate))
				}
			} else if tt.expectedError != nil {
				// For failure cases, validate the error payload
				switch failure := result.Failure.(type) {
				case roundevents.ParticipantUpdateErrorPayload:
					if failure.RoundID != tt.payload.RoundID {
						t.Errorf("expected RoundID: %v, got: %v", tt.payload.RoundID, failure.RoundID)
					}
					if failure.UserID != tt.payload.UserID {
						t.Errorf("expected UserID: %v, got: %v", tt.payload.UserID, failure.UserID)
					}
				case roundevents.RoundParticipantJoinErrorPayload:
					if failure.ParticipantJoinRequest.RoundID != tt.payload.RoundID {
						t.Errorf("expected RoundID: %v, got: %v", tt.payload.RoundID, failure.ParticipantJoinRequest.RoundID)
					}
				default:
					t.Errorf("unexpected failure payload type: %T", result.Failure)
				}
			}
		})
	}
}

// Helper function for formatting bool pointers
func formatBoolPtr(b *bool) string {
	if b == nil {
		return "nil"
	}
	return fmt.Sprintf("%t", *b)
}
