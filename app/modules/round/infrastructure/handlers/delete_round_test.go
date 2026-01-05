package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundDeleteRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("12345678901234567")

	testPayload := &roundevents.RoundDeleteRequestPayloadV1{
		RoundID:              testRoundID,
		RequestingUserUserID: testUserID,
	}

	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes) // This is the incoming message

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	// Create an expected message for the successful output (RoundDeleteValidated)
	expectedValidatedPayload := roundevents.RoundDeleteValidatedPayloadV1{
		RoundDeleteRequestPayload: *testPayload,
	}
	expectedValidatedPayloadBytes, _ := json.Marshal(expectedValidatedPayload)
	expectedValidatedMsg := message.NewMessage("test-id", expectedValidatedPayloadBytes)

	// Mock dependencies
	mockRoundService := roundmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockSetup      func()
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle RoundDeleteRequest",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundDeleteRequestPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateRoundDeleteRequest(
					gomock.Any(),
					roundevents.RoundDeleteRequestPayloadV1{
						RoundID:              testRoundID,
						RequestingUserUserID: testUserID,
					},
				).Return(
					roundservice.RoundOperationResult{
						// CORRECTED: Service returns RoundDeleteValidatedPayload on success
						Success: &roundevents.RoundDeleteValidatedPayloadV1{
							RoundDeleteRequestPayload: *testPayload, // Populate with the original request payload
						},
					},
					nil,
				)

				// The payload passed to CreateResultMessage is the validated payload
				validatedPayloadForMock := roundevents.RoundDeleteValidatedPayloadV1{
					RoundDeleteRequestPayload: *testPayload,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					validatedPayloadForMock, // Use the correct payload type for the mock expectation
					roundevents.RoundDeleteValidatedV1,
				).Return(expectedValidatedMsg, nil) // Return the message created with RoundDeleteValidatedPayload
			},
			msg:     testMsg,
			want:    []*message.Message{expectedValidatedMsg}, // CORRECTED: Expect the message created from RoundDeleteValidatedPayload
			wantErr: false,
		},
		{
			name: "Fail to unmarshal payload",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload",
		},
		{
			name: "Service failure in ValidateRoundDeleteRequest",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundDeleteRequestPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateRoundDeleteRequest(
					gomock.Any(),
					roundevents.RoundDeleteRequestPayloadV1{
						RoundID:              testRoundID,
						RequestingUserUserID: testUserID,
					},
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to validate RoundDeleteRequest: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundDeleteRequestPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateRoundDeleteRequest(
					gomock.Any(),
					roundevents.RoundDeleteRequestPayloadV1{
						RoundID:              testRoundID,
						RequestingUserUserID: testUserID,
					},
				).Return(
					roundservice.RoundOperationResult{
						// CORRECTED: Service returns RoundDeleteValidatedPayload on success
						Success: &roundevents.RoundDeleteValidatedPayloadV1{
							RoundDeleteRequestPayload: *testPayload, // Populate with the original request payload
						},
					},
					nil,
				)

				// The payload passed to CreateResultMessage is the validated payload
				validatedPayloadForMock := roundevents.RoundDeleteValidatedPayloadV1{
					RoundDeleteRequestPayload: *testPayload,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					validatedPayloadForMock, // Use the correct payload type for the mock expectation
					roundevents.RoundDeleteValidatedV1,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Service failure with non-error result",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundDeleteRequestPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateRoundDeleteRequest(
					gomock.Any(),
					roundevents.RoundDeleteRequestPayloadV1{
						RoundID:              testRoundID,
						RequestingUserUserID: testUserID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundDeleteErrorPayloadV1{
							Error: "non-error failure",
						},
					},
					nil,
				)

				failureResultPayload := &roundevents.RoundDeleteErrorPayloadV1{
					Error: "non-error failure",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					roundevents.RoundDeleteErrorV1,
				).Return(testMsg, nil)
			},
			msg:            testMsg,
			want:           []*message.Message{testMsg},
			wantErr:        false,
			expectedErrMsg: "",
		},
		{
			name: "Service failure with error result and CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundDeleteRequestPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateRoundDeleteRequest(
					gomock.Any(),
					roundevents.RoundDeleteRequestPayloadV1{
						RoundID:              testRoundID,
						RequestingUserUserID: testUserID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundDeleteErrorPayloadV1{
							Error: "internal service error",
						},
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to validate RoundDeleteRequest: internal service error",
		},
		{
			name: "Unknown result from ValidateRoundDeleteRequest",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundDeleteRequestPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateRoundDeleteRequest(
					gomock.Any(),
					roundevents.RoundDeleteRequestPayloadV1{
						RoundID:              testRoundID,
						RequestingUserUserID: testUserID,
					},
				).Return(
					roundservice.RoundOperationResult{},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service",
		},
		{
			name: "Invalid payload type",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload type: expected RoundDeleteRequestPayload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload type: expected RoundDeleteRequestPayload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := &RoundHandlers{
				roundService: mockRoundService,
				logger:       logger,
				tracer:       tracer,
				metrics:      metrics,
				helpers:      mockHelpers,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, tracer, mockHelpers, metrics)
				},
			}

			got, err := h.HandleRoundDeleteRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundDeleteRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundDeleteRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundDeleteRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundDeleteAuthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRoundID := sharedtypes.RoundID(uuid.New())

	testPayload := &roundevents.RoundDeleteAuthorizedPayloadV1{
		RoundID: testRoundID,
	}

	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	// Mock dependencies
	mockRoundService := roundmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockSetup      func()
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle RoundDeleteAuthorized",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundDeleteAuthorizedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().DeleteRound(
					gomock.Any(),
					roundevents.RoundDeleteAuthorizedPayloadV1{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundDeletedPayloadV1{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				updateResultPayload := &roundevents.RoundDeletedPayloadV1{
					RoundID: testRoundID,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.RoundDeletedV1,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Fail to unmarshal payload",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload",
		},
		{
			name: "Service failure in DeleteRound",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundDeleteAuthorizedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().DeleteRound(
					gomock.Any(),
					roundevents.RoundDeleteAuthorizedPayloadV1{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:     testMsg,
			want:    nil,
			wantErr: true,
			// Direct error from handler, no wrapper prefix
			expectedErrMsg: "RoundService.DeleteRound failed: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundDeleteAuthorizedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().DeleteRound(
					gomock.Any(),
					roundevents.RoundDeleteAuthorizedPayloadV1{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundDeletedPayloadV1{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				updateResultPayload := &roundevents.RoundDeletedPayloadV1{
					RoundID: testRoundID,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.RoundDeletedV1,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:     testMsg,
			want:    nil,
			wantErr: true,
			// Direct error from handler, no wrapper prefix
			expectedErrMsg: "failed to create RoundDeleted success message: failed to create result message",
		},
		{
			name: "Service failure with database error",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundDeleteAuthorizedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().DeleteRound(
					gomock.Any(),
					roundevents.RoundDeleteAuthorizedPayloadV1{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundDeleteErrorPayloadV1{
							RoundDeleteRequest: &roundevents.RoundDeleteRequestPayloadV1{
								RoundID: testRoundID,
							},
							Error: "failed to delete round from database: connection timeout",
						},
					},
					fmt.Errorf("failed to delete round %s from DB: connection timeout", testRoundID.String()),
				)
			},
			msg:            testMsg,
			want:           nil, // When service returns an error, handler returns nil messages
			wantErr:        true,
			expectedErrMsg: fmt.Sprintf("RoundService.DeleteRound failed: failed to delete round %s from DB: connection timeout", testRoundID.String()),
		},
		{
			name: "Service failure with error result and CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundDeleteAuthorizedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().DeleteRound(
					gomock.Any(),
					roundevents.RoundDeleteAuthorizedPayloadV1{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundDeleteErrorPayloadV1{
							Error: "internal service error",
						},
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:     testMsg,
			want:    nil,
			wantErr: true,
			// When both service error AND failure result exist, the service error takes precedence
			expectedErrMsg: "RoundService.DeleteRound failed: internal service error",
		},
		{
			name: "Unknown result from DeleteRound",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundDeleteAuthorizedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().DeleteRound(
					gomock.Any(),
					roundevents.RoundDeleteAuthorizedPayloadV1{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{},
					nil,
				)
			},
			msg:     testMsg,
			want:    nil,
			wantErr: true,
			// Direct error from handler, no wrapper prefix
			expectedErrMsg: fmt.Sprintf("unexpected result from RoundService.DeleteRound for round %s", testRoundID.String()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := &RoundHandlers{
				roundService: mockRoundService,
				logger:       logger,
				tracer:       tracer,
				metrics:      metrics,
				helpers:      mockHelpers,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, tracer, mockHelpers, metrics)
				},
			}

			got, err := h.HandleRoundDeleteAuthorized(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundDeleteAuthorized() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundDeleteAuthorized() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundDeleteAuthorized() = %v, want %v", got, tt.want)
			}
		})
	}
}
