package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundReminder(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testReminderType := "test-reminder-type"
	testUserIDs := []sharedtypes.DiscordID{"user1", "user2"}

	testPayload := &roundevents.DiscordReminderPayload{
		RoundID:      testRoundID,
		ReminderType: testReminderType,
	}

	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
		mockSetup      func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers)
	}{
		{
			name: "Successfully handle RoundReminder",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.DiscordReminderPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ProcessRoundReminder(
					gomock.Any(),
					roundevents.DiscordReminderPayload{
						RoundID:      testRoundID,
						ReminderType: testReminderType,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.DiscordReminderPayload{ // Changed to DiscordReminderPayload
							RoundID:      testRoundID,
							ReminderType: testReminderType,
							UserIDs:      testUserIDs, // Added UserIDs so len > 0
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateNewMessage(
					gomock.Any(),
					roundevents.DiscordRoundReminder,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Successfully handle RoundReminder with no participants",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.DiscordReminderPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ProcessRoundReminder(
					gomock.Any(),
					roundevents.DiscordReminderPayload{
						RoundID:      testRoundID,
						ReminderType: testReminderType,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.DiscordReminderPayload{ // Changed to DiscordReminderPayload
							RoundID:      testRoundID,
							ReminderType: testReminderType,
							UserIDs:      []sharedtypes.DiscordID{}, // Empty UserIDs
						},
					},
					nil,
				)

				// No CreateResultMessage call expected when UserIDs is empty
			},
			msg:     testMsg,
			want:    []*message.Message{}, // Empty slice, not nil
			wantErr: false,
		},
		{
			name: "Fail to unmarshal payload",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload",
		},
		{
			name: "Service failure in ProcessRoundReminder",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.DiscordReminderPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ProcessRoundReminder(
					gomock.Any(),
					roundevents.DiscordReminderPayload{
						RoundID:      testRoundID,
						ReminderType: testReminderType,
					},
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to process round reminder: internal service error", // Updated to match handler
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.DiscordReminderPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ProcessRoundReminder(
					gomock.Any(),
					roundevents.DiscordReminderPayload{
						RoundID:      testRoundID,
						ReminderType: testReminderType,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.DiscordReminderPayload{ // Changed to DiscordReminderPayload
							RoundID:      testRoundID,
							ReminderType: testReminderType,
							UserIDs:      testUserIDs, // Added UserIDs so len > 0
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateNewMessage(
					gomock.Any(),
					roundevents.DiscordRoundReminder,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create Discord reminder message: failed to create result message", // Updated to match handler
		},
		{
			name: "Unknown result from ProcessRoundReminder",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.DiscordReminderPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ProcessRoundReminder(
					gomock.Any(),
					roundevents.DiscordReminderPayload{
						RoundID:      testRoundID,
						ReminderType: testReminderType,
					},
				).Return(
					roundservice.RoundOperationResult{}, // Return empty result
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "service returned neither success nor failure", // Updated to match handler
		},
		{
			name: "Failure result from ProcessRoundReminder",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.DiscordReminderPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ProcessRoundReminder(
					gomock.Any(),
					roundevents.DiscordReminderPayload{
						RoundID:      testRoundID,
						ReminderType: testReminderType,
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundErrorPayload{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundError,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundService := roundmocks.NewMockService(ctrl)
			mockHelpers := mocks.NewMockHelpers(ctrl)

			tt.mockSetup(mockRoundService, mockHelpers)

			h := &RoundHandlers{
				roundService: mockRoundService,
				logger:       logger,
				tracer:       tracer,
				metrics:      metrics,
				helpers:      mockHelpers,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, metrics, tracer, mockHelpers)
				},
			}

			got, err := h.HandleRoundReminder(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundReminder() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundReminder() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundReminder() = %v, want %v", got, tt.want)
			}
		})
	}
}
