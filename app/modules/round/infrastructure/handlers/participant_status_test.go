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
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleParticipantJoinRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("12345678901234567")

	testPayload := &roundevents.ParticipantJoinRequestPayload{
		RoundID:  testRoundID,
		UserID:   testUserID,
		Response: roundtypes.ResponseAccept,
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
			name: "Successfully handle ParticipantJoinRequest",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantJoinRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().CheckParticipantStatus(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:  testRoundID,
						UserID:   testUserID,
						Response: roundtypes.ResponseAccept,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ParticipantJoinRequestPayload{
							RoundID:  testRoundID,
							UserID:   testUserID,
							Response: roundtypes.ResponseAccept,
						},
					},
					nil,
				)

				updateResultPayload := &roundevents.ParticipantJoinRequestPayload{
					RoundID:  testRoundID,
					UserID:   testUserID,
					Response: roundtypes.ResponseAccept,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.RoundParticipantJoinValidationRequest,
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
			name: "Service failure in CheckParticipantStatus",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantJoinRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().CheckParticipantStatus(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:  testRoundID,
						UserID:   testUserID,
						Response: roundtypes.ResponseAccept,
					},
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle ParticipantJoinRequest event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantJoinRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().CheckParticipantStatus(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:  testRoundID,
						UserID:   testUserID,
						Response: roundtypes.ResponseAccept,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ParticipantJoinRequestPayload{
							RoundID:  testRoundID,
							UserID:   testUserID,
							Response: roundtypes.ResponseAccept,
						},
					},
					nil,
				)

				updateResultPayload := &roundevents.ParticipantJoinRequestPayload{
					RoundID:  testRoundID,
					UserID:   testUserID,
					Response: roundtypes.ResponseAccept,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.RoundParticipantJoinValidationRequest,
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
						*out.(*roundevents.ParticipantJoinRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().CheckParticipantStatus(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:  testRoundID,
						UserID:   testUserID,
						Response: roundtypes.ResponseAccept,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundParticipantJoinErrorPayload{
							Error: "non-error failure",
						},
					},
					nil,
				)

				failureResultPayload := &roundevents.RoundParticipantJoinErrorPayload{
					Error: "non-error failure",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					roundevents.RoundParticipantStatusCheckError,
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
						*out.(*roundevents.ParticipantJoinRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().CheckParticipantStatus(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:  testRoundID,
						UserID:   testUserID,
						Response: roundtypes.ResponseAccept,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundParticipantJoinErrorPayload{
							Error: "internal service error",
						},
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle ParticipantJoinRequest event: internal service error",
		},
		{
			name: "Unknown result from CheckParticipantStatus",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantJoinRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().CheckParticipantStatus(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:  testRoundID,
						UserID:   testUserID,
						Response: roundtypes.ResponseAccept,
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
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload type: expected ParticipantJoinRequestPayload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload type: expected ParticipantJoinRequestPayload",
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

			got, err := h.HandleParticipantJoinRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleParticipantJoinRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleParticipantJoinRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleParticipantJoinRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleParticipantJoinValidationRequest(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("12345678901234567")

	testPayload := &roundevents.ParticipantJoinValidationRequestPayload{
		RoundID:  testRoundID,
		UserID:   testUserID,
		Response: roundtypes.ResponseAccept,
	}

	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &roundmetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

	tests := []struct {
		name           string
		mockSetup      func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers)
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle ParticipantJoinValidationRequest",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantJoinValidationRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateParticipantJoinRequest(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:  testRoundID,
						UserID:   testUserID,
						Response: roundtypes.ResponseAccept,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ParticipantJoinRequestPayload{
							RoundID:  testRoundID,
							UserID:   testUserID,
							Response: roundtypes.ResponseAccept,
						},
					},
					nil,
				)

				updateResultPayload := roundevents.TagLookupRequestPayload{
					RoundID:  testRoundID,
					UserID:   testUserID,
					Response: roundtypes.ResponseAccept,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.LeaderboardGetTagNumberRequest,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Fail to unmarshal payload",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload",
		},
		{
			name: "Service failure in ValidateParticipantJoinRequest",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantJoinValidationRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateParticipantJoinRequest(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:  testRoundID,
						UserID:   testUserID,
						Response: roundtypes.ResponseAccept,
					},
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle ParticipantJoinValidationRequest event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantJoinValidationRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateParticipantJoinRequest(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:  testRoundID,
						UserID:   testUserID,
						Response: roundtypes.ResponseAccept,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ParticipantJoinRequestPayload{
							RoundID:  testRoundID,
							UserID:   testUserID,
							Response: roundtypes.ResponseAccept,
						},
					},
					nil,
				)

				updateResultPayload := roundevents.TagLookupRequestPayload{
					RoundID:  testRoundID,
					UserID:   testUserID,
					Response: roundtypes.ResponseAccept,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.LeaderboardGetTagNumberRequest,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Service failure with non-error result",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantJoinValidationRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateParticipantJoinRequest(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:  testRoundID,
						UserID:   testUserID,
						Response: roundtypes.ResponseAccept,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundParticipantJoinErrorPayload{
							Error: "non-error failure",
						},
					},
					nil,
				)

				failureResultPayload := &roundevents.RoundParticipantJoinErrorPayload{
					Error: "non-error failure",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					roundevents.RoundParticipantJoinError,
				).Return(testMsg, nil)
			},
			msg:            testMsg,
			want:           []*message.Message{testMsg},
			wantErr:        false,
			expectedErrMsg: "",
		},
		{
			name: "Service failure with error result and CreateResultMessage fails",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantJoinValidationRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateParticipantJoinRequest(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:  testRoundID,
						UserID:   testUserID,
						Response: roundtypes.ResponseAccept,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundParticipantJoinErrorPayload{
							Error: "internal service error",
						},
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle ParticipantJoinValidationRequest event: internal service error",
		},
		{
			name: "Unknown result from ValidateParticipantJoinRequest",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantJoinValidationRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateParticipantJoinRequest(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:  testRoundID,
						UserID:   testUserID,
						Response: roundtypes.ResponseAccept,
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
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload type: expected ParticipantJoinValidationRequestPayload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload type: expected ParticipantJoinValidationRequestPayload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundService := roundmocks.NewMockService(ctrl)
			mockHelpers := mocks.NewMockHelpers(ctrl)

			tt.mockSetup(mockRoundService, mockHelpers)

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

			got, err := h.HandleParticipantJoinValidationRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleParticipantJoinValidationRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleParticipantJoinValidationRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleParticipantJoinValidationRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleParticipantRemovalRequest(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("12345678901234567")

	testPayload := &roundevents.ParticipantRemovalRequestPayload{
		RoundID: testRoundID,
		UserID:  testUserID,
	}

	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &roundmetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

	tests := []struct {
		name           string
		mockSetup      func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers)
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle ParticipantRemovalRequest",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantRemovalRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ParticipantRemoval(
					gomock.Any(),
					roundevents.ParticipantRemovalRequestPayload{
						RoundID: testRoundID,
						UserID:  testUserID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ParticipantRemovedPayload{
							RoundID: testRoundID,
							UserID:  testUserID,
						},
					},
					nil,
				)

				updateResultPayload := &roundevents.ParticipantRemovedPayload{
					RoundID: testRoundID,
					UserID:  testUserID,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.RoundParticipantRemoved,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Fail to unmarshal payload",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload",
		},
		{
			name: "Service failure in ParticipantRemoval",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantRemovalRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ParticipantRemoval(
					gomock.Any(),
					roundevents.ParticipantRemovalRequestPayload{
						RoundID: testRoundID,
						UserID:  testUserID,
					},
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle ParticipantRemovalRequest event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantRemovalRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ParticipantRemoval(
					gomock.Any(),
					roundevents.ParticipantRemovalRequestPayload{
						RoundID: testRoundID,
						UserID:  testUserID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ParticipantRemovedPayload{
							RoundID: testRoundID,
							UserID:  testUserID,
						},
					},
					nil,
				)

				updateResultPayload := &roundevents.ParticipantRemovedPayload{
					RoundID: testRoundID,
					UserID:  testUserID,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.RoundParticipantRemoved,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Service failure with non-error result",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantRemovalRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ParticipantRemoval(
					gomock.Any(),
					roundevents.ParticipantRemovalRequestPayload{
						RoundID: testRoundID,
						UserID:  testUserID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.ParticipantRemovalErrorPayload{
							Error: "non-error failure",
						},
					},
					nil,
				)

				failureResultPayload := &roundevents.ParticipantRemovalErrorPayload{
					Error: "non-error failure",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					roundevents.RoundParticipantRemovalError,
				).Return(testMsg, nil)
			},
			msg:            testMsg,
			want:           []*message.Message{testMsg},
			wantErr:        false,
			expectedErrMsg: "",
		},
		{
			name: "Service failure with error result and CreateResultMessage fails",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantRemovalRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ParticipantRemoval(
					gomock.Any(),
					roundevents.ParticipantRemovalRequestPayload{
						RoundID: testRoundID,
						UserID:  testUserID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.ParticipantRemovalErrorPayload{
							Error: "internal service error",
						},
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle ParticipantRemovalRequest event: internal service error",
		},
		{
			name: "Unknown result from ParticipantRemoval",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantRemovalRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ParticipantRemoval(
					gomock.Any(),
					roundevents.ParticipantRemovalRequestPayload{
						RoundID: testRoundID,
						UserID:  testUserID,
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
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload type: expected ParticipantRemovalRequestPayload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload type: expected ParticipantRemovalRequestPayload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundService := roundmocks.NewMockService(ctrl)
			mockHelpers := mocks.NewMockHelpers(ctrl)

			tt.mockSetup(mockRoundService, mockHelpers)

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

			got, err := h.HandleParticipantRemovalRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleParticipantRemovalRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleParticipantRemovalRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleParticipantRemovalRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
