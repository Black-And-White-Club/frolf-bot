package leaderboardhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/leaderboard"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleTagAvailabilityCheckRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUserID := sharedtypes.DiscordID(uuid.New().String())
	testTagNumber := sharedtypes.TagNumber(1)

	testPayload := &leaderboardevents.TagAvailabilityCheckRequestedPayload{
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
			name: "Successfully handle TagAvailabilityCheckRequested",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagAvailabilityCheckRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().CheckTagAvailability(
					gomock.Any(),
					*testPayload,
				).Return(
					&leaderboardevents.TagAvailabilityCheckResultPayload{
						UserID:    testUserID,
						TagNumber: &testTagNumber,
						Available: true,
					},
					nil,
					nil,
				)

				successResultPayload := &leaderboardevents.TagAvailabilityCheckResultPayload{
					UserID:    testUserID,
					TagNumber: &testTagNumber,
					Available: true,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					successResultPayload,
					leaderboardevents.TagAvailable,
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
			name: "Service failure in CheckTagAvailability",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagAvailabilityCheckRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().CheckTagAvailability(
					gomock.Any(),
					*testPayload,
				).Return(
					nil,
					nil,
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle TagAvailabilityCheckRequested event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagAvailabilityCheckRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().CheckTagAvailability(
					gomock.Any(),
					*testPayload,
				).Return(
					&leaderboardevents.TagAvailabilityCheckResultPayload{
						UserID:    testUserID,
						TagNumber: &testTagNumber,
						Available: true,
					},
					nil,
					nil,
				)

				successResultPayload := &leaderboardevents.TagAvailabilityCheckResultPayload{
					UserID:    testUserID,
					TagNumber: &testTagNumber,
					Available: true,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					successResultPayload,
					leaderboardevents.TagAvailable,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Tag is not available",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagAvailabilityCheckRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().CheckTagAvailability(
					gomock.Any(),
					*testPayload,
				).Return(
					&leaderboardevents.TagAvailabilityCheckResultPayload{
						UserID:    testUserID,
						TagNumber: &testTagNumber,
						Available: false,
					},
					nil,
					nil,
				)

				tagNotAvailablePayload := &leaderboardevents.TagUnavailablePayload{
					UserID:    testUserID,
					TagNumber: &testTagNumber,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					tagNotAvailablePayload,
					leaderboardevents.TagUnavailable,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Service failure with failure payload",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagAvailabilityCheckRequestedPayload) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().CheckTagAvailability(
					gomock.Any(),
					*testPayload,
				).Return(
					nil,
					&leaderboardevents.TagAvailabilityCheckFailedPayload{
						UserID:    testUserID,
						TagNumber: &testTagNumber,
						Reason:    "test reason",
					},
					nil,
				)

				failurePayload := &leaderboardevents.TagAvailabilityCheckFailedPayload{
					UserID:    testUserID,
					TagNumber: &testTagNumber,
					Reason:    "test reason",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failurePayload,
					leaderboardevents.TagAvailableCheckFailure,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
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
				helpers:            mockHelpers,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, metrics, tracer, mockHelpers)
				},
			}

			got, err := h.HandleTagAvailabilityCheckRequested(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagAvailabilityCheckRequested() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleTagAvailabilityCheckRequested() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleTagAvailabilityCheckRequested() = %v, want %v", got, tt.want)
			}
		})
	}
}
