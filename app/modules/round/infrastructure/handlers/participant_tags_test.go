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
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleScheduledRoundTagUpdate(t *testing.T) {
	// Corrected type for testChangedTags to match expected payload type
	// Use pointers to TagNumber values
	tn1 := sharedtypes.TagNumber(1)
	tn2 := sharedtypes.TagNumber(13)
	testChangedTags := map[sharedtypes.DiscordID]*sharedtypes.TagNumber{
		sharedtypes.DiscordID("user1"): &tn1, // Take address of the TagNumber value
		sharedtypes.DiscordID("user2"): &tn2, // Take address of the TagNumber value
	}
	testRoundID := sharedtypes.RoundID(uuid.New())
	// Corrected type for testScheduledRoundID

	testPayload := &roundevents.ScheduledRoundTagUpdatePayloadV1{
		ChangedTags: testChangedTags,
	}

	// Create the correct payload type that the handler expects
	tagsUpdatedPayload := &roundevents.TagsUpdatedForScheduledRoundsPayloadV1{
		UpdatedRounds: []roundevents.RoundUpdateInfoV1{
			{
				RoundID:             testRoundID,
				Title:               roundtypes.Title("Test Round"),
				EventMessageID:      "some-event-message-id",
				UpdatedParticipants: []roundtypes.Participant{},
				ParticipantsChanged: 2,
			},
		},
		Summary: roundevents.UpdateSummaryV1{
			TotalRoundsProcessed: 1,
			RoundsUpdated:        1,
			ParticipantsUpdated:  2,
		},
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
			name: "Successfully handle ScheduledRoundTagUpdate",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						// The handler expects a map[string]interface{}, not a structured payload
						tagUpdateMap := out.(*map[string]interface{})

						// Convert the structured testPayload to the map format that the handler expects
						changedTagsMap := make(map[string]interface{})
						for userID, tagNumber := range testPayload.ChangedTags {
							if tagNumber != nil {
								changedTagsMap[string(userID)] = int(*tagNumber)
							}
						}

						*tagUpdateMap = map[string]interface{}{
							"source":       "test_source",
							"batch_id":     "test_batch_id",
							"changed_tags": changedTagsMap,
						}
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateScheduledRoundsWithNewTags(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Success: tagsUpdatedPayload,
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					tagsUpdatedPayload,
					roundevents.TagsUpdatedForScheduledRoundsV1,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
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
			name: "Service failure in UpdateScheduledRoundsWithNewTags",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						// The handler expects a map[string]interface{}, not a structured payload
						tagUpdateMap := out.(*map[string]interface{})

						// Convert the structured testPayload to the map format that the handler expects
						changedTagsMap := make(map[string]interface{})
						for userID, tagNumber := range testPayload.ChangedTags {
							if tagNumber != nil {
								changedTagsMap[string(userID)] = int(*tagNumber)
							}
						}

						*tagUpdateMap = map[string]interface{}{
							"source":       "test_source",
							"batch_id":     "test_batch_id",
							"changed_tags": changedTagsMap,
						}
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateScheduledRoundsWithNewTags(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle ScheduledRoundTagUpdate event: internal service error",
		},
		{
			name: "Service returns failure payload",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						// The handler expects a map[string]interface{}, not a structured payload
						tagUpdateMap := out.(*map[string]interface{})

						// Convert the structured testPayload to the map format that the handler expects
						changedTagsMap := make(map[string]interface{})
						for userID, tagNumber := range testPayload.ChangedTags {
							if tagNumber != nil {
								changedTagsMap[string(userID)] = int(*tagNumber)
							}
						}

						*tagUpdateMap = map[string]interface{}{
							"source":       "test_source",
							"batch_id":     "test_batch_id",
							"changed_tags": changedTagsMap,
						}
						return nil
					},
				)

				failurePayload := &roundevents.RoundErrorPayload{
					RoundID: testRoundID,
					Error:   "some error",
				}
				mockRoundService.EXPECT().UpdateScheduledRoundsWithNewTags(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Failure: failurePayload,
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failurePayload,
					roundevents.RoundUpdateError,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						// The handler expects a map[string]interface{}, not a structured payload
						tagUpdateMap := out.(*map[string]interface{})

						// Convert the structured testPayload to the map format that the handler expects
						changedTagsMap := make(map[string]interface{})
						for userID, tagNumber := range testPayload.ChangedTags {
							if tagNumber != nil {
								changedTagsMap[string(userID)] = int(*tagNumber)
							}
						}

						*tagUpdateMap = map[string]interface{}{
							"source":       "test_source",
							"batch_id":     "test_batch_id",
							"changed_tags": changedTagsMap,
						}
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateScheduledRoundsWithNewTags(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Success: tagsUpdatedPayload,
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					tagsUpdatedPayload,
					roundevents.TagsUpdatedForScheduledRoundsV1,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Unknown result from UpdateScheduledRoundsWithNewTags",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						// The handler expects a map[string]interface{}, not a structured payload
						tagUpdateMap := out.(*map[string]interface{})

						// Convert the structured testPayload to the map format that the handler expects
						changedTagsMap := make(map[string]interface{})
						for userID, tagNumber := range testPayload.ChangedTags {
							if tagNumber != nil {
								changedTagsMap[string(userID)] = int(*tagNumber)
							}
						}

						*tagUpdateMap = map[string]interface{}{
							"source":       "test_source",
							"batch_id":     "test_batch_id",
							"changed_tags": changedTagsMap,
						}
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateScheduledRoundsWithNewTags(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{}, // Return empty result
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "service returned neither success nor failure", // Fix: Match the actual error message from handler
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
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, tracer, mockHelpers, metrics)
				},
			}

			got, err := h.HandleScheduledRoundTagUpdate(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScheduledRoundTagUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleScheduledRoundTagUpdate() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleScheduledRoundTagUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}
