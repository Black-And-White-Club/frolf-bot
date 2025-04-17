package leaderboardhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboardmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleLeaderboardUpdateRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRoundID := sharedtypes.RoundID(uuid.New())
	testSortedParticipantTags := []string{
		"1:12345678901234567",
		"2:12345678901234568",
	}

	testPayload := &leaderboardevents.LeaderboardUpdateRequestedPayload{
		RoundID:               testRoundID,
		SortedParticipantTags: testSortedParticipantTags,
		Source:                "round",
		UpdateID:              testRoundID.String(),
	}
	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	// Mock dependencies
	mockLeaderboardService := leaderboardmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &leaderboardmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockSetup      func()
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle LeaderboardUpdateRequested",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.LeaderboardUpdateRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().UpdateLeaderboard(
					gomock.Any(),
					testRoundID,
					testSortedParticipantTags,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.LeaderboardUpdatedPayload{
							LeaderboardID:   1,
							RoundID:         testRoundID,
							LeaderboardData: map[int]string{},
						},
					},
					nil,
				)

				updateResultPayload := &leaderboardevents.LeaderboardUpdatedPayload{
					LeaderboardID:   1,
					RoundID:         testRoundID,
					LeaderboardData: map[int]string{},
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					leaderboardevents.LeaderboardUpdated,
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
			name: "Service failure in UpdateLeaderboard",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.LeaderboardUpdateRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().UpdateLeaderboard(
					gomock.Any(),
					testRoundID,
					testSortedParticipantTags,
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to update leaderboard: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.LeaderboardUpdateRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().UpdateLeaderboard(
					gomock.Any(),
					testRoundID,
					testSortedParticipantTags,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.LeaderboardUpdatedPayload{
							LeaderboardID:   1,
							RoundID:         testRoundID,
							LeaderboardData: map[int]string{},
						},
					},
					nil,
				)

				updateResultPayload := &leaderboardevents.LeaderboardUpdatedPayload{
					LeaderboardID:   1,
					RoundID:         testRoundID,
					LeaderboardData: map[int]string{},
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					leaderboardevents.LeaderboardUpdated,
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
						*out.(*leaderboardevents.LeaderboardUpdateRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().UpdateLeaderboard(
					gomock.Any(),
					testRoundID,
					testSortedParticipantTags,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.LeaderboardUpdateFailedPayload{
							RoundID: testRoundID,
							Reason:  "non-error failure",
						},
					},
					nil,
				)

				failureResultPayload := &leaderboardevents.LeaderboardUpdateFailedPayload{
					RoundID: testRoundID,
					Reason:  "non-error failure",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					leaderboardevents.LeaderboardUpdateFailed,
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
						*out.(*leaderboardevents.LeaderboardUpdateRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().UpdateLeaderboard(
					gomock.Any(),
					testRoundID,
					testSortedParticipantTags,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.LeaderboardUpdateFailedPayload{
							RoundID: testRoundID,
							Reason:  "internal service error",
						},
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to update leaderboard: internal service error",
		},
		{
			name: "Unknown result from UpdateLeaderboard",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.LeaderboardUpdateRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().UpdateLeaderboard(
					gomock.Any(),
					testRoundID,
					testSortedParticipantTags,
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
		{
			name: "Invalid payload type",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload type: expected LeaderboardUpdateRequestedPayload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload type: expected LeaderboardUpdateRequestedPayload",
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

			got, err := h.HandleLeaderboardUpdateRequested(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleLeaderboardUpdateRequested() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleLeaderboardUpdateRequested() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleLeaderboardUpdateRequested() = %v, want %v", got, tt.want)
			}
		})
	}
}
