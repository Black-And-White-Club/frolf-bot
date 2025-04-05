package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/round"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleAllScoresSubmitted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRoundID := sharedtypes.RoundID(uuid.New())

	testPayload := &roundevents.AllScoresSubmittedPayload{
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
			name: "Successfully handle AllScoresSubmitted",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.AllScoresSubmittedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().FinalizeRound(
					gomock.Any(),
					roundevents.AllScoresSubmittedPayload{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundFinalizedPayload{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				updateResultPayload := &roundevents.RoundFinalizedPayload{
					RoundID: testRoundID,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.RoundFinalized,
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
			name: "Service failure in FinalizeRound",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.AllScoresSubmittedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().FinalizeRound(
					gomock.Any(),
					roundevents.AllScoresSubmittedPayload{
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
			expectedErrMsg: "failed to handle AllScoresSubmitted event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.AllScoresSubmittedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().FinalizeRound(
					gomock.Any(),
					roundevents.AllScoresSubmittedPayload{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundFinalizedPayload{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				updateResultPayload := &roundevents.RoundFinalizedPayload{
					RoundID: testRoundID,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.RoundFinalized,
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
						*out.(*roundevents.AllScoresSubmittedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().FinalizeRound(
					gomock.Any(),
					roundevents.AllScoresSubmittedPayload{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundFinalizationErrorPayload{
							Error: "non-error failure",
						},
					},
					nil,
				)

				failureResultPayload := &roundevents.RoundFinalizationErrorPayload{
					Error: "non-error failure",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					roundevents.RoundFinalizationError,
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
						*out.(*roundevents.AllScoresSubmittedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().FinalizeRound(
					gomock.Any(),
					roundevents.AllScoresSubmittedPayload{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundFinalizationErrorPayload{
							Error: "internal service error",
						},
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle AllScoresSubmitted event: internal service error",
		},
		{
			name: "Unknown result from FinalizeRound",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.AllScoresSubmittedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().FinalizeRound(
					gomock.Any(),
					roundevents.AllScoresSubmittedPayload{
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
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload type: expected AllScoresSubmittedPayload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload type: expected AllScoresSubmittedPayload",
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

			got, err := h.HandleAllScoresSubmitted(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleAllScoresSubmitted() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleAllScoresSubmitted() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleAllScoresSubmitted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundFinalized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRoundID := sharedtypes.RoundID(uuid.New())

	testPayload := &roundevents.RoundFinalizedPayload{
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
			name: "Successfully handle RoundFinalized",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundFinalizedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().NotifyScoreModule(
					gomock.Any(),
					roundevents.RoundFinalizedPayload{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ProcessRoundScoresRequestPayload{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				updateResultPayload := &roundevents.ProcessRoundScoresRequestPayload{
					RoundID: testRoundID,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.ProcessRoundScoresRequest,
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
			name: "Service failure in NotifyScoreModule",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundFinalizedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().NotifyScoreModule(
					gomock.Any(),
					roundevents.RoundFinalizedPayload{
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
			expectedErrMsg: "failed to handle RoundFinalized event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundFinalizedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().NotifyScoreModule(
					gomock.Any(),
					roundevents.RoundFinalizedPayload{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ProcessRoundScoresRequestPayload{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				updateResultPayload := &roundevents.ProcessRoundScoresRequestPayload{
					RoundID: testRoundID,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.ProcessRoundScoresRequest,
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
						*out.(*roundevents.RoundFinalizedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().NotifyScoreModule(
					gomock.Any(),
					roundevents.RoundFinalizedPayload{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundFinalizationErrorPayload{
							Error: "non-error failure",
						},
					},
					nil,
				)

				failureResultPayload := &roundevents.RoundFinalizationErrorPayload{
					Error: "non-error failure",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					roundevents.RoundFinalizationError,
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
						*out.(*roundevents.RoundFinalizedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().NotifyScoreModule(
					gomock.Any(),
					roundevents.RoundFinalizedPayload{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundFinalizationErrorPayload{
							Error: "internal service error",
						},
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle RoundFinalized event: internal service error",
		},
		{
			name: "Unknown result from NotifyScoreModule",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundFinalizedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().NotifyScoreModule(
					gomock.Any(),
					roundevents.RoundFinalizedPayload{
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
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload type: expected RoundFinalizedPayload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload type: expected RoundFinalizedPayload",
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

			got, err := h.HandleRoundFinalized(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundFinalized() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundFinalized() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundFinalized() = %v, want %v", got, tt.want)
			}
		})
	}
}
