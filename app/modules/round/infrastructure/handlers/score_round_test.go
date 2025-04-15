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

func TestRoundHandlers_HandleScoreUpdateRequest(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testParticipant := sharedtypes.DiscordID("1234567890")
	testScore := sharedtypes.Score(42)

	testPayload := &roundevents.ScoreUpdateRequestPayload{
		RoundID:     testRoundID,
		Participant: testParticipant,
		Score:       &testScore,
	}

	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &roundmetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

	tests := []struct {
		name           string
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
		mockSetup      func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers)
	}{
		{
			name: "Successfully handle ScoreUpdateRequest",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScoreUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateScoreUpdateRequest(
					gomock.Any(),
					roundevents.ScoreUpdateRequestPayload{
						RoundID:     testRoundID,
						Participant: testParticipant,
						Score:       &testScore,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ScoreUpdateValidatedPayload{
							ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
								RoundID:     testRoundID,
								Participant: testParticipant,
								Score:       &testScore,
							},
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundScoreUpdateValidated,
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
			name: "Service failure in ValidateScoreUpdateRequest",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScoreUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateScoreUpdateRequest(
					gomock.Any(),
					roundevents.ScoreUpdateRequestPayload{
						RoundID:     testRoundID,
						Participant: testParticipant,
						Score:       &testScore,
					},
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle ScoreUpdateRequest event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScoreUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateScoreUpdateRequest(
					gomock.Any(),
					roundevents.ScoreUpdateRequestPayload{
						RoundID:     testRoundID,
						Participant: testParticipant,
						Score:       &testScore,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ScoreUpdateValidatedPayload{
							ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
								RoundID:     testRoundID,
								Participant: testParticipant,
								Score:       &testScore,
							},
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundScoreUpdateValidated,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Unknown result from ValidateScoreUpdateRequest",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScoreUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateScoreUpdateRequest(
					gomock.Any(),
					roundevents.ScoreUpdateRequestPayload{
						RoundID:     testRoundID,
						Participant: testParticipant,
						Score:       &testScore,
					},
				).Return(
					roundservice.RoundOperationResult{}, // Return empty result
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service",
		},
		{
			name: "Failure result from ValidateScoreUpdateRequest",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScoreUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateScoreUpdateRequest(
					gomock.Any(),
					roundevents.ScoreUpdateRequestPayload{
						RoundID:     testRoundID,
						Participant: testParticipant,
						Score:       &testScore,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundErrorPayload{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundScoreUpdateError,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
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

			got, err := h.HandleScoreUpdateRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScoreUpdateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleScoreUpdateRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleScoreUpdateRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleScoreUpdateValidated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testParticipant := sharedtypes.DiscordID("1234567890")
	testScore := sharedtypes.Score(42)

	testPayload := &roundevents.ScoreUpdateValidatedPayload{
		ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
			RoundID:     testRoundID,
			Participant: testParticipant,
			Score:       &testScore,
		},
	}

	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &roundmetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

	tests := []struct {
		name           string
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
		mockSetup      func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers)
	}{
		{
			name: "Successfully handle ScoreUpdateValidated",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScoreUpdateValidatedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateParticipantScore(
					gomock.Any(),
					roundevents.ScoreUpdateValidatedPayload{
						ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
							RoundID:     testRoundID,
							Participant: testParticipant,
							Score:       &testScore,
						},
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ParticipantScoreUpdatedPayload{
							RoundID:     testRoundID,
							Participant: testParticipant,
							Score:       testScore,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundParticipantScoreUpdated,
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
			name: "Service failure in UpdateParticipantScore",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScoreUpdateValidatedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateParticipantScore(
					gomock.Any(),
					roundevents.ScoreUpdateValidatedPayload{
						ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
							RoundID:     testRoundID,
							Participant: testParticipant,
							Score:       &testScore,
						},
					},
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle ScoreUpdateValidated event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScoreUpdateValidatedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateParticipantScore(
					gomock.Any(),
					roundevents.ScoreUpdateValidatedPayload{
						ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
							RoundID:     testRoundID,
							Participant: testParticipant,
							Score:       &testScore,
						},
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ParticipantScoreUpdatedPayload{
							RoundID:     testRoundID,
							Participant: testParticipant,
							Score:       testScore,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundParticipantScoreUpdated,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Unknown result from UpdateParticipantScore",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScoreUpdateValidatedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateParticipantScore(
					gomock.Any(),
					roundevents.ScoreUpdateValidatedPayload{
						ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
							RoundID:     testRoundID,
							Participant: testParticipant,
							Score:       &testScore,
						},
					},
				).Return(
					roundservice.RoundOperationResult{}, // Return empty result
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service",
		},
		{
			name: "Failure result from UpdateParticipantScore",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ScoreUpdateValidatedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateParticipantScore(
					gomock.Any(),
					roundevents.ScoreUpdateValidatedPayload{
						ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
							RoundID:     testRoundID,
							Participant: testParticipant,
							Score:       &testScore,
						},
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundErrorPayload{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundScoreUpdateError,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
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

			got, err := h.HandleScoreUpdateValidated(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScoreUpdateValidated() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleScoreUpdateValidated() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleScoreUpdateValidated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleParticipantScoreUpdated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testParticipant := sharedtypes.DiscordID("1234567890")
	testScore := sharedtypes.Score(42)
	testEventMessageID := sharedtypes.RoundID(uuid.New())

	testPayload := &roundevents.ParticipantScoreUpdatedPayload{
		RoundID:        testRoundID,
		Participant:    testParticipant,
		Score:          testScore,
		EventMessageID: &testEventMessageID,
	}

	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &roundmetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

	tests := []struct {
		name           string
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
		mockSetup      func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers)
	}{
		{
			name: "Successfully handle ParticipantScoreUpdated",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantScoreUpdatedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().CheckAllScoresSubmitted(
					gomock.Any(),
					roundevents.ParticipantScoreUpdatedPayload{
						RoundID:        testRoundID,
						Participant:    testParticipant,
						Score:          testScore,
						EventMessageID: &testEventMessageID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.AllScoresSubmittedPayload{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundAllScoresSubmitted,
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
			name: "Service failure in CheckAllScoresSubmitted",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantScoreUpdatedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().CheckAllScoresSubmitted(
					gomock.Any(),
					roundevents.ParticipantScoreUpdatedPayload{
						RoundID:        testRoundID,
						Participant:    testParticipant,
						Score:          testScore,
						EventMessageID: &testEventMessageID,
					},
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle ParticipantScoreUpdated event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantScoreUpdatedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().CheckAllScoresSubmitted(
					gomock.Any(),
					roundevents.ParticipantScoreUpdatedPayload{
						RoundID:        testRoundID,
						Participant:    testParticipant,
						Score:          testScore,
						EventMessageID: &testEventMessageID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.AllScoresSubmittedPayload{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundAllScoresSubmitted,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create all scores submitted message: failed to create result message",
		},
		{
			name: "Unknown result from CheckAllScoresSubmitted",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantScoreUpdatedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().CheckAllScoresSubmitted(
					gomock.Any(),
					roundevents.ParticipantScoreUpdatedPayload{
						RoundID:        testRoundID,
						Participant:    testParticipant,
						Score:          testScore,
						EventMessageID: &testEventMessageID,
					},
				).Return(
					roundservice.RoundOperationResult{}, // Return empty result
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service",
		},
		{
			name: "Failure result from CheckAllScoresSubmitted",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantScoreUpdatedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().CheckAllScoresSubmitted(
					gomock.Any(),
					roundevents.ParticipantScoreUpdatedPayload{
						RoundID:        testRoundID,
						Participant:    testParticipant,
						Score:          testScore,
						EventMessageID: &testEventMessageID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundErrorPayload{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundError,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
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

			got, err := h.HandleParticipantScoreUpdated(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleParticipantScoreUpdated() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleParticipantScoreUpdated() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleParticipantScoreUpdated() = %v, want %v", got, tt.want)
			}
		})
	}
}
