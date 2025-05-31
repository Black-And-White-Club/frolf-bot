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

func TestLeaderboardHandlers_HandleTagAssignment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUserID := sharedtypes.DiscordID(uuid.New().String())
	testTagNumber := sharedtypes.TagNumber(1)
	testAssignmentID := sharedtypes.RoundID(uuid.New())
	testUpdateID := sharedtypes.RoundID(uuid.New())
	testSource := "manual"
	testReason := "test-reason"

	testPayload := &leaderboardevents.TagAssignmentRequestedPayload{
		UserID:    testUserID,
		TagNumber: &testTagNumber,
		Source:    testSource,
		UpdateID:  testUpdateID,
	}
	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	testUserCreationPayload := &leaderboardevents.TagAssignmentRequestedPayload{
		UserID:    testUserID,
		TagNumber: &testTagNumber,
		Source:    "user_creation",
		UpdateID:  testUpdateID,
	}

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
			name: "Successfully handle TagAssignmentRequested - manual source",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    testUserID,
						TagNumber: testTagNumber,
					},
				}

				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),                          // context
					sharedtypes.ServiceUpdateSourceManual, // source - manual for non-user_creation
					expectedRequests,                      // requests
					(*sharedtypes.DiscordID)(nil),         // requestingUserID - nil for individual assignments
					uuid.UUID(testUpdateID),               // operationID - use UpdateID
					gomock.Any(),                          // batchID - generated UUID
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
			name: "Successfully handle TagAssignmentRequested - user_creation source",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagAssignmentRequestedPayload) = *testUserCreationPayload
						return nil
					},
				)

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    testUserID,
						TagNumber: testTagNumber,
					},
				}

				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(), // context
					sharedtypes.ServiceUpdateSourceCreateUser, // source - user creation
					expectedRequests,              // requests
					(*sharedtypes.DiscordID)(nil), // requestingUserID - nil for individual assignments
					uuid.UUID(testUpdateID),       // operationID - use UpdateID
					gomock.Any(),                  // batchID - generated UUID
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.TagAssignedPayload{
							UserID:       testUserID,
							TagNumber:    &testTagNumber,
							AssignmentID: testAssignmentID,
							Source:       "user_creation",
						},
					},
					nil,
				)

				successResultPayload := &leaderboardevents.TagAssignedPayload{
					UserID:       testUserID,
					TagNumber:    &testTagNumber,
					AssignmentID: testAssignmentID,
					Source:       "user_creation",
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
			name: "Handles tag swap flow",
			mockSetup: func() {
				swapPayload := &leaderboardevents.TagSwapRequestedPayload{
					RequestorID: testUserID,
					TargetID:    "target-user",
				}

				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    testUserID,
						TagNumber: testTagNumber,
					},
				}

				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					sharedtypes.ServiceUpdateSourceManual,
					expectedRequests,
					(*sharedtypes.DiscordID)(nil),
					uuid.UUID(testUpdateID),
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: swapPayload,
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					swapPayload,
					leaderboardevents.TagSwapRequested,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Handles tag swap flow but CreateResultMessage fails",
			mockSetup: func() {
				swapPayload := &leaderboardevents.TagSwapRequestedPayload{
					RequestorID: testUserID,
					TargetID:    "target-user",
				}

				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    testUserID,
						TagNumber: testTagNumber,
					},
				}

				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					sharedtypes.ServiceUpdateSourceManual,
					expectedRequests,
					(*sharedtypes.DiscordID)(nil),
					uuid.UUID(testUpdateID),
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: swapPayload,
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					swapPayload,
					leaderboardevents.TagSwapRequested,
				).Return(nil, fmt.Errorf("failed to create swap message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create tag swap message: failed to create swap message",
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
			name: "Service failure in ProcessTagAssignments",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    testUserID,
						TagNumber: testTagNumber,
					},
				}

				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					sharedtypes.ServiceUpdateSourceManual,
					expectedRequests,
					(*sharedtypes.DiscordID)(nil),
					uuid.UUID(testUpdateID),
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to process tag assignment: internal service error",
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

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    testUserID,
						TagNumber: testTagNumber,
					},
				}

				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					sharedtypes.ServiceUpdateSourceManual,
					expectedRequests,
					(*sharedtypes.DiscordID)(nil),
					uuid.UUID(testUpdateID),
					gomock.Any(),
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

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    testUserID,
						TagNumber: testTagNumber,
					},
				}

				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					sharedtypes.ServiceUpdateSourceManual,
					expectedRequests,
					(*sharedtypes.DiscordID)(nil),
					uuid.UUID(testUpdateID),
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.TagAssignmentFailedPayload{
							UserID:    testUserID,
							TagNumber: &testTagNumber,
							Source:    testSource,
							Reason:    testReason,
						},
					},
					nil,
				)

				failureResultPayload := &leaderboardevents.TagAssignmentFailedPayload{
					UserID:    testUserID,
					TagNumber: &testTagNumber,
					Source:    testSource,
					Reason:    testReason,
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

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    testUserID,
						TagNumber: testTagNumber,
					},
				}

				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					sharedtypes.ServiceUpdateSourceManual,
					expectedRequests,
					(*sharedtypes.DiscordID)(nil),
					uuid.UUID(testUpdateID),
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.TagAssignmentFailedPayload{
							UserID:    testUserID,
							TagNumber: &testTagNumber,
							Source:    testSource,
							Reason:    testReason,
						},
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to process tag assignment: internal service error",
		},
		{
			name: "Unknown result from ProcessTagAssignments",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    testUserID,
						TagNumber: testTagNumber,
					},
				}

				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					sharedtypes.ServiceUpdateSourceManual,
					expectedRequests,
					(*sharedtypes.DiscordID)(nil),
					uuid.UUID(testUpdateID),
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
				Helpers:            mockHelpers,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, metrics, tracer, mockHelpers)
				},
			}

			got, err := h.HandleTagAssignment(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagAssignment() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && (err == nil || !containsErrorMsg(err.Error(), tt.expectedErrMsg)) {
				t.Errorf("HandleTagAssignment() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleTagAssignment() = %v, want %v", got, tt.want)
			}
		})
	}
}
