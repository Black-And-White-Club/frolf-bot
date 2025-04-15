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

func TestLeaderboardHandlers_HandleTagAssignment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUserID := sharedtypes.DiscordID(uuid.New().String())
	testTagNumber := sharedtypes.TagNumber(1)
	testAssignmentID := sharedtypes.RoundID(uuid.New())
	testSource := "test-source"
	testReason := "test-reason"
	testUpdateType := "test-update-type"

	testPayload := &leaderboardevents.TagAssignmentRequestedPayload{
		UserID:    testUserID,
		TagNumber: &testTagNumber,
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
			name: "Successfully handle TagAssignmentRequested",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().TagAssignmentRequested(
					gomock.Any(),
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.TagAssignedPayload{
							UserID:       testUserID,
							TagNumber:    &testTagNumber,
							AssignmentID: testAssignmentID,
							Source:       testSource,
						},
					},
					nil,
				)

				successResultPayload := &leaderboardevents.TagAssignedPayload{
					UserID:       testUserID,
					TagNumber:    &testTagNumber,
					AssignmentID: testAssignmentID,
					Source:       testSource,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					successResultPayload,
					leaderboardevents.LeaderboardTagAssignmentSuccess,
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
			name: "Service failure in TagAssignmentRequested",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().TagAssignmentRequested(
					gomock.Any(),
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle TagAssignmentRequested event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().TagAssignmentRequested(
					gomock.Any(),
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.TagAssignedPayload{
							UserID:       testUserID,
							TagNumber:    &testTagNumber,
							AssignmentID: testAssignmentID,
							Source:       testSource,
						},
					},
					nil,
				)

				successResultPayload := &leaderboardevents.TagAssignedPayload{
					UserID:       testUserID,
					TagNumber:    &testTagNumber,
					AssignmentID: testAssignmentID,
					Source:       testSource,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					successResultPayload,
					leaderboardevents.LeaderboardTagAssignmentSuccess,
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
						*out.(*leaderboardevents.TagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().TagAssignmentRequested(
					gomock.Any(),
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.TagAssignmentFailedPayload{
							UserID:     testUserID,
							TagNumber:  &testTagNumber,
							Source:     testSource,
							UpdateType: testUpdateType,
							Reason:     testReason,
						},
					},
					nil,
				)

				failureResultPayload := &leaderboardevents.TagAssignmentFailedPayload{
					UserID:     testUserID,
					TagNumber:  &testTagNumber,
					Source:     testSource,
					UpdateType: testUpdateType,
					Reason:     testReason,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					leaderboardevents.LeaderboardTagAssignmentFailed,
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
						*out.(*leaderboardevents.TagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().TagAssignmentRequested(
					gomock.Any(),
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.TagAssignmentFailedPayload{
							UserID:     testUserID,
							TagNumber:  &testTagNumber,
							Source:     testSource,
							UpdateType: testUpdateType,
							Reason:     testReason,
						},
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle TagAssignmentRequested event: internal service error",
		},
		{
			name: "Unknown result from TagAssignmentRequested",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().TagAssignmentRequested(
					gomock.Any(),
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					nil,
				)

				// Ensure no calls to CreateResultMessage are made
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service",
		},
		{
			name: "Invalid payload type",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload type: expected TagAssignmentRequestedPayload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload type: expected TagAssignmentRequestedPayload",
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

			got, err := h.HandleTagAssignment(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagAssignment() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleTagAssignment() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleTagAssignment() = %v, want %v", got, tt.want)
			}
		})
	}
}
