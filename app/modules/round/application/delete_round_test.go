package roundservice

import (
	"context"
	"errors"
	"fmt" // Import fmt for error string matching
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	queuemocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/queue/mocks"
	rounddbmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
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
		mockDBSetup             func(*rounddbmocks.MockRepository)
		mockRoundValidatorSetup func(*roundutil.MockRoundValidator)
		payload                 roundevents.RoundDeleteRequestPayloadV1
		expectedResult          results.OperationResult
		expectedError           error
	}{
		{
			name: "valid round delete request",
			mockDBSetup: func(mockDB *rounddbmocks.MockRepository) {
				guildID := sharedtypes.GuildID("guild-123")
				roundID := sharedtypes.RoundID(uuid.MustParse("b236f541-e988-41b4-ab81-0e906f2ac270"))
				mockDB.EXPECT().GetRound(gomock.Any(), guildID, roundID).Return(&roundtypes.Round{
					ID:        roundID,
					GuildID:   guildID,
					CreatedBy: "Test User",
				}, nil)
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
				// No specific validator mocks needed for validation
			},
			payload: roundevents.RoundDeleteRequestPayloadV1{
				GuildID:              sharedtypes.GuildID("guild-123"),
				RoundID:              sharedtypes.RoundID(uuid.MustParse("b236f541-e988-41b4-ab81-0e906f2ac270")),
				RequestingUserUserID: "Test User",
			},
			expectedResult: results.OperationResult{
				Success: &roundevents.RoundDeleteValidatedPayloadV1{
					RoundDeleteRequestPayload: roundevents.RoundDeleteRequestPayloadV1{
						GuildID:              sharedtypes.GuildID("guild-123"),
						RoundID:              sharedtypes.RoundID(uuid.MustParse("b236f541-e988-41b4-ab81-0e906f2ac270")),
						RequestingUserUserID: "Test User",
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "invalid round delete request - empty round ID",
			mockDBSetup: func(mockDB *rounddbmocks.MockRepository) {
				// No DB calls expected for empty round ID
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
			},
			payload: roundevents.RoundDeleteRequestPayloadV1{
				GuildID:              sharedtypes.GuildID("guild-123"),
				RoundID:              sharedtypes.RoundID(uuid.Nil), // Use uuid.Nil for empty UUID
				RequestingUserUserID: "Test User",
			},
			expectedResult: results.OperationResult{
				Failure: &roundevents.RoundDeleteErrorPayloadV1{
					RoundDeleteRequest: &roundevents.RoundDeleteRequestPayloadV1{
						GuildID:              sharedtypes.GuildID("guild-123"),
						RoundID:              sharedtypes.RoundID(uuid.Nil),
						RequestingUserUserID: "Test User",
					},
					Error: "round ID cannot be zero",
				},
			},
			expectedError: nil,
		},
		{
			name: "invalid round delete request - empty requesting user ID",
			mockDBSetup: func(mockDB *rounddbmocks.MockRepository) {
				// No DB calls expected for empty user ID
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
			},
			payload: func() roundevents.RoundDeleteRequestPayloadV1 {
				guildID := sharedtypes.GuildID("guild-123")
				roundID := sharedtypes.RoundID(uuid.New())
				return roundevents.RoundDeleteRequestPayloadV1{
					GuildID:              guildID,
					RoundID:              roundID,
					RequestingUserUserID: "",
				}
			}(),
			expectedResult: func() results.OperationResult {
				guildID := sharedtypes.GuildID("guild-123")
				roundID := sharedtypes.RoundID(uuid.New()) // This will be overwritten in the test
				return results.OperationResult{
					Failure: &roundevents.RoundDeleteErrorPayloadV1{
						RoundDeleteRequest: &roundevents.RoundDeleteRequestPayloadV1{
							GuildID:              guildID,
							RoundID:              roundID,
							RequestingUserUserID: "",
						},
						Error: "requesting user's Discord ID cannot be empty",
					},
				}
			}(),
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks for each subtest
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockDB := rounddbmocks.NewMockRepository(ctrl)

			tt.mockDBSetup(mockDB)
			tt.mockRoundValidatorSetup(mockRoundValidator)

			s := &RoundService{
				repo:           mockDB,
				logger:         logger,
				metrics:        mockMetrics,
				tracer:         tracer,
				roundValidator: mockRoundValidator,
			}

			result, err := s.ValidateRoundDeleteRequest(ctx, tt.payload)

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
				} else if successPayload, ok := result.Success.(*roundevents.RoundDeleteValidatedPayloadV1); !ok {
					t.Errorf("expected result.Success to be of type *roundevents.RoundDeleteValidatedPayloadV1, got %T", result.Success)
				} else {
					// Validate the payload content
					if successPayload.RoundDeleteRequestPayload.RequestingUserUserID != tt.payload.RequestingUserUserID {
						t.Errorf("expected RequestingUserUserID %s, got %s", tt.payload.RequestingUserUserID, successPayload.RoundDeleteRequestPayload.RequestingUserUserID)
					}
				}
			} else if tt.expectedResult.Failure != nil {
				// Type assertion for Failure payload
				if result.Failure == nil {
					t.Errorf("expected failure result, got success")
				} else if failurePayload, ok := result.Failure.(*roundevents.RoundDeleteErrorPayloadV1); !ok {
					t.Errorf("expected result.Failure to be of type *roundevents.RoundDeleteErrorPayloadV1, got %T", result.Failure)
				} else if expectedFailurePayload, ok := tt.expectedResult.Failure.(*roundevents.RoundDeleteErrorPayloadV1); !ok {
					t.Errorf("expected tt.expectedResult.Failure to be of type *roundevents.RoundDeleteErrorPayloadV1, got %T", tt.expectedResult.Failure)
				} else if failurePayload.Error != expectedFailurePayload.Error {
					t.Errorf("expected failure error %q, got %q", expectedFailurePayload.Error, failurePayload.Error)
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
		name           string
		mockDBSetup    func(*rounddbmocks.MockRepository)
		mockQueueSetup func(*queuemocks.MockQueueService)
		payload        roundevents.RoundDeleteAuthorizedPayloadV1
		expectedResult results.OperationResult
		expectedError  error
	}{
		{
			name: "delete round successfully",
			mockDBSetup: func(mockDB *rounddbmocks.MockRepository) {
				guildID := sharedtypes.GuildID("guild-123")
				// Mock GetRound call to verify round exists
				mockDB.EXPECT().GetRound(gomock.Any(), guildID, gomock.Eq(id)).Return(&roundtypes.Round{
					ID:             id,
					GuildID:        guildID,
					EventMessageID: "test-event-message-id",
				}, nil)
				mockDB.EXPECT().DeleteRound(gomock.Any(), guildID, gomock.Eq(id)).Return(nil)
			},
			mockQueueSetup: func(mockQueue *queuemocks.MockQueueService) {
				mockQueue.EXPECT().CancelRoundJobs(gomock.Any(), gomock.Eq(id)).Return(nil)
			},
			payload: roundevents.RoundDeleteAuthorizedPayloadV1{
				GuildID: sharedtypes.GuildID("guild-123"),
				RoundID: id,
			},
			expectedResult: results.OperationResult{
				Success: &roundevents.RoundDeletedPayloadV1{
					GuildID:        sharedtypes.GuildID("guild-123"),
					RoundID:        id,
					EventMessageID: "test-event-message-id",
				},
			},
			expectedError: nil,
		},
		{
			name: "delete round fails - delete round error",
			mockDBSetup: func(mockDB *rounddbmocks.MockRepository) {
				guildID := sharedtypes.GuildID("guild-123")
				// Mock GetRound call to verify round exists
				mockDB.EXPECT().GetRound(gomock.Any(), guildID, gomock.Eq(id)).Return(&roundtypes.Round{ID: id, GuildID: guildID}, nil)
				mockDB.EXPECT().DeleteRound(gomock.Any(), guildID, gomock.Eq(id)).Return(errors.New("delete round error"))
			},
			mockQueueSetup: func(mockQueue *queuemocks.MockQueueService) {
				// No queue calls expected when DB delete fails
			},
			payload: roundevents.RoundDeleteAuthorizedPayloadV1{
				GuildID: sharedtypes.GuildID("guild-123"),
				RoundID: id,
			},
			expectedResult: results.OperationResult{
				Failure: &roundevents.RoundDeleteErrorPayloadV1{
					RoundDeleteRequest: nil,
					Error:              fmt.Sprintf("failed to delete round from database: %v", errors.New("delete round error")),
				},
			},
			expectedError: nil, // DeleteRound returns failure payload with nil error
		},
		{
			name: "delete round succeeds despite cancel queue jobs error",
			mockDBSetup: func(mockDB *rounddbmocks.MockRepository) {
				guildID := sharedtypes.GuildID("guild-123")
				// Mock GetRound call to verify round exists
				mockDB.EXPECT().GetRound(gomock.Any(), guildID, gomock.Eq(id)).Return(&roundtypes.Round{
					ID:             id,
					GuildID:        guildID,
					EventMessageID: "test-event-message-id",
				}, nil)
				mockDB.EXPECT().DeleteRound(gomock.Any(), guildID, gomock.Eq(id)).Return(nil)
			},
			mockQueueSetup: func(mockQueue *queuemocks.MockQueueService) {
				mockQueue.EXPECT().CancelRoundJobs(gomock.Any(), gomock.Eq(id)).Return(errors.New("cancel queue jobs error"))
			},
			payload: roundevents.RoundDeleteAuthorizedPayloadV1{
				GuildID: sharedtypes.GuildID("guild-123"),
				RoundID: id,
			},
			expectedResult: results.OperationResult{
				Success: &roundevents.RoundDeletedPayloadV1{
					GuildID:        sharedtypes.GuildID("guild-123"),
					RoundID:        id,
					EventMessageID: "test-event-message-id",
				},
			},
			expectedError: nil, // The service logs a warning but returns nil error for the overall operation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks for each subtest
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockDB := rounddbmocks.NewMockRepository(ctrl)
			mockQueue := queuemocks.NewMockQueueService(ctrl)

			tt.mockDBSetup(mockDB)
			tt.mockQueueSetup(mockQueue)

			s := &RoundService{
				repo:           mockDB,
				queueService:   mockQueue,
				logger:         logger,
				metrics:        mockMetrics,
				tracer:         tracer,
				roundValidator: mockRoundValidator,
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
				} else if successPayload, ok := result.Success.(*roundevents.RoundDeletedPayloadV1); !ok {
					t.Errorf("expected result.Success to be of type *roundevents.RoundDeletedPayloadV1, got %T", result.Success)
				} else if expectedSuccessPayload, ok := tt.expectedResult.Success.(*roundevents.RoundDeletedPayloadV1); !ok {
					t.Errorf("expected tt.expectedResult.Success to be of type *roundevents.RoundDeletedPayloadV1, got %T", tt.expectedResult.Success)
				} else if successPayload.RoundID != expectedSuccessPayload.RoundID {
					t.Errorf("expected success RoundID %s, got %s", expectedSuccessPayload.RoundID, successPayload.RoundID)
				}
			} else if tt.expectedResult.Failure != nil {
				// Type assertion for Failure payload
				if result.Failure == nil {
					t.Errorf("expected failure result, got success")
				} else if failurePayload, ok := result.Failure.(*roundevents.RoundDeleteErrorPayloadV1); !ok {
					t.Errorf("expected result.Failure to be of type *roundevents.RoundDeleteErrorPayloadV1, got %T", result.Failure)
				} else if expectedFailurePayload, ok := tt.expectedResult.Failure.(*roundevents.RoundDeleteErrorPayloadV1); !ok {
					t.Errorf("expected tt.expectedResult.Failure to be of type *roundevents.RoundDeleteErrorPayloadV1, got %T", tt.expectedResult.Failure)
				} else if failurePayload.Error != expectedFailurePayload.Error {
					t.Errorf("expected failure error %q, got %q", expectedFailurePayload.Error, failurePayload.Error)
				}
			}
		})
	}
}
