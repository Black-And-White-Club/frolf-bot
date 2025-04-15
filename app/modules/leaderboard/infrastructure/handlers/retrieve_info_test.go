package leaderboardhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboardmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleGetLeaderboardRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testMsg := message.NewMessage("test-id", []byte{})

	// Mock dependencies
	mockLeaderboardService := leaderboardmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &leaderboardmetrics.NoOpMetrics{}
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
			name: "Successfully handle GetLeaderboardRequest",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.GetLeaderboardRequestPayload) = leaderboardevents.GetLeaderboardRequestPayload{}
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetLeaderboard(
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.GetLeaderboardResponsePayload{
							Leaderboard: []leaderboardevents.LeaderboardEntry{},
						},
					},
					nil,
				)

				updateResultPayload := &leaderboardevents.GetLeaderboardResponsePayload{
					Leaderboard: []leaderboardevents.LeaderboardEntry{},
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					leaderboardevents.GetLeaderboardResponse,
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
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload",
		},
		{
			name: "Service failure in GetLeaderboard",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.GetLeaderboardRequestPayload) = leaderboardevents.GetLeaderboardRequestPayload{}
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetLeaderboard(
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to get leaderboard: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.GetLeaderboardRequestPayload) = leaderboardevents.GetLeaderboardRequestPayload{}
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetLeaderboard(
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.GetLeaderboardResponsePayload{
							Leaderboard: []leaderboardevents.LeaderboardEntry{},
						},
					},
					nil,
				)

				updateResultPayload := &leaderboardevents.GetLeaderboardResponsePayload{
					Leaderboard: []leaderboardevents.LeaderboardEntry{},
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					leaderboardevents.GetLeaderboardResponse,
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
						*out.(*leaderboardevents.GetLeaderboardRequestPayload) = leaderboardevents.GetLeaderboardRequestPayload{}
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetLeaderboard(
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.GetLeaderboardFailedPayload{
							Reason: "non-error failure",
						},
					},
					nil,
				)

				failureResultPayload := &leaderboardevents.GetLeaderboardFailedPayload{
					Reason: "non-error failure",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					leaderboardevents.GetLeaderboardFailed,
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
						*out.(*leaderboardevents.GetLeaderboardRequestPayload) = leaderboardevents.GetLeaderboardRequestPayload{}
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetLeaderboard(
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.GetLeaderboardFailedPayload{
							Reason: "internal service error",
						},
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to get leaderboard: internal service error",
		},
		{
			name: "Unknown result from GetLeaderboard",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.GetLeaderboardRequestPayload) = leaderboardevents.GetLeaderboardRequestPayload{}
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetLeaderboard(
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := &LeaderboardHandlers{
				leaderboardService: mockLeaderboardService,
				logger:             logger,
				tracer:             tracer,
				metrics:            metrics,
				helpers:            mockHelpers,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, metrics, tracer, mockHelpers)
				},
			}

			got, err := h.HandleGetLeaderboardRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetLeaderboardRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleGetLeaderboardRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleGetLeaderboardRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleGetTagByUserIDRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUserID := sharedtypes.DiscordID("12345678901234567")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testTag := sharedtypes.TagNumber(1)

	testPayload := &leaderboardevents.TagNumberRequestPayload{
		UserID:  testUserID,
		RoundID: testRoundID,
	}
	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	// Mock dependencies
	mockLeaderboardService := leaderboardmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &leaderboardmetrics.NoOpMetrics{}
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
			name: "Successfully handle GetTagByUserIDRequest",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagNumberRequestPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					testUserID,
					testRoundID,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.GetTagNumberResponsePayload{
							TagNumber: &testTag,
							UserID:    testUserID,
							RoundID:   testRoundID,
						},
					},
					nil,
				)

				updateResultPayload := &leaderboardevents.GetTagNumberResponsePayload{
					TagNumber: &testTag,
					UserID:    testUserID,
					RoundID:   testRoundID,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					leaderboardevents.GetTagNumberResponse,
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
			name: "Service failure in GetTagByUserID",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagNumberRequestPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					testUserID,
					testRoundID,
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to get tag by userID: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagNumberRequestPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					testUserID,
					testRoundID,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.GetTagNumberResponsePayload{
							TagNumber: &testTag,
							UserID:    testUserID,
							RoundID:   testRoundID,
						},
					},
					nil,
				)

				updateResultPayload := &leaderboardevents.GetTagNumberResponsePayload{
					TagNumber: &testTag,
					UserID:    testUserID,
					RoundID:   testRoundID,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					leaderboardevents.GetTagNumberResponse,
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
						*out.(*leaderboardevents.TagNumberRequestPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					testUserID,
					testRoundID,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.GetTagNumberFailedPayload{
							Reason: "non-error failure",
						},
					},
					nil,
				)

				failureResultPayload := &leaderboardevents.GetTagNumberFailedPayload{
					Reason: "non-error failure",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					leaderboardevents.GetTagNumberFailed,
				).Return(testMsg, nil)
			},
			msg:            testMsg,
			want:           []*message.Message{testMsg},
			wantErr:        false,
			expectedErrMsg: "",
		},
		{
			name: "Unknown result from GetTagByUserID",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagNumberRequestPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					testUserID,
					testRoundID,
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := &LeaderboardHandlers{
				leaderboardService: mockLeaderboardService,
				logger:             logger,
				tracer:             tracer,
				metrics:            metrics,
				helpers:            mockHelpers,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, metrics, tracer, mockHelpers)
				},
			}

			got, err := h.HandleGetTagByUserIDRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetTagByUserIDRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleGetTagByUserIDRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleGetTagByUserIDRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
