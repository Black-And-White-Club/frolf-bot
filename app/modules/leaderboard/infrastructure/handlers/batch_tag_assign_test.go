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

func TestLeaderboardHandlers_HandleBatchTagAssignmentRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRequestingUserID := sharedtypes.DiscordID("12345678901234567")
	testBatchID := uuid.New().String()
	testAssignments := []leaderboardevents.TagAssignmentInfo{
		{
			UserID:    sharedtypes.DiscordID("12345678901234567"),
			TagNumber: sharedtypes.TagNumber(1),
		},
	}

	testPayload := &leaderboardevents.BatchTagAssignmentRequestedPayload{
		RequestingUserID: testRequestingUserID,
		BatchID:          testBatchID,
		Assignments:      testAssignments,
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
			name: "Successfully handle BatchTagAssignmentRequested",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.BatchTagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().BatchTagAssignmentRequested(
					gomock.Any(),
					leaderboardevents.BatchTagAssignmentRequestedPayload{
						RequestingUserID: testRequestingUserID,
						BatchID:          testBatchID,
						Assignments:      testAssignments,
					},
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.BatchTagAssignedPayload{
							RequestingUserID: testRequestingUserID,
							BatchID:          testBatchID,
							AssignmentCount:  len(testAssignments),
							Assignments:      testAssignments,
						},
					},
					nil,
				)

				updateResultPayload := &leaderboardevents.BatchTagAssignedPayload{
					RequestingUserID: testRequestingUserID,
					BatchID:          testBatchID,
					AssignmentCount:  len(testAssignments),
					Assignments:      testAssignments,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					leaderboardevents.LeaderboardBatchTagAssigned,
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
			name: "Service failure in BatchTagAssignmentRequested",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.BatchTagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().BatchTagAssignmentRequested(
					gomock.Any(),
					leaderboardevents.BatchTagAssignmentRequestedPayload{
						RequestingUserID: testRequestingUserID,
						BatchID:          testBatchID,
						Assignments:      testAssignments,
					},
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle BatchTagAssignmentRequested event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.BatchTagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().BatchTagAssignmentRequested(
					gomock.Any(),
					leaderboardevents.BatchTagAssignmentRequestedPayload{
						RequestingUserID: testRequestingUserID,
						BatchID:          testBatchID,
						Assignments:      testAssignments,
					},
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.BatchTagAssignedPayload{
							RequestingUserID: testRequestingUserID,
							BatchID:          testBatchID,
							AssignmentCount:  len(testAssignments),
							Assignments:      testAssignments,
						},
					},
					nil,
				)

				updateResultPayload := &leaderboardevents.BatchTagAssignedPayload{
					RequestingUserID: testRequestingUserID,
					BatchID:          testBatchID,
					AssignmentCount:  len(testAssignments),
					Assignments:      testAssignments,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					leaderboardevents.LeaderboardBatchTagAssigned,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Service failure and CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.BatchTagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().BatchTagAssignmentRequested(
					gomock.Any(),
					leaderboardevents.BatchTagAssignmentRequestedPayload{
						RequestingUserID: testRequestingUserID,
						BatchID:          testBatchID,
						Assignments:      testAssignments,
					},
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.BatchTagAssignmentFailedPayload{
							RequestingUserID: testRequestingUserID,
							BatchID:          testBatchID,
							Reason:           "internal service error",
						},
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle BatchTagAssignmentRequested event: internal service error",
		},
		{
			name: "Service failure with non-error result",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.BatchTagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().BatchTagAssignmentRequested(
					gomock.Any(),
					leaderboardevents.BatchTagAssignmentRequestedPayload{
						RequestingUserID: testRequestingUserID,
						BatchID:          testBatchID,
						Assignments:      testAssignments,
					},
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.BatchTagAssignmentFailedPayload{
							RequestingUserID: testRequestingUserID,
							BatchID:          testBatchID,
							Reason:           "non-error failure",
						},
					},
					nil,
				)

				failureResultPayload := &leaderboardevents.BatchTagAssignmentFailedPayload{
					RequestingUserID: testRequestingUserID,
					BatchID:          testBatchID,
					Reason:           "non-error failure",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					leaderboardevents.LeaderboardBatchTagAssignmentFailed,
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
						*out.(*leaderboardevents.BatchTagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().BatchTagAssignmentRequested(
					gomock.Any(),
					leaderboardevents.BatchTagAssignmentRequestedPayload{
						RequestingUserID: testRequestingUserID,
						BatchID:          testBatchID,
						Assignments:      testAssignments,
					},
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.BatchTagAssignmentFailedPayload{
							RequestingUserID: testRequestingUserID,
							BatchID:          testBatchID,
							Reason:           "internal service error",
						},
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle BatchTagAssignmentRequested event: internal service error",
		},
		{
			name: "Unknown result from BatchTagAssignmentRequested",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.BatchTagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().BatchTagAssignmentRequested(
					gomock.Any(),
					leaderboardevents.BatchTagAssignmentRequestedPayload{
						RequestingUserID: testRequestingUserID,
						BatchID:          testBatchID,
						Assignments:      testAssignments,
					},
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
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload type: expected BatchTagAssignmentRequestedPayload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload type: expected BatchTagAssignmentRequestedPayload",
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

			got, err := h.HandleBatchTagAssignmentRequested(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleBatchTagAssignmentRequested() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleBatchTagAssignmentRequested() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleBatchTagAssignmentRequested() = %v, want %v", got, tt.want)
			}
		})
	}
}
