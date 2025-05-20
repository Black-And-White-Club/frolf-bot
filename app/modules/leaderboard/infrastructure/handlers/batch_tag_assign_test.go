package leaderboardhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
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

// Helper function to convert between tag assignment types
func convertToLeaderboardTagAssignments(assignments []sharedevents.TagAssignmentInfo) []leaderboardevents.TagAssignmentInfo {
	result := make([]leaderboardevents.TagAssignmentInfo, len(assignments))
	for i, assignment := range assignments {
		result[i] = leaderboardevents.TagAssignmentInfo{
			UserID:    assignment.UserID,
			TagNumber: assignment.TagNumber,
		}
	}
	return result
}

func TestLeaderboardHandlers_HandleBatchTagAssignmentRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRequestingUserID := sharedtypes.DiscordID("12345678901234567")
	testBatchID := uuid.New().String()

	// Use the shared events type since that's what the incoming event would use
	testSharedAssignments := []sharedevents.TagAssignmentInfo{
		{
			UserID:    sharedtypes.DiscordID("12345678901234567"),
			TagNumber: sharedtypes.TagNumber(1),
		},
	}

	// Convert to leaderboard events type for the return value
	testLeaderboardAssignments := convertToLeaderboardTagAssignments(testSharedAssignments)

	testPayload := &sharedevents.BatchTagAssignmentRequestedPayload{
		RequestingUserID: testRequestingUserID,
		BatchID:          testBatchID,
		Assignments:      testSharedAssignments,
	}
	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	// Create a separate message for the failure case
	failureMsg := message.NewMessage("failure-msg-id", []byte("failure message"))

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
						*out.(*sharedevents.BatchTagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().BatchTagAssignmentRequested(
					gomock.Any(),
					sharedevents.BatchTagAssignmentRequestedPayload{
						RequestingUserID: testRequestingUserID,
						BatchID:          testBatchID,
						Assignments:      testSharedAssignments,
					},
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.BatchTagAssignedPayload{
							RequestingUserID: testRequestingUserID,
							BatchID:          testBatchID,
							AssignmentCount:  len(testSharedAssignments),
							Assignments:      testLeaderboardAssignments, // Use the converted assignments
						},
					},
					nil,
				)

				updateResultPayload := &leaderboardevents.BatchTagAssignedPayload{
					RequestingUserID: testRequestingUserID,
					BatchID:          testBatchID,
					AssignmentCount:  len(testSharedAssignments),
					Assignments:      testLeaderboardAssignments, // Use the converted assignments
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
			expectedErrMsg: "transient unmarshal error: invalid payload",
		},
		{
			name: "Service failure in BatchTagAssignmentRequested",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.BatchTagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().BatchTagAssignmentRequested(
					gomock.Any(),
					sharedevents.BatchTagAssignmentRequestedPayload{
						RequestingUserID: testRequestingUserID,
						BatchID:          testBatchID,
						Assignments:      testSharedAssignments,
					},
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					fmt.Errorf("internal service error"),
				)

				// New expectation for creating a failure message
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&leaderboardevents.BatchTagAssignmentFailedPayload{
						RequestingUserID: testRequestingUserID,
						BatchID:          testBatchID,
						Reason:           "internal service error",
					},
					leaderboardevents.LeaderboardBatchTagAssignmentFailed,
				).Return(failureMsg, nil)
			},
			msg:            testMsg,
			want:           []*message.Message{failureMsg},
			wantErr:        false,
			expectedErrMsg: "",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.BatchTagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().BatchTagAssignmentRequested(
					gomock.Any(),
					sharedevents.BatchTagAssignmentRequestedPayload{
						RequestingUserID: testRequestingUserID,
						BatchID:          testBatchID,
						Assignments:      testSharedAssignments,
					},
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.BatchTagAssignedPayload{
							RequestingUserID: testRequestingUserID,
							BatchID:          testBatchID,
							AssignmentCount:  len(testLeaderboardAssignments),
							Assignments:      testLeaderboardAssignments,
						},
					},
					nil,
				)

				updateResultPayload := &leaderboardevents.BatchTagAssignedPayload{
					RequestingUserID: testRequestingUserID,
					BatchID:          testBatchID,
					AssignmentCount:  len(testLeaderboardAssignments),
					Assignments:      testLeaderboardAssignments,
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
			name: "Service failure with error and CreateResultMessage creates failure message",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.BatchTagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().BatchTagAssignmentRequested(
					gomock.Any(),
					sharedevents.BatchTagAssignmentRequestedPayload{
						RequestingUserID: testRequestingUserID,
						BatchID:          testBatchID,
						Assignments:      testSharedAssignments,
					},
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.BatchTagAssignmentFailedPayload{
							RequestingUserID: testRequestingUserID,
							BatchID:          testBatchID,
							Reason:           "custom service error",
						},
					},
					fmt.Errorf("internal service error"),
				)

				// The handler should use the service's custom reason
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&leaderboardevents.BatchTagAssignmentFailedPayload{
						RequestingUserID: testRequestingUserID,
						BatchID:          testBatchID,
						Reason:           "custom service error",
					},
					leaderboardevents.LeaderboardBatchTagAssignmentFailed,
				).Return(failureMsg, nil)
			},
			msg:            testMsg,
			want:           []*message.Message{failureMsg},
			wantErr:        false,
			expectedErrMsg: "",
		},
		{
			name: "Service failure with non-error result",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.BatchTagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().BatchTagAssignmentRequested(
					gomock.Any(),
					sharedevents.BatchTagAssignmentRequestedPayload{
						RequestingUserID: testRequestingUserID,
						BatchID:          testBatchID,
						Assignments:      testSharedAssignments,
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
			name: "Service error and CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.BatchTagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().BatchTagAssignmentRequested(
					gomock.Any(),
					sharedevents.BatchTagAssignmentRequestedPayload{
						RequestingUserID: testRequestingUserID,
						BatchID:          testBatchID,
						Assignments:      testSharedAssignments,
					},
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					fmt.Errorf("internal service error"),
				)

				// CreateResultMessage fails for the failure message
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&leaderboardevents.BatchTagAssignmentFailedPayload{
						RequestingUserID: testRequestingUserID,
						BatchID:          testBatchID,
						Reason:           "internal service error",
					},
					leaderboardevents.LeaderboardBatchTagAssignmentFailed,
				).Return(nil, fmt.Errorf("failed to create failure message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create failure message after service error: failed to create failure message",
		},
		{
			name: "Unknown result from BatchTagAssignmentRequested",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.BatchTagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().BatchTagAssignmentRequested(
					gomock.Any(),
					sharedevents.BatchTagAssignmentRequestedPayload{
						RequestingUserID: testRequestingUserID,
						BatchID:          testBatchID,
						Assignments:      testSharedAssignments,
					},
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service: neither success nor failure payload set and no error",
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
				Helpers:            mockHelpers,
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
