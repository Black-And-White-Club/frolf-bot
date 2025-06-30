package leaderboardhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
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

func TestLeaderboardHandlers_HandleBatchTagAssignmentRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRequestingUserID := sharedtypes.DiscordID(uuid.New().String())
	testBatchID := uuid.New().String()
	testUserID1 := sharedtypes.DiscordID(uuid.New().String())
	testUserID2 := sharedtypes.DiscordID(uuid.New().String())
	testTagNumber1 := sharedtypes.TagNumber(1)
	testTagNumber2 := sharedtypes.TagNumber(2)

	testPayload := &sharedevents.BatchTagAssignmentRequestedPayload{
		RequestingUserID: testRequestingUserID,
		BatchID:          testBatchID,
		Assignments: []sharedevents.TagAssignmentInfo{
			{
				UserID:    testUserID1,
				TagNumber: testTagNumber1,
			},
			{
				UserID:    testUserID2,
				TagNumber: testTagNumber2,
			},
		},
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
						*out.(*sharedevents.BatchTagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    testUserID1,
						TagNumber: testTagNumber1,
					},
					{
						UserID:    testUserID2,
						TagNumber: testTagNumber2,
					},
				}

				batchID, _ := uuid.Parse(testBatchID)
				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),            // context
					sharedtypes.GuildID(""), /* or set a test guildID if available */
					testPayload,             // source
					expectedRequests,        // requests
					&testRequestingUserID,   // requestingUserID
					gomock.Any(),            // operationID
					batchID,                 // batchID
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.BatchTagAssignedPayload{
							RequestingUserID: testRequestingUserID,
							BatchID:          testBatchID,
							AssignmentCount:  2,
							Assignments: []leaderboardevents.TagAssignmentInfo{
								{UserID: testUserID1, TagNumber: testTagNumber1},
								{UserID: testUserID2, TagNumber: testTagNumber2},
							},
						},
					},
					nil,
				)

				successResultPayload := &leaderboardevents.BatchTagAssignedPayload{
					RequestingUserID: testRequestingUserID,
					BatchID:          testBatchID,
					AssignmentCount:  2,
					Assignments: []leaderboardevents.TagAssignmentInfo{
						{UserID: testUserID1, TagNumber: testTagNumber1},
						{UserID: testUserID2, TagNumber: testTagNumber2},
					},
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					successResultPayload,
					leaderboardevents.LeaderboardBatchTagAssigned, // Fixed: Use correct event constant
				).Return(testMsg, nil)

				// Expect the tag update message for scheduled rounds
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(), // The tag update payload (map[string]interface{})
					sharedevents.TagUpdateForScheduledRounds,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg, testMsg}, // Expect both success and tag update messages
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
			name: "Invalid batch ID format",
			mockSetup: func() {
				invalidPayload := &sharedevents.BatchTagAssignmentRequestedPayload{
					RequestingUserID: testRequestingUserID,
					BatchID:          "invalid-uuid",
					Assignments: []sharedevents.TagAssignmentInfo{
						{
							UserID:    testUserID1,
							TagNumber: testTagNumber1,
						},
					},
				}

				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.BatchTagAssignmentRequestedPayload) = *invalidPayload
						return nil
					},
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "invalid batch ID format:",
		},
		{
			name: "Service failure in ProcessTagAssignments",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.BatchTagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    testUserID1,
						TagNumber: testTagNumber1,
					},
					{
						UserID:    testUserID2,
						TagNumber: testTagNumber2,
					},
				}

				batchID, _ := uuid.Parse(testBatchID)
				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					sharedtypes.GuildID(""), /* or set a test guildID if available */
					gomock.Any(),
					expectedRequests,
					&testRequestingUserID,
					gomock.Any(),
					batchID,
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to process batch tag assignments: internal service error",
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

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    testUserID1,
						TagNumber: testTagNumber1,
					},
					{
						UserID:    testUserID2,
						TagNumber: testTagNumber2,
					},
				}

				batchID, _ := uuid.Parse(testBatchID)
				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					sharedtypes.GuildID(""), // or set a test guildID if available
					gomock.Any(),            // Allow any payload type for source determination
					expectedRequests,
					&testRequestingUserID,
					gomock.Any(),
					batchID,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.BatchTagAssignedPayload{
							RequestingUserID: testRequestingUserID,
							BatchID:          testBatchID,
							AssignmentCount:  2,
							Assignments: []leaderboardevents.TagAssignmentInfo{
								{UserID: testUserID1, TagNumber: testTagNumber1},
								{UserID: testUserID2, TagNumber: testTagNumber2},
							},
						},
					},
					nil,
				)

				successResultPayload := &leaderboardevents.BatchTagAssignedPayload{
					RequestingUserID: testRequestingUserID,
					BatchID:          testBatchID,
					AssignmentCount:  2,
					Assignments: []leaderboardevents.TagAssignmentInfo{
						{UserID: testUserID1, TagNumber: testTagNumber1},
						{UserID: testUserID2, TagNumber: testTagNumber2},
					},
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					successResultPayload,
					leaderboardevents.LeaderboardBatchTagAssigned,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Service failure with custom failure payload",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.BatchTagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    testUserID1,
						TagNumber: testTagNumber1,
					},
					{
						UserID:    testUserID2,
						TagNumber: testTagNumber2,
					},
				}

				batchID, _ := uuid.Parse(testBatchID)
				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					sharedtypes.GuildID(""), // or set a test guildID if available
					gomock.Any(),            // Allow any payload type for source determination
					expectedRequests,
					&testRequestingUserID,
					gomock.Any(),
					batchID,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.BatchTagAssignmentFailedPayload{
							RequestingUserID: testRequestingUserID,
							BatchID:          testBatchID,
							Reason:           "custom service error",
						},
					},
					nil,
				)

				failurePayload := &leaderboardevents.BatchTagAssignmentFailedPayload{
					RequestingUserID: testRequestingUserID,
					BatchID:          testBatchID,
					Reason:           "custom service error",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failurePayload,
					leaderboardevents.LeaderboardBatchTagAssignmentFailed,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Service failure with error and CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.BatchTagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    testUserID1,
						TagNumber: testTagNumber1,
					},
					{
						UserID:    testUserID2,
						TagNumber: testTagNumber2,
					},
				}

				batchID, _ := uuid.Parse(testBatchID)
				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					sharedtypes.GuildID(""), // or set a test guildID if available
					gomock.Any(),            // Allow any payload type for source determination
					expectedRequests,
					&testRequestingUserID,
					gomock.Any(),
					batchID,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.BatchTagAssignmentFailedPayload{
							RequestingUserID: testRequestingUserID,
							BatchID:          testBatchID,
							Reason:           "internal service error",
						},
					},
					nil,
				)

				failurePayload := &leaderboardevents.BatchTagAssignmentFailedPayload{
					RequestingUserID: testRequestingUserID,
					BatchID:          testBatchID,
					Reason:           "internal service error",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failurePayload,
					leaderboardevents.LeaderboardBatchTagAssignmentFailed,
				).Return(nil, fmt.Errorf("failed to create failure message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create failure message: failed to create failure message",
		},
		{
			name: "Unknown result from ProcessTagAssignments",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.BatchTagAssignmentRequestedPayload) = *testPayload
						return nil
					},
				)

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    testUserID1,
						TagNumber: testTagNumber1,
					},
					{
						UserID:    testUserID2,
						TagNumber: testTagNumber2,
					},
				}

				batchID, _ := uuid.Parse(testBatchID)
				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					sharedtypes.GuildID(""), // or set a test guildID if available
					gomock.Any(),            // Allow any payload type for source determination
					expectedRequests,
					&testRequestingUserID,
					gomock.Any(),
					batchID,
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil, // Expect nil when both Success and Failure are nil
			wantErr:        false,
			expectedErrMsg: "",
		},
		{
			name: "Empty assignments batch",
			mockSetup: func() {
				emptyPayload := &sharedevents.BatchTagAssignmentRequestedPayload{
					RequestingUserID: testRequestingUserID,
					BatchID:          testBatchID,
					Assignments:      []sharedevents.TagAssignmentInfo{},
				}

				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.BatchTagAssignmentRequestedPayload) = *emptyPayload
						return nil
					},
				)

				expectedRequests := []sharedtypes.TagAssignmentRequest{}

				batchID, _ := uuid.Parse(testBatchID)
				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					sharedtypes.GuildID(""), // or set a test guildID if available
					gomock.Any(),            // Allow any payload type for source determination
					expectedRequests,
					&testRequestingUserID,
					gomock.Any(),
					batchID,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.BatchTagAssignedPayload{
							RequestingUserID: testRequestingUserID,
							BatchID:          testBatchID,
							AssignmentCount:  0,
							Assignments:      []leaderboardevents.TagAssignmentInfo{},
						},
					},
					nil,
				)

				successResultPayload := &leaderboardevents.BatchTagAssignedPayload{
					RequestingUserID: testRequestingUserID,
					BatchID:          testBatchID,
					AssignmentCount:  0,
					Assignments:      []leaderboardevents.TagAssignmentInfo{},
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					successResultPayload,
					leaderboardevents.LeaderboardBatchTagAssigned,
				).Return(testMsg, nil)

				// Expect the tag update message for scheduled rounds
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(), // The tag update payload (map[string]interface{})
					sharedevents.TagUpdateForScheduledRounds,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg, testMsg}, // Expect both success and tag update messages
			wantErr: false,
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
			if tt.wantErr && tt.expectedErrMsg != "" && (err == nil || !containsErrorMsg(err.Error(), tt.expectedErrMsg)) {
				t.Errorf("HandleBatchTagAssignmentRequested() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleBatchTagAssignmentRequested() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to check if error message contains expected text
func containsErrorMsg(actual, expected string) bool {
	return len(expected) == 0 || (len(actual) > 0 && strings.Contains(actual, expected))
}
