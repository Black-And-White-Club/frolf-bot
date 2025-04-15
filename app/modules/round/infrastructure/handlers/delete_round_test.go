package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundDeleteRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("12345678901234567")

	testPayload := &roundevents.RoundDeleteRequestPayload{
		RoundID:              testRoundID,
		RequestingUserUserID: testUserID,
	}

	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	// Mock dependencies
	mockRoundService := roundmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &roundmetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

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
						*out.(*roundevents.RoundDeleteRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateRoundDeleteRequest(
					gomock.Any(),
					roundevents.RoundDeleteRequestPayload{
						RoundID:              testRoundID,
						RequestingUserUserID: testUserID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundDeleteAuthorizedPayload{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				updateResultPayload := &roundevents.RoundDeleteAuthorizedPayload{
					RoundID: testRoundID,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.RoundDeleteAuthorized,
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
			name: "Service failure in ValidateRoundDeleteRequest",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundDeleteRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateRoundDeleteRequest(
					gomock.Any(),
					roundevents.RoundDeleteRequestPayload{
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
			expectedErrMsg: "failed to handle RoundDeleteRequest event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundDeleteRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateRoundDeleteRequest(
					gomock.Any(),
					roundevents.RoundDeleteRequestPayload{
						RoundID:              testRoundID,
						RequestingUserUserID: testUserID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundDeleteAuthorizedPayload{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				updateResultPayload := &roundevents.RoundDeleteAuthorizedPayload{
					RoundID: testRoundID,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.RoundDeleteAuthorized,
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
						*out.(*roundevents.RoundDeleteRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateRoundDeleteRequest(
					gomock.Any(),
					roundevents.RoundDeleteRequestPayload{
						RoundID:              testRoundID,
						RequestingUserUserID: testUserID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundDeleteErrorPayload{
							Error: "non-error failure",
						},
					},
					nil,
				)

				failureResultPayload := &roundevents.RoundDeleteErrorPayload{
					Error: "non-error failure",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					roundevents.RoundDeleteError,
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
						*out.(*roundevents.RoundDeleteRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateRoundDeleteRequest(
					gomock.Any(),
					roundevents.RoundDeleteRequestPayload{
						RoundID:              testRoundID,
						RequestingUserUserID: testUserID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundDeleteErrorPayload{
							Error: "internal service error",
						},
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle RoundDeleteRequest event: internal service error",
		},
		{
			name: "Unknown result from ValidateRoundDeleteRequest",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundDeleteRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateRoundDeleteRequest(
					gomock.Any(),
					roundevents.RoundDeleteRequestPayload{
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
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, metrics, tracer, mockHelpers)
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

	testPayload := &roundevents.RoundDeleteAuthorizedPayload{
		RoundID: testRoundID,
	}

	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	// Mock dependencies
	mockRoundService := roundmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &roundmetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

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
						*out.(*roundevents.RoundDeleteAuthorizedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().DeleteRound(
					gomock.Any(),
					roundevents.RoundDeleteAuthorizedPayload{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundDeletedPayload{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				updateResultPayload := &roundevents.RoundDeletedPayload{
					RoundID: testRoundID,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.RoundDeleted,
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
						*out.(*roundevents.RoundDeleteAuthorizedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().DeleteRound(
					gomock.Any(),
					roundevents.RoundDeleteAuthorizedPayload{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle RoundDeleteAuthorized event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundDeleteAuthorizedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().DeleteRound(
					gomock.Any(),
					roundevents.RoundDeleteAuthorizedPayload{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundDeletedPayload{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				updateResultPayload := &roundevents.RoundDeletedPayload{
					RoundID: testRoundID,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.RoundDeleted,
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
						*out.(*roundevents.RoundDeleteAuthorizedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().DeleteRound(
					gomock.Any(),
					roundevents.RoundDeleteAuthorizedPayload{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundDeleteErrorPayload{
							Error: "non-error failure",
						},
					},
					nil,
				)

				failureResultPayload := &roundevents.RoundDeleteErrorPayload{
					Error: "non-error failure",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					roundevents.RoundDeleteError,
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
						*out.(*roundevents.RoundDeleteAuthorizedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().DeleteRound(
					gomock.Any(),
					roundevents.RoundDeleteAuthorizedPayload{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundDeleteErrorPayload{
							Error: "internal service error",
						},
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle RoundDeleteAuthorized event: internal service error",
		},
		{
			name: "Unknown result from DeleteRound",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundDeleteAuthorizedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().DeleteRound(
					gomock.Any(),
					roundevents.RoundDeleteAuthorizedPayload{
						RoundID: testRoundID,
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
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload type: expected RoundDeleteAuthorizedPayload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload type: expected RoundDeleteAuthorizedPayload",
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
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, metrics, tracer, mockHelpers)
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
