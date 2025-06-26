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
	// Fix: Use correct "tag:userID" format
	testSortedParticipantTags := []string{
		"1:12345678901234567", // 1st place
		"2:12345678901234568", // 2nd place
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

				// Expected service request format - convert tag:userID pairs to assignments
				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    sharedtypes.DiscordID("12345678901234567"),
						TagNumber: sharedtypes.TagNumber(1), // 1st place
					},
					{
						UserID:    sharedtypes.DiscordID("12345678901234568"),
						TagNumber: sharedtypes.TagNumber(2), // 2nd place
					},
				}

				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(), // context
					sharedtypes.ServiceUpdateSourceProcessScores, // source - score processing
					expectedRequests,              // requests
					(*sharedtypes.DiscordID)(nil), // requestingUserID - system operation
					uuid.UUID(testRoundID),        // operationID - use roundID
					gomock.Any(),                  // batchID - generated UUID
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.LeaderboardUpdatedPayload{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				updateResultPayload := &leaderboardevents.LeaderboardUpdatedPayload{
					RoundID: testRoundID,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					leaderboardevents.LeaderboardUpdated,
				).Return(testMsg, nil)

				// Expect the tag update message for scheduled rounds
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(), // The tag update payload (map[string]interface{})
					leaderboardevents.TagUpdateForScheduledRounds,
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
			name: "Invalid tag format - missing colon",
			mockSetup: func() {
				invalidPayload := &leaderboardevents.LeaderboardUpdateRequestedPayload{
					RoundID:               testRoundID,
					SortedParticipantTags: []string{"12345678901234567"}, // Missing "tag:" prefix
					Source:                "round",
					UpdateID:              testRoundID.String(),
				}

				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.LeaderboardUpdateRequestedPayload) = *invalidPayload
						return nil
					},
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "invalid tag format: 12345678901234567",
		},
		{
			name: "Invalid tag number format",
			mockSetup: func() {
				invalidPayload := &leaderboardevents.LeaderboardUpdateRequestedPayload{
					RoundID:               testRoundID,
					SortedParticipantTags: []string{"invalid:12345678901234567"}, // Invalid tag number
					Source:                "round",
					UpdateID:              testRoundID.String(),
				}

				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.LeaderboardUpdateRequestedPayload) = *invalidPayload
						return nil
					},
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to parse tag number: strconv.Atoi: parsing \"invalid\": invalid syntax",
		},
		{
			name: "Service failure in ProcessTagAssignments",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.LeaderboardUpdateRequestedPayload) = *testPayload
						return nil
					},
				)

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    sharedtypes.DiscordID("12345678901234567"),
						TagNumber: sharedtypes.TagNumber(1),
					},
					{
						UserID:    sharedtypes.DiscordID("12345678901234568"),
						TagNumber: sharedtypes.TagNumber(2),
					},
				}

				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					sharedtypes.ServiceUpdateSourceProcessScores,
					expectedRequests,
					(*sharedtypes.DiscordID)(nil),
					uuid.UUID(testRoundID),
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to process score-based tag assignments: internal service error",
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

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    sharedtypes.DiscordID("12345678901234567"),
						TagNumber: sharedtypes.TagNumber(1),
					},
					{
						UserID:    sharedtypes.DiscordID("12345678901234568"),
						TagNumber: sharedtypes.TagNumber(2),
					},
				}

				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					sharedtypes.ServiceUpdateSourceProcessScores,
					expectedRequests,
					(*sharedtypes.DiscordID)(nil),
					uuid.UUID(testRoundID),
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.LeaderboardUpdatedPayload{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				updateResultPayload := &leaderboardevents.LeaderboardUpdatedPayload{
					RoundID: testRoundID,
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
			name: "Service failure with custom failure payload",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.LeaderboardUpdateRequestedPayload) = *testPayload
						return nil
					},
				)

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    sharedtypes.DiscordID("12345678901234567"),
						TagNumber: sharedtypes.TagNumber(1),
					},
					{
						UserID:    sharedtypes.DiscordID("12345678901234568"),
						TagNumber: sharedtypes.TagNumber(2),
					},
				}

				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					sharedtypes.ServiceUpdateSourceProcessScores,
					expectedRequests,
					(*sharedtypes.DiscordID)(nil),
					uuid.UUID(testRoundID),
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.LeaderboardUpdateFailedPayload{
							RoundID: testRoundID,
							Reason:  "custom service error",
						},
					},
					nil,
				)

				failurePayload := &leaderboardevents.LeaderboardUpdateFailedPayload{
					RoundID: testRoundID,
					Reason:  "custom service error",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failurePayload,
					leaderboardevents.LeaderboardUpdateFailed,
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
						*out.(*leaderboardevents.LeaderboardUpdateRequestedPayload) = *testPayload
						return nil
					},
				)

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    sharedtypes.DiscordID("12345678901234567"),
						TagNumber: sharedtypes.TagNumber(1),
					},
					{
						UserID:    sharedtypes.DiscordID("12345678901234568"),
						TagNumber: sharedtypes.TagNumber(2),
					},
				}

				// When the service returns an error, the handler returns that error directly
				// It doesn't try to create a failure message
				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					sharedtypes.ServiceUpdateSourceProcessScores,
					expectedRequests,
					(*sharedtypes.DiscordID)(nil),
					uuid.UUID(testRoundID),
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to process score-based tag assignments: internal service error",
		},
		{
			name: "Unknown result from ProcessTagAssignments",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.LeaderboardUpdateRequestedPayload) = *testPayload
						return nil
					},
				)

				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    sharedtypes.DiscordID("12345678901234567"),
						TagNumber: sharedtypes.TagNumber(1),
					},
					{
						UserID:    sharedtypes.DiscordID("12345678901234568"),
						TagNumber: sharedtypes.TagNumber(2),
					},
				}

				mockLeaderboardService.EXPECT().ProcessTagAssignments(
					gomock.Any(),
					sharedtypes.ServiceUpdateSourceProcessScores,
					expectedRequests,
					(*sharedtypes.DiscordID)(nil),
					uuid.UUID(testRoundID),
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected service result: neither success nor failure set",
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

			got, err := h.HandleLeaderboardUpdateRequested(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleLeaderboardUpdateRequested() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && (err == nil || !containsErrorMsg(err.Error(), tt.expectedErrMsg)) {
				t.Errorf("HandleLeaderboardUpdateRequested() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleLeaderboardUpdateRequested() = %v, want %v", got, tt.want)
			}
		})
	}
}
