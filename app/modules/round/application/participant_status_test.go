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

var (
	testUserID           = sharedtypes.DiscordID("user1")
	testEventMessageID   = "discord_message_id_123"
	testParticipantScore = sharedtypes.Score(10)
	testTagNumber        = sharedtypes.TagNumber(1)
	joinedLateFalse      = false // Define the variable
	joinedLateTrue       = true  // Define the variable
)

func TestRoundService_CheckParticipantStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)

	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("test-user-123")

	tests := []struct {
		name           string
		mockDBSetup    func(*rounddb.MockRoundDB)
		payload        roundevents.ParticipantJoinRequestPayload
		expectedResult RoundOperationResult
		expectedError  error
	}{
		{
			name: "new participant joining",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// GetParticipant returns nil (participant not found)
				mockDB.EXPECT().GetParticipant(ctx, testRoundID, testUserID).Return(nil, nil)
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:  testRoundID,
				UserID:   testUserID,
				Response: roundtypes.ResponseAccept,
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.ParticipantJoinValidationRequestPayload{
					RoundID:  testRoundID,
					UserID:   testUserID,
					Response: roundtypes.ResponseAccept,
				},
			},
			expectedError: nil,
		},
		{
			name: "participant changing status",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// GetParticipant returns existing participant with different status
				existingParticipant := &roundtypes.Participant{
					UserID:   testUserID,
					Response: roundtypes.ResponseTentative,
				}
				mockDB.EXPECT().GetParticipant(ctx, testRoundID, testUserID).Return(existingParticipant, nil)
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:  testRoundID,
				UserID:   testUserID,
				Response: roundtypes.ResponseAccept, // Different from existing status
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.ParticipantJoinValidationRequestPayload{
					RoundID:  testRoundID,
					UserID:   testUserID,
					Response: roundtypes.ResponseAccept,
				},
			},
			expectedError: nil,
		},
		{
			name: "toggle participant status (same status clicked)",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// GetParticipant returns existing participant with same status
				existingParticipant := &roundtypes.Participant{
					UserID:   testUserID,
					Response: roundtypes.ResponseAccept,
				}
				mockDB.EXPECT().GetParticipant(ctx, testRoundID, testUserID).Return(existingParticipant, nil)
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:  testRoundID,
				UserID:   testUserID,
				Response: roundtypes.ResponseAccept, // Same as existing status
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.ParticipantRemovalRequestPayload{
					RoundID: testRoundID,
					UserID:  testUserID,
				},
			},
			expectedError: nil,
		},
		{
			name: "failure checking participant status",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// GetParticipant returns error
				mockDB.EXPECT().GetParticipant(ctx, testRoundID, testUserID).Return(nil, errors.New("db error"))
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:  testRoundID,
				UserID:   testUserID,
				Response: roundtypes.ResponseAccept,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.ParticipantStatusCheckErrorPayload{
					RoundID: testRoundID,
					UserID:  testUserID,
					Error:   "failed to get participant status: db error",
				},
			},
			expectedError: nil, // Service returns failure payload with nil error
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

			// Validate result payload
			if tt.expectedResult.Success != nil {
				if result.Success == nil {
					t.Errorf("expected success result, got failure")
				} else {
					// Check the specific payload type
					switch expectedPayload := tt.expectedResult.Success.(type) {
					case *roundevents.ParticipantJoinValidationRequestPayload:
						if actualPayload, ok := result.Success.(*roundevents.ParticipantJoinValidationRequestPayload); !ok {
							t.Errorf("expected *roundevents.ParticipantJoinValidationRequestPayload, got: %T", result.Success)
						} else {
							if actualPayload.RoundID != expectedPayload.RoundID {
								t.Errorf("expected RoundID %s, got %s", expectedPayload.RoundID, actualPayload.RoundID)
							}
							if actualPayload.UserID != expectedPayload.UserID {
								t.Errorf("expected UserID %s, got %s", expectedPayload.UserID, actualPayload.UserID)
							}
							if actualPayload.Response != expectedPayload.Response {
								t.Errorf("expected Response %s, got %s", expectedPayload.Response, actualPayload.Response)
							}
						}
					case *roundevents.ParticipantRemovalRequestPayload:
						if actualPayload, ok := result.Success.(*roundevents.ParticipantRemovalRequestPayload); !ok {
							t.Errorf("expected *roundevents.ParticipantRemovalRequestPayload, got: %T", result.Success)
						} else {
							if actualPayload.RoundID != expectedPayload.RoundID {
								t.Errorf("expected RoundID %s, got %s", expectedPayload.RoundID, actualPayload.RoundID)
							}
							if actualPayload.UserID != expectedPayload.UserID {
								t.Errorf("expected UserID %s, got %s", expectedPayload.UserID, actualPayload.UserID)
							}
						}
					default:
						t.Errorf("unexpected success payload type: %T", expectedPayload)
					}
				}
			} else if tt.expectedResult.Failure != nil {
				if result.Failure == nil {
					t.Errorf("expected failure result, got success")
				} else if failurePayload, ok := result.Failure.(*roundevents.ParticipantStatusCheckErrorPayload); !ok {
					t.Errorf("expected *roundevents.ParticipantStatusCheckErrorPayload, got: %T", result.Failure)
				} else if expectedFailurePayload, ok := tt.expectedResult.Failure.(*roundevents.ParticipantStatusCheckErrorPayload); !ok {
					t.Errorf("expected tt.expectedResult.Failure to be *roundevents.ParticipantStatusCheckErrorPayload, got: %T", tt.expectedResult.Failure)
				} else {
					if failurePayload.RoundID != expectedFailurePayload.RoundID {
						t.Errorf("expected failure RoundID %s, got %s", expectedFailurePayload.RoundID, failurePayload.RoundID)
					}
					if failurePayload.UserID != expectedFailurePayload.UserID {
						t.Errorf("expected failure UserID %s, got %s", expectedFailurePayload.UserID, failurePayload.UserID)
					}
					if failurePayload.Error != expectedFailurePayload.Error {
						t.Errorf("expected failure error %q, got %q", expectedFailurePayload.Error, failurePayload.Error)
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
			expectedError: nil, // <-- CHANGED: error is returned in Failure payload, not as error return value
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

			result, err := s.ValidateParticipantJoinRequest(ctx, tt.payload)

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

			// Additional: Validate the failure payload for the error case
			if tt.expectedResult.Failure != nil {
				if result.Failure == nil {
					t.Errorf("expected failure result, got nil failure payload")
				} else {
					expectedFailure, ok := tt.expectedResult.Failure.(roundevents.RoundParticipantJoinErrorPayload)
					actualFailure, ok2 := result.Failure.(roundevents.RoundParticipantJoinErrorPayload)
					if ok && ok2 {
						if expectedFailure.Error != actualFailure.Error {
							t.Errorf("expected failure error: %q, got: %q", expectedFailure.Error, actualFailure.Error)
						}
					}
				}
			}
		})
	}
}

func TestRoundService_ParticipantRemoval(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}

	testRoundID := sharedtypes.RoundID(uuid.New())

	tests := []struct {
		name           string
		mockDBSetup    func(*rounddb.MockRoundDB)
		payload        roundevents.ParticipantRemovalRequestPayload
		expectedResult RoundOperationResult
		expectedError  error
	}{
		{
			name: "success removing participant",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(&roundtypes.Round{
					ID:             testRoundID,
					EventMessageID: testEventMessageID,
					Participants: []roundtypes.Participant{
						{UserID: testUserID, Response: roundtypes.ResponseAccept},
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept},
					},
				}, nil).Times(1)
				mockDB.EXPECT().RemoveParticipant(ctx, testRoundID, testUserID).Return([]roundtypes.Participant{
					{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept},
				}, nil).Times(1)
			},
			payload: roundevents.ParticipantRemovalRequestPayload{
				RoundID: testRoundID,
				UserID:  testUserID,
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.ParticipantRemovedPayload{
					RoundID:        testRoundID,
					UserID:         testUserID,
					EventMessageID: testEventMessageID,
					AcceptedParticipants: []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept},
					},
					DeclinedParticipants:  []roundtypes.Participant{},
					TentativeParticipants: []roundtypes.Participant{},
				},
			},
			expectedError: nil,
		},
		{
			name: "success when participant not found",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(&roundtypes.Round{
					ID:             testRoundID,
					EventMessageID: testEventMessageID,
					Participants: []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept},
					},
				}, nil).Times(1)
				mockDB.EXPECT().RemoveParticipant(ctx, testRoundID, testUserID).Return([]roundtypes.Participant{
					{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept},
				}, nil).Times(1)
			},
			payload: roundevents.ParticipantRemovalRequestPayload{
				RoundID: testRoundID,
				UserID:  testUserID,
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.ParticipantRemovedPayload{
					RoundID:               testRoundID,
					UserID:                testUserID,
					EventMessageID:        testEventMessageID,
					AcceptedParticipants:  []roundtypes.Participant{{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept}},
					DeclinedParticipants:  []roundtypes.Participant{},
					TentativeParticipants: []roundtypes.Participant{},
				},
			},
			expectedError: nil,
		},
		{
			name: "failure fetching round details before removal",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(nil, errors.New("db error")).Times(1)
			},
			payload: roundevents.ParticipantRemovalRequestPayload{
				RoundID: testRoundID,
				UserID:  testUserID,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.ParticipantRemovalErrorPayload{
					RoundID: testRoundID,
					UserID:  testUserID,
					Error:   "failed to fetch round details: db error",
				},
			},
			expectedError: nil, // Service returns failure payload with nil error
		},
		{
			name: "failure removing participant from database",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(&roundtypes.Round{
					ID:             testRoundID,
					EventMessageID: testEventMessageID,
					Participants: []roundtypes.Participant{
						{UserID: testUserID, Response: roundtypes.ResponseAccept},
					},
				}, nil).Times(1)
				mockDB.EXPECT().RemoveParticipant(ctx, testRoundID, testUserID).Return(nil, errors.New("db remove error")).Times(1)
			},
			payload: roundevents.ParticipantRemovalRequestPayload{
				RoundID: testRoundID,
				UserID:  testUserID,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.ParticipantRemovalErrorPayload{
					RoundID: testRoundID,
					UserID:  testUserID,
					Error:   "failed to remove participant: db remove error",
				},
			},
			expectedError: nil, // Service returns failure payload with nil error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockDB := rounddb.NewMockRoundDB(ctrl)
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

			result, err := s.ParticipantRemoval(ctx, tt.payload)

			if (err != nil) != (tt.expectedError != nil) {
				t.Fatalf("expected error presence %v, got %v (error: %v)", tt.expectedError != nil, err != nil, err)
			}
			if err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
			}

			// Determine if this should be a success or failure case based on expectedResult
			if tt.expectedResult.Success != nil {
				if result.Success == nil {
					t.Errorf("expected success result, got nil success payload")
					return
				}
				successPayload, ok := result.Success.(*roundevents.ParticipantRemovedPayload)
				if !ok {
					t.Errorf("expected success payload of type *ParticipantRemovedPayload, got %T", result.Success)
					return
				}
				expectedSuccessPayload, ok := tt.expectedResult.Success.(*roundevents.ParticipantRemovedPayload)
				if !ok {
					t.Fatalf("test setup error: expectedResult.Success is not *ParticipantRemovedPayload")
				}

				if successPayload.RoundID != expectedSuccessPayload.RoundID {
					t.Errorf("expected RoundID: %v, got: %v", expectedSuccessPayload.RoundID, successPayload.RoundID)
				}
				if successPayload.UserID != expectedSuccessPayload.UserID {
					t.Errorf("expected UserID: %v, got: %v", expectedSuccessPayload.UserID, successPayload.UserID)
				}
				if successPayload.EventMessageID != expectedSuccessPayload.EventMessageID {
					t.Errorf("expected EventMessageID: %q, got: %q", expectedSuccessPayload.EventMessageID, successPayload.EventMessageID)
				}
				if len(successPayload.AcceptedParticipants) != len(expectedSuccessPayload.AcceptedParticipants) {
					t.Errorf("expected %d accepted participants, got %d", len(expectedSuccessPayload.AcceptedParticipants), len(successPayload.AcceptedParticipants))
				}
				if len(successPayload.DeclinedParticipants) != len(expectedSuccessPayload.DeclinedParticipants) {
					t.Errorf("expected %d declined participants, got %d", len(expectedSuccessPayload.DeclinedParticipants), len(successPayload.DeclinedParticipants))
				}
				if len(successPayload.TentativeParticipants) != len(expectedSuccessPayload.TentativeParticipants) {
					t.Errorf("expected %d tentative participants, got %d", len(expectedSuccessPayload.TentativeParticipants), len(successPayload.TentativeParticipants))
				}
			} else if tt.expectedResult.Failure != nil {
				if result.Failure == nil {
					t.Errorf("expected failure result, got nil failure payload")
					return
				}
				failurePayload, ok := result.Failure.(*roundevents.ParticipantRemovalErrorPayload)
				if !ok {
					t.Errorf("expected failure payload of type *ParticipantRemovalErrorPayload, got %T", result.Failure)
					return
				}
				expectedFailurePayload, ok := tt.expectedResult.Failure.(*roundevents.ParticipantRemovalErrorPayload)
				if !ok {
					t.Fatalf("test setup error: expectedResult.Failure is not *ParticipantRemovalErrorPayload")
				}

				if failurePayload.RoundID != expectedFailurePayload.RoundID {
					t.Errorf("expected RoundID: %v, got: %v", expectedFailurePayload.RoundID, failurePayload.RoundID)
				}
				if failurePayload.UserID != expectedFailurePayload.UserID {
					t.Errorf("expected UserID: %v, got: %v", expectedFailurePayload.UserID, failurePayload.UserID)
				}
				if failurePayload.Error != expectedFailurePayload.Error {
					t.Errorf("expected Error: %q, got: %q", expectedFailurePayload.Error, failurePayload.Error)
				}
			}
		})
	}
}

func TestRoundService_UpdateParticipantStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}

	testRoundID := sharedtypes.RoundID(uuid.New())

	tests := []struct {
		name           string
		mockDBSetup    func(*rounddb.MockRoundDB)
		payload        roundevents.ParticipantJoinRequestPayload
		expectedResult RoundOperationResult
		expectedError  error
	}{
		{
			name: "success updating participant status (Decline)",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(gomock.Any(), testRoundID).Return(&roundtypes.Round{
					ID:             testRoundID,
					EventMessageID: testEventMessageID,
					Participants: []roundtypes.Participant{
						{UserID: testUserID, Response: roundtypes.ResponseAccept},
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept},
					},
				}, nil).Times(1)
				mockDB.EXPECT().UpdateParticipant(gomock.Any(), testRoundID, gomock.Any()).Return([]roundtypes.Participant{
					{UserID: testUserID, Response: roundtypes.ResponseDecline},
					{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept},
				}, nil).Times(1)
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:  testRoundID,
				UserID:   testUserID,
				Response: roundtypes.ResponseDecline,
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.ParticipantJoinedPayload{
					RoundID:        testRoundID,
					EventMessageID: testEventMessageID,
					AcceptedParticipants: []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept},
					},
					DeclinedParticipants: []roundtypes.Participant{
						{UserID: testUserID, Response: roundtypes.ResponseDecline},
					},
					TentativeParticipants: []roundtypes.Participant{},
					JoinedLate:            &joinedLateFalse,
				},
			},
			expectedError: nil,
		},
		{
			name: "success updating participant status with tag (Accept)",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(gomock.Any(), testRoundID).Return(&roundtypes.Round{
					ID:             testRoundID,
					EventMessageID: testEventMessageID,
					Participants: []roundtypes.Participant{
						{UserID: testUserID, Response: roundtypes.ResponseTentative},
					},
				}, nil).Times(1)
				mockDB.EXPECT().UpdateParticipant(gomock.Any(), testRoundID, gomock.Any()).Return([]roundtypes.Participant{
					{UserID: testUserID, Response: roundtypes.ResponseAccept, TagNumber: &testTagNumber},
				}, nil).Times(1)
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:   testRoundID,
				UserID:    testUserID,
				Response:  roundtypes.ResponseAccept,
				TagNumber: &testTagNumber,
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.ParticipantJoinedPayload{
					RoundID:        testRoundID,
					EventMessageID: testEventMessageID,
					AcceptedParticipants: []roundtypes.Participant{
						{UserID: testUserID, Response: roundtypes.ResponseAccept, TagNumber: &testTagNumber},
					},
					DeclinedParticipants:  []roundtypes.Participant{},
					TentativeParticipants: []roundtypes.Participant{},
					JoinedLate:            &joinedLateFalse,
				},
			},
			expectedError: nil,
		},
		{
			name: "success triggering tag lookup (Accept without tag)",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(gomock.Any(), testRoundID).Return(&roundtypes.Round{
					ID:             testRoundID,
					EventMessageID: testEventMessageID,
					State:          roundtypes.RoundStateUpcoming,
				}, nil).Times(1)
				mockDB.EXPECT().UpdateParticipant(gomock.Any(), testRoundID, gomock.Any()).Return([]roundtypes.Participant{
					{UserID: testUserID, Response: roundtypes.ResponseAccept, TagNumber: nil},
				}, nil).Times(1)
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:    testRoundID,
				UserID:     testUserID,
				Response:   roundtypes.ResponseAccept,
				JoinedLate: &joinedLateFalse,
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.ParticipantJoinedPayload{
					RoundID:        testRoundID,
					EventMessageID: testEventMessageID,
					AcceptedParticipants: []roundtypes.Participant{
						{UserID: testUserID, Response: roundtypes.ResponseAccept, TagNumber: nil},
					},
					DeclinedParticipants:  []roundtypes.Participant{},
					TentativeParticipants: []roundtypes.Participant{},
					JoinedLate:            &joinedLateFalse,
				},
			},
			expectedError: nil,
		},
		{
			name: "failure fetching round details for decline update",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(nil, errors.New("db error")).Times(1)
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:  testRoundID,
				UserID:   testUserID,
				Response: roundtypes.ResponseDecline,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundParticipantJoinErrorPayload{
					ParticipantJoinRequest: &roundevents.ParticipantJoinRequestPayload{
						RoundID:  testRoundID,
						UserID:   testUserID,
						Response: roundtypes.ResponseDecline,
					},
					Error:          "failed to fetch round details: db error",
					EventMessageID: "",
				},
			},
			expectedError: nil,
		},
		{
			name: "failure updating participant status in DB (Decline)",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(&roundtypes.Round{
					ID:             testRoundID,
					EventMessageID: testEventMessageID,
					Participants: []roundtypes.Participant{
						{UserID: testUserID, Response: roundtypes.ResponseAccept},
					},
				}, nil).Times(1)
				mockDB.EXPECT().UpdateParticipant(ctx, testRoundID, gomock.Any()).Return(nil, errors.New("db update error")).Times(1)
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:  testRoundID,
				UserID:   testUserID,
				Response: roundtypes.ResponseDecline,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundParticipantJoinErrorPayload{
					ParticipantJoinRequest: &roundevents.ParticipantJoinRequestPayload{
						RoundID:  testRoundID,
						UserID:   testUserID,
						Response: roundtypes.ResponseDecline,
					},
					Error:          "failed to update participant in DB: db update error",
					EventMessageID: testEventMessageID,
				},
			},
			expectedError: nil,
		},
		{
			name: "failure fetching round details for tag update",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(nil, errors.New("db error")).Times(1)
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:   testRoundID,
				UserID:    testUserID,
				Response:  roundtypes.ResponseAccept,
				TagNumber: &testTagNumber,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundParticipantJoinErrorPayload{
					ParticipantJoinRequest: &roundevents.ParticipantJoinRequestPayload{
						RoundID:   testRoundID,
						UserID:    testUserID,
						Response:  roundtypes.ResponseAccept,
						TagNumber: &testTagNumber,
					},
					Error:          "failed to fetch round details: db error",
					EventMessageID: "",
				},
			},
			expectedError: nil,
		},
		{
			name: "failure updating participant with tag in DB",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(&roundtypes.Round{
					ID:             testRoundID,
					EventMessageID: testEventMessageID,
					Participants: []roundtypes.Participant{
						{UserID: testUserID, Response: roundtypes.ResponseTentative},
					},
				}, nil).Times(1)
				mockDB.EXPECT().UpdateParticipant(ctx, testRoundID, gomock.Any()).Return(nil, errors.New("db update error")).Times(1)
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:   testRoundID,
				UserID:    testUserID,
				Response:  roundtypes.ResponseAccept,
				TagNumber: &testTagNumber,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundParticipantJoinErrorPayload{
					ParticipantJoinRequest: &roundevents.ParticipantJoinRequestPayload{
						RoundID:   testRoundID,
						UserID:    testUserID,
						Response:  roundtypes.ResponseAccept,
						TagNumber: &testTagNumber,
					},
					Error:          "failed to update participant in DB: db update error",
					EventMessageID: testEventMessageID,
				},
			},
			expectedError: nil,
		},
		{
			name: "unknown response type",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// No DB calls expected for this path
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:  testRoundID,
				UserID:   testUserID,
				Response: "UNKNOWN_RESPONSE_TYPE",
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundParticipantJoinErrorPayload{
					ParticipantJoinRequest: &roundevents.ParticipantJoinRequestPayload{
						RoundID:  testRoundID,
						UserID:   testUserID,
						Response: "UNKNOWN_RESPONSE_TYPE",
					},
					Error:          "unknown response type: UNKNOWN_RESPONSE_TYPE",
					EventMessageID: "",
				},
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockDB := rounddb.NewMockRoundDB(ctrl)
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

			result, err := s.UpdateParticipantStatus(ctx, tt.payload)

			if (err != nil) != (tt.expectedError != nil) {
				t.Fatalf("expected error presence %v, got %v (error: %v)", tt.expectedError != nil, err != nil, err)
			}
			if err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
			}

			// Validate result based on success or failure expectation
			if tt.expectedResult.Success != nil {
				if result.Success == nil {
					t.Errorf("expected success result, got nil success payload")
					return
				}

				switch expectedSuccess := tt.expectedResult.Success.(type) {
				case *roundevents.ParticipantJoinedPayload:
					successPayload, ok := result.Success.(*roundevents.ParticipantJoinedPayload)
					if !ok {
						t.Errorf("expected success payload of type *ParticipantJoinedPayload, got %T", result.Success)
						return
					}
					if successPayload.RoundID != expectedSuccess.RoundID {
						t.Errorf("expected RoundID: %v, got: %v", expectedSuccess.RoundID, successPayload.RoundID)
					}
					if successPayload.EventMessageID != expectedSuccess.EventMessageID {
						t.Errorf("expected EventMessageID: %q, got: %q", expectedSuccess.EventMessageID, successPayload.EventMessageID)
					}
					if len(successPayload.AcceptedParticipants) != len(expectedSuccess.AcceptedParticipants) {
						t.Errorf("expected %d accepted participants, got %d", len(expectedSuccess.AcceptedParticipants), len(successPayload.AcceptedParticipants))
					}
					if len(successPayload.DeclinedParticipants) != len(expectedSuccess.DeclinedParticipants) {
						t.Errorf("expected %d declined participants, got %d", len(expectedSuccess.DeclinedParticipants), len(successPayload.DeclinedParticipants))
					}
					if len(successPayload.TentativeParticipants) != len(expectedSuccess.TentativeParticipants) {
						t.Errorf("expected %d tentative participants, got %d", len(expectedSuccess.TentativeParticipants), len(successPayload.TentativeParticipants))
					}
					if (successPayload.JoinedLate == nil && expectedSuccess.JoinedLate != nil) ||
						(successPayload.JoinedLate != nil && expectedSuccess.JoinedLate == nil) ||
						(successPayload.JoinedLate != nil && expectedSuccess.JoinedLate != nil &&
							*successPayload.JoinedLate != *expectedSuccess.JoinedLate) {
						t.Errorf("expected JoinedLate: %v, got: %v",
							formatBoolPtr(expectedSuccess.JoinedLate),
							formatBoolPtr(successPayload.JoinedLate))
					}

				case *roundevents.TagLookupRequestPayload:
					successPayload, ok := result.Success.(*roundevents.TagLookupRequestPayload)
					if !ok {
						t.Errorf("expected success payload of type *TagLookupRequestPayload, got %T", result.Success)
						return
					}
					if successPayload.RoundID != expectedSuccess.RoundID {
						t.Errorf("expected RoundID: %v, got: %v", expectedSuccess.RoundID, successPayload.RoundID)
					}
					if successPayload.UserID != expectedSuccess.UserID {
						t.Errorf("expected UserID: %v, got: %v", expectedSuccess.UserID, successPayload.UserID)
					}
					if successPayload.Response != expectedSuccess.Response {
						t.Errorf("expected Response: %v, got: %v", expectedSuccess.Response, successPayload.Response)
					}
					if (successPayload.JoinedLate == nil && expectedSuccess.JoinedLate != nil) ||
						(successPayload.JoinedLate != nil && expectedSuccess.JoinedLate == nil) ||
						(successPayload.JoinedLate != nil && expectedSuccess.JoinedLate != nil &&
							*successPayload.JoinedLate != *expectedSuccess.JoinedLate) {
						t.Errorf("expected JoinedLate: %v, got: %v",
							formatBoolPtr(expectedSuccess.JoinedLate),
							formatBoolPtr(successPayload.JoinedLate))
					}

				default:
					t.Errorf("unexpected success payload type: %T", result.Success)
				}

			} else if tt.expectedResult.Failure != nil {
				if result.Failure == nil {
					t.Errorf("expected failure result, got nil failure payload")
					return
				}

				expectedFailure, ok := tt.expectedResult.Failure.(*roundevents.RoundParticipantJoinErrorPayload)
				if !ok {
					t.Errorf("expected failure payload of type *RoundParticipantJoinErrorPayload, got %T", tt.expectedResult.Failure)
					return
				}

				failurePayload, ok := result.Failure.(*roundevents.RoundParticipantJoinErrorPayload)
				if !ok {
					t.Errorf("expected failure payload of type *RoundParticipantJoinErrorPayload, got %T", result.Failure)
					return
				}

				if failurePayload.ParticipantJoinRequest.RoundID != expectedFailure.ParticipantJoinRequest.RoundID {
					t.Errorf("expected RoundID: %v, got: %v", expectedFailure.ParticipantJoinRequest.RoundID, failurePayload.ParticipantJoinRequest.RoundID)
				}
				if failurePayload.Error != expectedFailure.Error {
					t.Errorf("expected Error: %q, got: %q", expectedFailure.Error, failurePayload.Error)
				}
				if failurePayload.EventMessageID != expectedFailure.EventMessageID {
					t.Errorf("expected EventMessageID: %q, got: %q", expectedFailure.EventMessageID, failurePayload.EventMessageID)
				}
			}
		})
	}
}

// Helper function for formatting bool pointers for clearer error messages
func formatBoolPtr(b *bool) string {
	if b == nil {
		return "nil"
	}
	return fmt.Sprintf("%t", *b)
}
