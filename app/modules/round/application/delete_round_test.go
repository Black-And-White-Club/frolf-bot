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
		mockDBSetup             func(*rounddbmocks.MockRoundDB)
		mockRoundValidatorSetup func(*roundutil.MockRoundValidator)
		payload                 roundevents.RoundDeleteRequestPayload
		expectedResult          RoundOperationResult
		expectedError           error
	}{
		{
			name: "valid round delete request",
			mockDBSetup: func(mockDB *rounddbmocks.MockRoundDB) {
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
			payload: roundevents.RoundDeleteRequestPayload{
				GuildID:              sharedtypes.GuildID("guild-123"),
				RoundID:              sharedtypes.RoundID(uuid.MustParse("b236f541-e988-41b4-ab81-0e906f2ac270")),
				RequestingUserUserID: "Test User",
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.RoundDeleteValidatedPayload{
					RoundDeleteRequestPayload: roundevents.RoundDeleteRequestPayload{
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
			mockDBSetup: func(mockDB *rounddbmocks.MockRoundDB) {
				// No DB calls expected for empty round ID
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
			},
			payload: roundevents.RoundDeleteRequestPayload{
				GuildID:              sharedtypes.GuildID("guild-123"),
				RoundID:              sharedtypes.RoundID(uuid.Nil), // Use uuid.Nil for empty UUID
				RequestingUserUserID: "Test User",
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundDeleteErrorPayload{
					RoundDeleteRequest: &roundevents.RoundDeleteRequestPayload{
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
			mockDBSetup: func(mockDB *rounddbmocks.MockRoundDB) {
				// No DB calls expected for empty user ID
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
			},
			payload: func() roundevents.RoundDeleteRequestPayload {
				guildID := sharedtypes.GuildID("guild-123")
				roundID := sharedtypes.RoundID(uuid.New())
				return roundevents.RoundDeleteRequestPayload{
					GuildID:              guildID,
					RoundID:              roundID,
					RequestingUserUserID: "",
				}
			}(),
			expectedResult: func() RoundOperationResult {
				guildID := sharedtypes.GuildID("guild-123")
				roundID := sharedtypes.RoundID(uuid.New()) // This will be overwritten in the test
				return RoundOperationResult{
					Failure: &roundevents.RoundDeleteErrorPayload{
						RoundDeleteRequest: &roundevents.RoundDeleteRequestPayload{
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
			mockDB := rounddbmocks.NewMockRoundDB(ctrl)

			tt.mockDBSetup(mockDB)
			tt.mockRoundValidatorSetup(mockRoundValidator)

			s := &RoundService{
				RoundDB:        mockDB,
				logger:         logger,
				metrics:        mockMetrics,
				tracer:         tracer,
				roundValidator: mockRoundValidator,
				serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc(ctx)
				},
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
				} else if successPayload, ok := result.Success.(*roundevents.RoundDeleteValidatedPayload); !ok {
					t.Errorf("expected result.Success to be of type *roundevents.RoundDeleteValidatedPayload, got %T", result.Success)
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
		mockDBSetup    func(*rounddbmocks.MockRoundDB)
		mockQueueSetup func(*queuemocks.MockQueueService)
		payload        roundevents.RoundDeleteAuthorizedPayload
		expectedResult RoundOperationResult
		expectedError  error
	}{
		{
			name: "delete round successfully",
			mockDBSetup: func(mockDB *rounddbmocks.MockRoundDB) {
				guildID := sharedtypes.GuildID("guild-123")
				// Mock GetRound call to verify round exists
				mockDB.EXPECT().GetRound(ctx, guildID, gomock.Eq(id)).Return(&roundtypes.Round{
					ID:             id,
					GuildID:        guildID,
					EventMessageID: "test-event-message-id",
				}, nil)
				mockDB.EXPECT().DeleteRound(ctx, guildID, gomock.Eq(id)).Return(nil)
			},
			mockQueueSetup: func(mockQueue *queuemocks.MockQueueService) {
				mockQueue.EXPECT().CancelRoundJobs(ctx, gomock.Eq(id)).Return(nil)
			},
			payload: roundevents.RoundDeleteAuthorizedPayload{
				GuildID: sharedtypes.GuildID("guild-123"),
				RoundID: id,
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.RoundDeletedPayload{
					GuildID:        sharedtypes.GuildID("guild-123"),
					RoundID:        id,
					EventMessageID: "test-event-message-id",
				},
			},
			expectedError: nil,
		},
		{
			name: "delete round fails - delete round error",
			mockDBSetup: func(mockDB *rounddbmocks.MockRoundDB) {
				guildID := sharedtypes.GuildID("guild-123")
				// Mock GetRound call to verify round exists
				mockDB.EXPECT().GetRound(ctx, guildID, gomock.Eq(id)).Return(&roundtypes.Round{ID: id, GuildID: guildID}, nil)
				mockDB.EXPECT().DeleteRound(ctx, guildID, gomock.Eq(id)).Return(errors.New("delete round error"))
			},
			mockQueueSetup: func(mockQueue *queuemocks.MockQueueService) {
				// No queue calls expected when DB delete fails
			},
			payload: roundevents.RoundDeleteAuthorizedPayload{
				GuildID: sharedtypes.GuildID("guild-123"),
				RoundID: id,
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundDeleteErrorPayload{
					RoundDeleteRequest: nil,
					Error:              fmt.Sprintf("failed to delete round from database: %v", errors.New("delete round error")),
				},
			},
			expectedError: nil, // DeleteRound returns failure payload with nil error
		},
		{
			name: "delete round succeeds despite cancel queue jobs error",
			mockDBSetup: func(mockDB *rounddbmocks.MockRoundDB) {
				guildID := sharedtypes.GuildID("guild-123")
				// Mock GetRound call to verify round exists
				mockDB.EXPECT().GetRound(ctx, guildID, gomock.Eq(id)).Return(&roundtypes.Round{
					ID:             id,
					GuildID:        guildID,
					EventMessageID: "test-event-message-id",
				}, nil)
				mockDB.EXPECT().DeleteRound(ctx, guildID, gomock.Eq(id)).Return(nil)
			},
			mockQueueSetup: func(mockQueue *queuemocks.MockQueueService) {
				mockQueue.EXPECT().CancelRoundJobs(ctx, gomock.Eq(id)).Return(errors.New("cancel queue jobs error"))
			},
			payload: roundevents.RoundDeleteAuthorizedPayload{
				GuildID: sharedtypes.GuildID("guild-123"),
				RoundID: id,
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.RoundDeletedPayload{
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
			mockDB := rounddbmocks.NewMockRoundDB(ctrl)
			mockQueue := queuemocks.NewMockQueueService(ctrl)

			tt.mockDBSetup(mockDB)
			tt.mockQueueSetup(mockQueue)

			s := &RoundService{
				RoundDB:        mockDB,
				QueueService:   mockQueue,
				logger:         logger,
				metrics:        mockMetrics,
				tracer:         tracer,
				roundValidator: mockRoundValidator,
				serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc(ctx)
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
