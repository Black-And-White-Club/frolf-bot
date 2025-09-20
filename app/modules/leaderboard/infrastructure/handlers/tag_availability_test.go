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
	leaderboardmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
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
			name: "Successfully handle TagAvailabilityCheckRequested - Tag Available",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagAvailabilityCheckRequestedPayload) = *testPayload
						return nil
					},
				)

				successResultPayload := &leaderboardevents.TagAvailabilityCheckResultPayload{
					UserID:    testUserID,
					TagNumber: &testTagNumber,
					Available: true,
				}

				mockLeaderboardService.EXPECT().CheckTagAvailability(
					gomock.Any(),
					gomock.Any(), // guildID
					*testPayload,
				).Return(
					successResultPayload,
					nil,
					nil,
				)

				// Mock for TagAvailable message
				tagAvailableMsg := message.NewMessage("tag-available-id", []byte("tag available"))
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					successResultPayload,
					leaderboardevents.TagAvailable,
				).Return(tagAvailableMsg, nil)

				// Mock for TagAssignmentRequested message
				tagAssignmentMsg := message.NewMessage("tag-assignment-id", []byte("tag assignment"))
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(), // Use gomock.Any() because we can't predict the UUID
					sharedevents.LeaderboardBatchTagAssignmentRequested,
				).DoAndReturn(func(msg *message.Message, payload interface{}, eventType string) (*message.Message, error) {
					// Verify payload type and source
					assignPayload, ok := payload.(*sharedevents.BatchTagAssignmentRequestedPayload)
					if !ok {
						t.Errorf("Expected BatchTagAssignmentRequestedPayload but got %T", payload)
					}

					// Verify the important fields
					if len(assignPayload.Assignments) != 1 ||
						assignPayload.Assignments[0].UserID != testUserID ||
						assignPayload.Assignments[0].TagNumber != testTagNumber {
						t.Errorf("BatchTagAssignmentRequestedPayload has wrong assignments")
					}

					if assignPayload.RequestingUserID != "system" {
						t.Errorf("BatchTagAssignmentRequestedPayload has wrong RequestingUserID")
					}

					return tagAssignmentMsg, nil
				})
			},
			msg: testMsg,
			want: []*message.Message{
				message.NewMessage("tag-available-id", []byte("tag available")),
				message.NewMessage("tag-assignment-id", []byte("tag assignment")),
			},
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
					gomock.Any(), // guildID
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
			name: "Service success but first CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagAvailabilityCheckRequestedPayload) = *testPayload
						return nil
					},
				)

				successResultPayload := &leaderboardevents.TagAvailabilityCheckResultPayload{
					UserID:    testUserID,
					TagNumber: &testTagNumber,
					Available: true,
				}

				mockLeaderboardService.EXPECT().CheckTagAvailability(
					gomock.Any(),
					gomock.Any(), // guildID
					*testPayload,
				).Return(
					successResultPayload,
					nil,
					nil,
				)

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
			name: "Service success but second CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.TagAvailabilityCheckRequestedPayload) = *testPayload
						return nil
					},
				)

				successResultPayload := &leaderboardevents.TagAvailabilityCheckResultPayload{
					UserID:    testUserID,
					TagNumber: &testTagNumber,
					Available: true,
				}

				mockLeaderboardService.EXPECT().CheckTagAvailability(
					gomock.Any(),
					gomock.Any(), // guildID
					*testPayload,
				).Return(
					successResultPayload,
					nil,
					nil,
				)

				tagAvailableMsg := message.NewMessage("tag-available-id", []byte("tag available"))
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					successResultPayload,
					leaderboardevents.TagAvailable,
				).Return(tagAvailableMsg, nil)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					sharedevents.LeaderboardBatchTagAssignmentRequested,
				).Return(nil, fmt.Errorf("failed to create tag assignment message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create tag assignment message",
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
					gomock.Any(), // guildID
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

				tagUnavailableMsg := message.NewMessage("tag-unavailable-id", []byte("tag unavailable"))
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					tagNotAvailablePayload,
					leaderboardevents.TagUnavailable,
				).Return(tagUnavailableMsg, nil)
			},
			msg: testMsg,
			want: []*message.Message{
				message.NewMessage("tag-unavailable-id", []byte("tag unavailable")),
			},
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

				failurePayload := &leaderboardevents.TagAvailabilityCheckFailedPayload{
					UserID:    testUserID,
					TagNumber: &testTagNumber,
					Reason:    "test reason",
				}

				mockLeaderboardService.EXPECT().CheckTagAvailability(
					gomock.Any(),
					gomock.Any(), // guildID
					*testPayload,
				).Return(
					nil,
					failurePayload,
					nil,
				)

				failureMsg := message.NewMessage("failure-id", []byte("failure"))
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failurePayload,
					leaderboardevents.TagAvailableCheckFailure,
				).Return(failureMsg, nil)
			},
			msg: testMsg,
			want: []*message.Message{
				message.NewMessage("failure-id", []byte("failure")),
			},
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

			got, err := h.HandleTagAvailabilityCheckRequested(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagAvailabilityCheckRequested() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleTagAvailabilityCheckRequested() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			// Custom comparison function for messages
			messagesEqual := func(got, want []*message.Message) bool {
				if len(got) != len(want) {
					return false
				}
				for i := range got {
					// Compare message IDs and payloads
					if got[i].UUID != want[i].UUID || !reflect.DeepEqual(got[i].Payload, want[i].Payload) {
						return false
					}
				}
				return true
			}

			if !messagesEqual(got, tt.want) {
				t.Errorf("HandleTagAvailabilityCheckRequested() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
