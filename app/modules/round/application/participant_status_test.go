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
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name        string
		mockDBSetup func(*rounddb.MockRoundDB)
		payload     roundevents.ParticipantRemovalRequestPayload
		// expectedResult and expectedError are part of the test struct,
		// but actual validation will be done more robustly below.
		expectedResult RoundOperationResult
		expectedError  error
	}{
		{
			name: "success removing participant",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// Mock GetRound before removal (participant exists)
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(&roundtypes.Round{
					ID:             testRoundID,
					EventMessageID: testEventMessageID,
					Participants: []roundtypes.Participant{
						{UserID: testUserID, Response: roundtypes.ResponseAccept},
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept},
					},
				}, nil).Times(1)
				// Mock RemoveParticipant
				mockDB.EXPECT().RemoveParticipant(ctx, testRoundID, testUserID).Return(nil).Times(1)
				// Mock GetRound after removal (participant removed)
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(&roundtypes.Round{
					ID:             testRoundID,
					EventMessageID: testEventMessageID,
					Participants: []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept},
					},
				}, nil).Times(1)
			},
			payload: roundevents.ParticipantRemovalRequestPayload{
				RoundID: testRoundID,
				UserID:  testUserID,
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.ParticipantRemovedPayload{
					RoundID:        testRoundID,
					UserID:         testUserID,
					EventMessageID: testEventMessageID, // Corrected: use testEventMessageID
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
				// Mock GetRound before removal (participant does not exist in round)
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(&roundtypes.Round{
					ID:             testRoundID,
					EventMessageID: testEventMessageID,
					Participants: []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept},
					},
				}, nil).Times(1)
				// No calls to RemoveParticipant or second GetRound expected as the participant isn't found
			},
			payload: roundevents.ParticipantRemovalRequestPayload{
				RoundID: testRoundID,
				UserID:  testUserID, // This user is not in the mocked participants list
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.ParticipantRemovedPayload{
					RoundID:               testRoundID,
					UserID:                testUserID,
					EventMessageID:        testEventMessageID,
					AcceptedParticipants:  []roundtypes.Participant{},
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
				Failure: roundevents.ParticipantRemovalErrorPayload{
					RoundID: testRoundID,
					UserID:  testUserID,
					Error:   "failed to fetch round details before removal: db error", // Corrected error message
				},
			},
			expectedError: errors.New("failed to fetch round details before removal: db error"), // Corrected error message
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
				mockDB.EXPECT().RemoveParticipant(ctx, testRoundID, testUserID).Return(errors.New("db remove error")).Times(1)
			},
			payload: roundevents.ParticipantRemovalRequestPayload{
				RoundID: testRoundID,
				UserID:  testUserID,
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.ParticipantRemovalErrorPayload{
					RoundID: testRoundID,
					UserID:  testUserID,
					Error:   "failed to remove participant from database: db remove error", // Corrected error message
				},
			},
			expectedError: errors.New("failed to remove participant from database: db remove error"), // Corrected error message
		},
		{
			name: "failure fetching round details after removal",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(&roundtypes.Round{
					ID:             testRoundID,
					EventMessageID: testEventMessageID,
					Participants: []roundtypes.Participant{
						{UserID: testUserID, Response: roundtypes.ResponseAccept},
					},
				}, nil).Times(1)
				mockDB.EXPECT().RemoveParticipant(ctx, testRoundID, testUserID).Return(nil).Times(1)
				mockDB.EXPECT().GetRound(ctx, testRoundID).Return(nil, errors.New("db fetch after remove error")).Times(1)
			},
			payload: roundevents.ParticipantRemovalRequestPayload{
				RoundID: testRoundID,
				UserID:  testUserID,
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.ParticipantRemovalErrorPayload{
					RoundID: testRoundID,
					UserID:  testUserID,
					Error:   "failed to fetch updated round after removal for discord update: db fetch after remove error", // Corrected error message
				},
			},
			expectedError: errors.New("failed to fetch updated round after removal for discord update: db fetch after remove error"), // Corrected error message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create new mocks for each test run to ensure isolation
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockDB := rounddb.NewMockRoundDB(ctrl)
			mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)
			mockEventBus := eventbus.NewMockEventBus(ctrl)

			tt.mockDBSetup(mockDB)
			// No specific mockRoundValidatorSetup or mockEventBusSetup needed for this test currently
			// tt.mockRoundValidatorSetup(mockRoundValidator)
			// tt.mockEventBusSetup(mockEventBus)

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

			// Validate error presence and message
			if (err != nil) != (tt.expectedError != nil) {
				t.Fatalf("expected error presence %v, got %v (error: %v)", tt.expectedError != nil, err != nil, err)
			}
			if err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
			}

			// Validate result based on success or failure expectation
			if tt.expectedError == nil { // Success case
				if result.Success == nil {
					t.Errorf("expected success result, got nil success payload")
					return
				}
				successPayload, ok := result.Success.(roundevents.ParticipantRemovedPayload)
				if !ok {
					t.Errorf("expected success payload of type ParticipantRemovedPayload, got %T", result.Success)
					return
				}
				expectedSuccessPayload, ok := tt.expectedResult.Success.(roundevents.ParticipantRemovedPayload)
				if !ok {
					t.Fatalf("test setup error: expectedResult.Success is not ParticipantRemovedPayload")
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
				// Compare participant lists (deep comparison might be needed for complex structs)
				if len(successPayload.AcceptedParticipants) != len(expectedSuccessPayload.AcceptedParticipants) {
					t.Errorf("expected %d accepted participants, got %d", len(expectedSuccessPayload.AcceptedParticipants), len(successPayload.AcceptedParticipants))
				}
				if len(successPayload.DeclinedParticipants) != len(expectedSuccessPayload.DeclinedParticipants) {
					t.Errorf("expected %d declined participants, got %d", len(expectedSuccessPayload.DeclinedParticipants), len(successPayload.DeclinedParticipants))
				}
				if len(successPayload.TentativeParticipants) != len(expectedSuccessPayload.TentativeParticipants) {
					t.Errorf("expected %d tentative participants, got %d", len(expectedSuccessPayload.TentativeParticipants), len(successPayload.TentativeParticipants))
				}
				// Add more robust slice comparison if element order/content matters.
			} else { // Failure case
				if result.Failure == nil {
					t.Errorf("expected failure result, got nil failure payload")
					return
				}
				failurePayload, ok := result.Failure.(roundevents.ParticipantRemovalErrorPayload)
				if !ok {
					t.Errorf("expected failure payload of type ParticipantRemovalErrorPayload, got %T", result.Failure)
					return
				}
				expectedFailurePayload, ok := tt.expectedResult.Failure.(roundevents.ParticipantRemovalErrorPayload)
				if !ok {
					t.Fatalf("test setup error: expectedResult.Failure is not ParticipantRemovalErrorPayload")
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

	tests := []struct {
		name        string
		mockDBSetup func(*rounddb.MockRoundDB)
		payload     roundevents.ParticipantJoinRequestPayload
		// expectedResult and expectedError are part of the test struct,
		// but actual validation will be done more robustly below.
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
				// Use gomock.Any() for the participant struct as its internal fields might vary
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
				Success: roundevents.ParticipantJoinedPayload{
					RoundID:        testRoundID,
					EventMessageID: testEventMessageID,
					AcceptedParticipants: []roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user2"), Response: roundtypes.ResponseAccept},
					},
					DeclinedParticipants: []roundtypes.Participant{
						{UserID: testUserID, Response: roundtypes.ResponseDecline},
					},
					TentativeParticipants: []roundtypes.Participant{},
					JoinedLate:            &joinedLateFalse, // Ensure this matches the payload's JoinedLate
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
				Success: roundevents.ParticipantJoinedPayload{
					RoundID:        testRoundID,
					EventMessageID: testEventMessageID,
					AcceptedParticipants: []roundtypes.Participant{
						{UserID: testUserID, Response: roundtypes.ResponseAccept, TagNumber: &testTagNumber},
					},
					DeclinedParticipants:  []roundtypes.Participant{},
					TentativeParticipants: []roundtypes.Participant{},
					JoinedLate:            &joinedLateFalse, // Changed from nil to &joinedLateFalse
				},
			},
			expectedError: nil,
		},
		{
			name: "success triggering tag lookup (Accept without tag)",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// No DB calls expected for this path as it returns TagLookupRequestPayload
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:  testRoundID,
				UserID:   testUserID,
				Response: roundtypes.ResponseAccept,
				// TagNumber is nil
			},
			expectedResult: RoundOperationResult{
				Success: roundevents.TagLookupRequestPayload{
					RoundID:    testRoundID,
					UserID:     testUserID,
					Response:   roundtypes.ResponseAccept,
					JoinedLate: nil, // Assuming nil if not explicitly set in payload
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
				Failure: roundevents.ParticipantUpdateErrorPayload{
					RoundID: testRoundID,
					UserID:  testUserID,
					Error:   "failed to fetch round details for decline update: db error", // Corrected error message
				},
			},
			expectedError: errors.New("failed to fetch round details for decline update: db error"), // Corrected error message
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
				Failure: roundevents.RoundParticipantJoinErrorPayload{
					ParticipantJoinRequest: &roundevents.ParticipantJoinRequestPayload{
						RoundID:  testRoundID,
						UserID:   testUserID,
						Response: roundtypes.ResponseDecline,
					},
					Error:          "failed to update participant status in DB: db update error", // Corrected error message
					EventMessageID: testEventMessageID,
				},
			},
			expectedError: errors.New("failed to update participant status: db update error"), // Corrected error message
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
				Failure: roundevents.ParticipantUpdateErrorPayload{
					RoundID: testRoundID,
					UserID:  testUserID,
					Error:   "failed to fetch round details for tag update: db error", // Corrected error message
				},
			},
			expectedError: errors.New("failed to fetch round details for tag update: db error"), // Corrected error message
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
				Failure: roundevents.RoundParticipantJoinErrorPayload{
					ParticipantJoinRequest: &roundevents.ParticipantJoinRequestPayload{
						RoundID:   testRoundID,
						UserID:    testUserID,
						Response:  roundtypes.ResponseAccept,
						TagNumber: &testTagNumber,
					},
					Error:          "failed to update participant with tag in DB: db update error", // Corrected error message
					EventMessageID: testEventMessageID,
				},
			},
			expectedError: errors.New("failed to update participant with tag: db update error"), // Corrected error message
		},
		{
			name: "unknown response type",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				// No DB calls expected for this path
			},
			payload: roundevents.ParticipantJoinRequestPayload{
				RoundID:  testRoundID,
				UserID:   testUserID,
				Response: "UNKNOWN_RESPONSE_TYPE", // Simulate an unknown response
			},
			expectedResult: RoundOperationResult{
				Failure: roundevents.ParticipantUpdateErrorPayload{
					RoundID: testRoundID,
					UserID:  testUserID,
					Error:   "unknown response type: UNKNOWN_RESPONSE_TYPE",
				},
			},
			expectedError: errors.New("unknown response type: UNKNOWN_RESPONSE_TYPE"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create new mocks for each test run to ensure isolation
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockDB := rounddb.NewMockRoundDB(ctrl)
			mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)
			mockEventBus := eventbus.NewMockEventBus(ctrl)

			tt.mockDBSetup(mockDB)
			// No specific mockRoundValidatorSetup or mockEventBusSetup needed for this test currently
			// tt.mockRoundValidatorSetup(mockRoundValidator)
			// tt.mockEventBusSetup(mockEventBus)

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

			// Validate error presence and message
			if (err != nil) != (tt.expectedError != nil) {
				t.Fatalf("expected error presence %v, got %v (error: %v)", tt.expectedError != nil, err != nil, err)
			}
			if err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
			}

			// Validate result based on success or failure expectation
			if tt.expectedError == nil { // Success case
				if result.Success == nil {
					t.Errorf("expected success result, got nil success payload")
					return
				}

				switch expectedSuccess := tt.expectedResult.Success.(type) {
				case roundevents.ParticipantJoinedPayload:
					successPayload, ok := result.Success.(roundevents.ParticipantJoinedPayload)
					if !ok {
						t.Errorf("expected success payload of type ParticipantJoinedPayload, got %T", result.Success)
						return
					}
					if successPayload.RoundID != expectedSuccess.RoundID {
						t.Errorf("expected RoundID: %v, got: %v", expectedSuccess.RoundID, successPayload.RoundID)
					}
					if successPayload.EventMessageID != expectedSuccess.EventMessageID {
						t.Errorf("expected EventMessageID: %q, got: %q", expectedSuccess.EventMessageID, successPayload.EventMessageID)
					}
					// Compare participant lists (deep comparison might be needed)
					if len(successPayload.AcceptedParticipants) != len(expectedSuccess.AcceptedParticipants) {
						t.Errorf("expected %d accepted participants, got %d", len(expectedSuccess.AcceptedParticipants), len(successPayload.AcceptedParticipants))
					}
					if len(successPayload.DeclinedParticipants) != len(expectedSuccess.DeclinedParticipants) {
						t.Errorf("expected %d declined participants, got %d", len(expectedSuccess.DeclinedParticipants), len(successPayload.DeclinedParticipants))
					}
					if len(successPayload.TentativeParticipants) != len(expectedSuccess.TentativeParticipants) {
						t.Errorf("expected %d tentative participants, got %d", len(expectedSuccess.TentativeParticipants), len(successPayload.TentativeParticipants))
					}
					// Validate JoinedLate
					if (successPayload.JoinedLate == nil && expectedSuccess.JoinedLate != nil) ||
						(successPayload.JoinedLate != nil && expectedSuccess.JoinedLate == nil) ||
						(successPayload.JoinedLate != nil && expectedSuccess.JoinedLate != nil &&
							*successPayload.JoinedLate != *expectedSuccess.JoinedLate) {
						t.Errorf("expected JoinedLate: %v, got: %v",
							formatBoolPtr(expectedSuccess.JoinedLate),
							formatBoolPtr(successPayload.JoinedLate))
					}

				case roundevents.TagLookupRequestPayload:
					successPayload, ok := result.Success.(roundevents.TagLookupRequestPayload)
					if !ok {
						t.Errorf("expected success payload of type TagLookupRequestPayload, got %T", result.Success)
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
					// Validate JoinedLate
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

			} else { // Failure case
				if result.Failure == nil {
					t.Errorf("expected failure result, got nil failure payload")
					return
				}

				switch expectedFailure := tt.expectedResult.Failure.(type) {
				case roundevents.ParticipantUpdateErrorPayload:
					failurePayload, ok := result.Failure.(roundevents.ParticipantUpdateErrorPayload)
					if !ok {
						t.Errorf("expected failure payload of type ParticipantUpdateErrorPayload, got %T", result.Failure)
						return
					}
					if failurePayload.RoundID != expectedFailure.RoundID {
						t.Errorf("expected RoundID: %v, got: %v", expectedFailure.RoundID, failurePayload.RoundID)
					}
					if failurePayload.UserID != expectedFailure.UserID {
						t.Errorf("expected UserID: %v, got: %v", expectedFailure.UserID, failurePayload.UserID)
					}
					if failurePayload.Error != expectedFailure.Error {
						t.Errorf("expected Error: %q, got: %q", expectedFailure.Error, failurePayload.Error)
					}
				case roundevents.RoundParticipantJoinErrorPayload:
					failurePayload, ok := result.Failure.(roundevents.RoundParticipantJoinErrorPayload)
					if !ok {
						t.Errorf("expected failure payload of type RoundParticipantJoinErrorPayload, got %T", result.Failure)
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
				default:
					t.Errorf("unexpected failure payload type: %T", result.Failure)
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
