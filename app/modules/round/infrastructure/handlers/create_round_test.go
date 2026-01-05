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

func TestRoundHandlers_HandleCreateRoundRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testTitle := roundtypes.Title("Test Round")
	testDescription := roundtypes.Description("This is a test round")
	testLocation := roundtypes.Location("Test Location")
	testStartTime := sharedtypes.StartTime(time.Now())
	testStartTimeString := "2024-01-01T12:00:00Z"
	testCreateRoundID := sharedtypes.RoundID(uuid.New())

	testUserID := sharedtypes.DiscordID("12345678901234567")

	testPayload := &roundevents.CreateRoundRequestedPayloadV1{
		Title:       testTitle,
		Description: testDescription,
		Location:    testLocation,
		StartTime:   testStartTimeString,
		UserID:      testUserID,
	}

	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	// Mock dependencies
	mockRoundService := roundmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockSetup      func()
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle CreateRoundRequest",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.CreateRoundRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateAndProcessRoundWithClock(
					gomock.Any(),
					roundevents.CreateRoundRequestedPayloadV1{
						Title:       testTitle,
						Description: testDescription,
						Location:    testLocation,
						StartTime:   testStartTimeString,
						UserID:      testUserID,
					},
					gomock.Any(), // time parser
					gomock.Any(), // clock
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundCreatedPayloadV1{
							BaseRoundPayload: roundtypes.BaseRoundPayload{
								RoundID:     testCreateRoundID,
								Title:       testTitle,
								Description: &testDescription,
								Location:    &testLocation,
								StartTime:   &testStartTime,
								UserID:      testUserID,
							},
							ChannelID: "test-channel-id",
						},
					},
					nil,
				)

				updateResultPayload := &roundevents.RoundCreatedPayloadV1{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testCreateRoundID,
						Title:       testTitle,
						Description: &testDescription,
						Location:    &testLocation,
						StartTime:   &testStartTime,
						UserID:      testUserID,
					},
					ChannelID: "test-channel-id",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.RoundEntityCreatedV1,
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
			name: "Service failure in ValidateAndProcessRound",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.CreateRoundRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateAndProcessRoundWithClock(
					gomock.Any(),
					roundevents.CreateRoundRequestedPayloadV1{
						Title:       testTitle,
						Description: testDescription,
						Location:    testLocation,
						StartTime:   testStartTimeString,
						UserID:      testUserID,
					},
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle CreateRoundRequest event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.CreateRoundRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateAndProcessRoundWithClock(
					gomock.Any(),
					roundevents.CreateRoundRequestedPayloadV1{
						Title:       testTitle,
						Description: testDescription,
						Location:    testLocation,
						StartTime:   testStartTimeString,
						UserID:      testUserID,
					},
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundCreatedPayloadV1{
							BaseRoundPayload: roundtypes.BaseRoundPayload{
								RoundID:     testCreateRoundID,
								Title:       testTitle,
								Description: &testDescription,
								Location:    &testLocation,
								StartTime:   &testStartTime,
								UserID:      testUserID,
							},
							ChannelID: "test-channel-id",
						},
					},
					nil,
				)

				updateResultPayload := &roundevents.RoundCreatedPayloadV1{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testCreateRoundID,
						Title:       testTitle,
						Description: &testDescription,
						Location:    &testLocation,
						StartTime:   &testStartTime,
						UserID:      testUserID,
					},
					ChannelID: "test-channel-id",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.RoundEntityCreatedV1,
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
						*out.(*roundevents.CreateRoundRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateAndProcessRoundWithClock(
					gomock.Any(),
					roundevents.CreateRoundRequestedPayloadV1{
						Title:       testTitle,
						Description: testDescription,
						Location:    testLocation,
						StartTime:   testStartTimeString,
						UserID:      testUserID,
					},
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundValidationFailedPayloadV1{
							UserID:       testUserID,
							ErrorMessages: []string{"non-error failure"},
						},
					},
					nil,
				)

				failureResultPayload := &roundevents.RoundValidationFailedPayloadV1{
					UserID:       testUserID,
					ErrorMessages: []string{"non-error failure"},
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					roundevents.RoundValidationFailedV1,
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
						*out.(*roundevents.CreateRoundRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateAndProcessRoundWithClock(
					gomock.Any(),
					roundevents.CreateRoundRequestedPayloadV1{
						Title:       testTitle,
						Description: testDescription,
						Location:    testLocation,
						StartTime:   testStartTimeString,
						UserID:      testUserID,
					},
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundValidationFailedPayloadV1{
							UserID:       testUserID,
							ErrorMessages: []string{"internal service error"},
						},
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle CreateRoundRequest event: internal service error",
		},
		{
			name: "Unknown result from ValidateAndProcessRound",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.CreateRoundRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateAndProcessRoundWithClock(
					gomock.Any(),
					roundevents.CreateRoundRequestedPayloadV1{
						Title:       testTitle,
						Description: testDescription,
						Location:    testLocation,
						StartTime:   testStartTimeString,
						UserID:      testUserID,
					},
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service",
		},
		{
			name: "Invalid payload type",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload type: expected CreateRoundRequestedPayload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload type: expected CreateRoundRequestedPayload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

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

			got, err := h.HandleCreateRoundRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleCreateRoundRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleCreateRoundRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleCreateRoundRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundEntityCreated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRoundID := sharedtypes.RoundID(uuid.New())
	testTitle := roundtypes.Title("Test Round")
	testDescription := roundtypes.Description("This is a test round")
	testLocation := roundtypes.Location("Test Location")
	testStartTime := sharedtypes.StartTime(time.Now())
	testUserID := sharedtypes.DiscordID("12345678901234567")

	testRound := roundtypes.Round{
		ID:          testRoundID,
		Title:       testTitle,
		Description: &testDescription,
		Location:    &testLocation,
		StartTime:   &testStartTime,
		CreatedBy:   testUserID,
	}

	guildID := sharedtypes.GuildID("guild-123")
	testPayload := &roundevents.RoundEntityCreatedPayloadV1{
		GuildID:          guildID,
		Round:            testRound,
		DiscordChannelID: "test-channel-id",
		DiscordGuildID:   "test-guild-id",
	}
	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	// Mock dependencies
	mockRoundService := roundmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockSetup      func()
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle RoundEntityCreated",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundEntityCreatedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().StoreRound(
					gomock.Any(),
					guildID,
					roundevents.RoundEntityCreatedPayloadV1{
						GuildID:          guildID,
						Round:            testRound,
						DiscordChannelID: "test-channel-id",
						DiscordGuildID:   "test-guild-id",
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundCreatedPayloadV1{
							BaseRoundPayload: roundtypes.BaseRoundPayload{
								RoundID:     testRoundID,
								Title:       testTitle,
								Description: &testDescription,
								Location:    &testLocation,
								StartTime:   &testStartTime,
								UserID:      testUserID,
							},
							ChannelID: "test-channel-id",
						},
					},
					nil,
				)

				updateResultPayload := &roundevents.RoundCreatedPayloadV1{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testTitle,
						Description: &testDescription,
						Location:    &testLocation,
						StartTime:   &testStartTime,
						UserID:      testUserID,
					},
					ChannelID: "test-channel-id",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.RoundCreatedV1,
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
			name: "Service failure in StoreRound",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundEntityCreatedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().StoreRound(
					gomock.Any(),
					guildID,
					roundevents.RoundEntityCreatedPayloadV1{
						GuildID:          guildID,
						Round:            testRound,
						DiscordChannelID: "test-channel-id",
						DiscordGuildID:   "test-guild-id",
					},
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle RoundEntityCreated event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundEntityCreatedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().StoreRound(
					gomock.Any(),
					guildID,
					roundevents.RoundEntityCreatedPayloadV1{
						GuildID:          guildID,
						Round:            testRound,
						DiscordChannelID: "test-channel-id",
						DiscordGuildID:   "test-guild-id",
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundCreatedPayloadV1{
							BaseRoundPayload: roundtypes.BaseRoundPayload{
								RoundID:     testRoundID,
								Title:       testTitle,
								Description: &testDescription,
								Location:    &testLocation,
								StartTime:   &testStartTime,
								UserID:      testUserID,
							},
							ChannelID: "test-channel-id",
						},
					},
					nil,
				)

				updateResultPayload := &roundevents.RoundCreatedPayloadV1{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       testTitle,
						Description: &testDescription,
						Location:    &testLocation,
						StartTime:   &testStartTime,
						UserID:      testUserID,
					},
					ChannelID: "test-channel-id",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					roundevents.RoundCreatedV1,
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
						*out.(*roundevents.RoundEntityCreatedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().StoreRound(
					gomock.Any(),
					guildID,
					roundevents.RoundEntityCreatedPayloadV1{
						GuildID:          guildID,
						Round:            testRound,
						DiscordChannelID: "test-channel-id",
						DiscordGuildID:   "test-guild-id",
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundCreationFailedPayloadV1{
							ErrorMessage: "non-error failure",
						},
					},
					nil,
				)

				failureResultPayload := &roundevents.RoundCreationFailedPayloadV1{
					ErrorMessage: "non-error failure",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					roundevents.RoundCreationFailedV1,
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
						*out.(*roundevents.RoundEntityCreatedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().StoreRound(
					gomock.Any(),
					guildID,
					roundevents.RoundEntityCreatedPayloadV1{
						GuildID:          guildID,
						Round:            testRound,
						DiscordChannelID: "test-channel-id",
						DiscordGuildID:   "test-guild-id",
					},
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundCreationFailedPayloadV1{
							ErrorMessage: "internal service error",
						},
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle RoundEntityCreated event: internal service error",
		},
		{
			name: "Unknown result from StoreRound",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundEntityCreatedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().StoreRound(
					gomock.Any(),
					guildID,
					roundevents.RoundEntityCreatedPayloadV1{
						GuildID:          guildID,
						Round:            testRound,
						DiscordChannelID: "test-channel-id",
						DiscordGuildID:   "test-guild-id",
					},
				).Return(
					roundservice.RoundOperationResult{},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service",
		},
		{
			name: "Invalid payload type",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload type: expected RoundEntityCreatedPayload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload type: expected RoundEntityCreatedPayload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

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

			got, err := h.HandleRoundEntityCreated(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundEntityCreated() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundEntityCreated() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundEntityCreated() = %v, want %v", got, tt.want)
			}
		})
	}
}
