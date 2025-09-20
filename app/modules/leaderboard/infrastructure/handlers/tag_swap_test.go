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
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleTagSwapRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRequestorID := sharedtypes.DiscordID("2468")
	testTargetID := sharedtypes.DiscordID("13579")

	testPayload := &leaderboardevents.TagSwapRequestedPayload{
		RequestorID: testRequestorID,
		TargetID:    testTargetID,
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
			name: "Successfully handle TagSwapRequested",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagSwapRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().TagSwapRequested(
					gomock.Any(),
					gomock.Any(), // guildID
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.TagSwapProcessedPayload{
							RequestorID: testRequestorID,
							TargetID:    testTargetID,
						},
					},
					nil,
				)

				successResultPayload := &leaderboardevents.TagSwapProcessedPayload{
					RequestorID: testRequestorID,
					TargetID:    testTargetID,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					successResultPayload,
					leaderboardevents.TagSwapProcessed,
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
			name: "Service failure in TagSwapRequested",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagSwapRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().TagSwapRequested(
					gomock.Any(),
					gomock.Any(), // guildID
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle TagSwapRequested event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagSwapRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().TagSwapRequested(
					gomock.Any(),
					gomock.Any(), // guildID
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.TagSwapProcessedPayload{
							RequestorID: testRequestorID,
							TargetID:    testTargetID,
						},
					},
					nil,
				)

				successResultPayload := &leaderboardevents.TagSwapProcessedPayload{
					RequestorID: testRequestorID,
					TargetID:    testTargetID,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					successResultPayload,
					leaderboardevents.TagSwapProcessed,
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
						*out.(*leaderboardevents.TagSwapRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().TagSwapRequested(
					gomock.Any(),
					gomock.Any(), // guildID
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.TagSwapFailedPayload{
							RequestorID: testRequestorID,
							TargetID:    testTargetID,
							Reason:      "test reason",
						},
					},
					nil,
				)

				failureResultPayload := &leaderboardevents.TagSwapFailedPayload{
					RequestorID: testRequestorID,
					TargetID:    testTargetID,
					Reason:      "test reason",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					leaderboardevents.TagSwapFailed,
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
						*out.(*leaderboardevents.TagSwapRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().TagSwapRequested(
					gomock.Any(),
					gomock.Any(), // guildID
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.TagSwapFailedPayload{
							RequestorID: testRequestorID,
							TargetID:    testTargetID,
							Reason:      "test reason",
						},
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle TagSwapRequested event: internal service error",
		},
		{
			name: "Unknown result from TagSwapRequested",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagSwapRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().TagSwapRequested(
					gomock.Any(),
					gomock.Any(), // guildID
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

			got, err := h.HandleTagSwapRequested(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagSwapRequested() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleTagSwapRequested() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleTagSwapRequested() = %v, want %v", got, tt.want)
			}
		})
	}
}
