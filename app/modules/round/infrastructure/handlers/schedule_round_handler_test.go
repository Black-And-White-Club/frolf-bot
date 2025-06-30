package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleDiscordMessageIDUpdated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testStartTime := sharedtypes.StartTime(time.Now())
	testTitle := roundtypes.Title("Test Round")
	testUserID := sharedtypes.DiscordID("1234567890")
	testEventMessageID := "discord-msg-123"

	// Payload for RoundScheduled event (input to HandleDiscordMessageIDUpdated)
	guildID := sharedtypes.GuildID("guild-123")
	testScheduledPayload := &roundevents.RoundScheduledPayload{
		GuildID: guildID,
		BaseRoundPayload: roundtypes.BaseRoundPayload{
			RoundID:     testRoundID,
			Title:       testTitle,
			Description: nil,
			Location:    nil,
			StartTime:   &testStartTime,
			UserID:      testUserID,
		},
		EventMessageID: testEventMessageID,
	}

	// Message for RoundScheduled event (input message to the handler)
	scheduledPayloadBytes, _ := json.Marshal(testScheduledPayload)
	testScheduledMsg := message.NewMessage("input-msg-id", scheduledPayloadBytes)

	// A dummy message for failure results, if needed
	dummyFailureMsg := message.NewMessage("failure-msg-id", []byte("dummy failure"))

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
			name: "Successfully handle DiscordMessageIDUpdated",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundScheduledPayload) = *testScheduledPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ScheduleRoundEvents(
					gomock.Any(),
					guildID,
					*testScheduledPayload, // Expect RoundScheduledPayload
					testEventMessageID,    // Expect EventMessageID
				).Return(
					roundservice.RoundOperationResult{
						Success: testScheduledPayload, // Return success
					},
					nil,
				)
				// No CreateResultMessage mock here, as HandleDiscordMessageIDUpdated returns nil on success
			},
			msg:     testScheduledMsg,
			want:    []*message.Message{}, // Changed from nil to empty slice to match handler
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
			name: "Service failure in ScheduleRoundEvents",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundScheduledPayload) = *testScheduledPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ScheduleRoundEvents(
					gomock.Any(),
					guildID,
					*testScheduledPayload,
					testEventMessageID,
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testScheduledMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to schedule round events: internal service error",
		},
		{
			name: "Unknown result from ScheduleRoundEvents",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundScheduledPayload) = *testScheduledPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ScheduleRoundEvents(
					gomock.Any(),
					guildID,
					*testScheduledPayload,
					testEventMessageID,
				).Return(
					roundservice.RoundOperationResult{}, // Return empty result
					nil,
				)
			},
			msg:            testScheduledMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "service returned neither success nor failure", // Changed to match handler
		},
		{
			name: "Failure result from ScheduleRoundEvents",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundScheduledPayload) = *testScheduledPayload
						return nil
					},
				)

				failurePayload := &roundevents.RoundErrorPayload{
					RoundID: testRoundID,
					// Add other fields if necessary for a complete RoundErrorPayload
				}
				mockRoundService.EXPECT().ScheduleRoundEvents(
					gomock.Any(),
					guildID,
					*testScheduledPayload,
					testEventMessageID,
				).Return(
					roundservice.RoundOperationResult{
						Failure: failurePayload,
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failurePayload, // Expect the failure payload
					roundevents.RoundError,
				).Return(dummyFailureMsg, nil) // Return a dummy failure message
			},
			msg:     testScheduledMsg,
			want:    []*message.Message{dummyFailureMsg},
			wantErr: false,
		},
		{
			name: "Failure result from ScheduleRoundEvents and CreateResultMessage fails",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundScheduledPayload) = *testScheduledPayload
						return nil
					},
				)

				failurePayload := &roundevents.RoundErrorPayload{
					RoundID: testRoundID,
				}
				mockRoundService.EXPECT().ScheduleRoundEvents(
					gomock.Any(),
					guildID,
					*testScheduledPayload,
					testEventMessageID,
				).Return(
					roundservice.RoundOperationResult{
						Failure: failurePayload,
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failurePayload,
					roundevents.RoundError,
				).Return(nil, fmt.Errorf("failed to create result message after failure"))
			},
			msg:            testScheduledMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create failure message: failed to create result message after failure",
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

			got, err := h.HandleDiscordMessageIDUpdated(tt.msg) // Call the correct handler

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleDiscordMessageIDUpdated() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleDiscordMessageIDUpdated() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleDiscordMessageIDUpdated() = %v, want %v", got, tt.want)
			}
		})
	}
}
