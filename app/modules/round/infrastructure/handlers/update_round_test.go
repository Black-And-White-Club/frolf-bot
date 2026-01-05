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

func TestRoundHandlers_HandleRoundUpdateRequest(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testTitle := roundtypes.Title("Test Round")
	testUserID := sharedtypes.DiscordID("1234567890")

	// Updated: Use UpdateRoundRequestedPayload (Discord payload)
	testTimezone := roundtypes.Timezone("America/Chicago")
	testPayload := &roundevents.UpdateRoundRequestedPayloadV1{
		RoundID:   testRoundID,
		UserID:    testUserID,
		Title:     &testTitle,
		Timezone:  &testTimezone,
		ChannelID: "123456789",
		MessageID: "987654321",
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
			name: "Successfully handle RoundUpdateRequest",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				// Updated: Unmarshal to UpdateRoundRequestedPayload
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.UpdateRoundRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				// Updated: Call ValidateAndProcessRoundUpdateWithClock with timeParser and clock
				mockRoundService.EXPECT().ValidateAndProcessRoundUpdateWithClock(
					gomock.Any(),
					*testPayload,
					gomock.Any(), // timeParser
					gomock.Any(), // clock
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundUpdateValidatedPayloadV1{
							RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayloadV1{
								RoundID: testRoundID,
								Title:   testTitle,
								UserID:  testUserID,
							},
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundUpdateValidatedV1,
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
			name: "Service failure in ValidateAndProcessRoundUpdate",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.UpdateRoundRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				// Updated: Method name and parameters
				mockRoundService.EXPECT().ValidateAndProcessRoundUpdateWithClock(
					gomock.Any(),
					*testPayload,
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
			expectedErrMsg: "failed to validate and process round update: internal service error", // Updated error message
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.UpdateRoundRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateAndProcessRoundUpdateWithClock(
					gomock.Any(),
					*testPayload,
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundUpdateValidatedPayloadV1{
							RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayloadV1{
								RoundID: testRoundID,
								Title:   testTitle,
								UserID:  testUserID,
							},
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundUpdateValidatedV1,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Unknown result from ValidateAndProcessRoundUpdate",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.UpdateRoundRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateAndProcessRoundUpdateWithClock(
					gomock.Any(),
					*testPayload,
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{}, // Return empty result
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service",
		},
		{
			name: "Failure result from ValidateAndProcessRoundUpdate",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.UpdateRoundRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateAndProcessRoundUpdateWithClock(
					gomock.Any(),
					*testPayload,
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundUpdateErrorPayloadV1{
							Error: "validation failed",
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundUpdateErrorV1,
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
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, tracer, mockHelpers, metrics)
				},
			}

			got, err := h.HandleRoundUpdateRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundUpdateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundUpdateRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundUpdateRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundUpdateValidated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testTitle := roundtypes.Title("Test Round")
	testUserID := sharedtypes.DiscordID("1234567890")

	// Updated: Use simplified RoundUpdateRequestPayload structure
	testPayload := &roundevents.RoundUpdateValidatedPayloadV1{
		RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayloadV1{
			RoundID: testRoundID,
			Title:   testTitle,
			UserID:  testUserID,
			// Remove BaseRoundPayload wrapper - use direct fields
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
			name: "Successfully handle RoundUpdateValidated",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundUpdateValidatedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateRoundEntity(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundEntityUpdatedPayloadV1{
							Round: roundtypes.Round{
								ID:        testRoundID,
								Title:     testTitle,
								CreatedBy: testUserID,
							},
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundUpdatedV1,
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
			name: "Service failure in UpdateRoundEntity",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundUpdateValidatedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateRoundEntity(
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
			expectedErrMsg: "failed to handle RoundUpdateValidated event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundUpdateValidatedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateRoundEntity(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundEntityUpdatedPayloadV1{
							Round: roundtypes.Round{
								ID:        testRoundID,
								Title:     testTitle,
								CreatedBy: testUserID,
							},
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundUpdatedV1,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Unknown result from UpdateRoundEntity",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundUpdateValidatedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateRoundEntity(
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
			expectedErrMsg: "unexpected result from service",
		},
		{
			name: "Failure result from UpdateRoundEntity",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundUpdateValidatedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateRoundEntity(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundUpdateErrorPayloadV1{
							RoundUpdateRequest: &roundevents.RoundUpdateRequestPayloadV1{
								RoundID: testRoundID,
								Title:   testTitle,
								UserID:  testUserID,
							},
							Error: "update failed",
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundUpdateErrorV1,
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
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, tracer, mockHelpers, metrics)
				},
			}

			got, err := h.HandleRoundUpdateValidated(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundUpdateValidated() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundUpdateValidated() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundUpdateValidated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundScheduleUpdate(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testTitle := roundtypes.Title("Test Round")
	testStartTime := sharedtypes.StartTime(time.Now().UTC())
	testLocation := roundtypes.Location("Test Location")

	// The handler receives RoundEntityUpdatedPayload, not RoundScheduleUpdatePayload
	testPayload := &roundevents.RoundEntityUpdatedPayloadV1{
		Round: roundtypes.Round{
			ID:        testRoundID,
			Title:     testTitle,
			StartTime: &testStartTime,
			Location:  &testLocation,
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
			name: "Successfully handle RoundScheduleUpdate",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundEntityUpdatedPayloadV1) = *testPayload
						return nil
					},
				)

				// The handler converts RoundEntityUpdatedPayload to RoundScheduleUpdatePayload
				expectedSchedulePayload := roundevents.RoundScheduleUpdatePayloadV1{
					RoundID:   testPayload.Round.ID,
					Title:     testPayload.Round.Title,
					StartTime: testPayload.Round.StartTime,
					Location:  testPayload.Round.Location,
				}

				mockRoundService.EXPECT().UpdateScheduledRoundEvents(
					gomock.Any(),
					expectedSchedulePayload,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundScheduleUpdatePayloadV1{
							RoundID:   testRoundID,
							Title:     testTitle,
							StartTime: &testStartTime,
							Location:  &testLocation,
						},
					},
					nil,
				)

				// No CreateResultMessage expectation because the handler returns empty array on success
			},
			msg:     testMsg,
			want:    []*message.Message{}, // Handler returns empty array on success
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
			name: "Service failure in UpdateScheduledRoundEvents",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundEntityUpdatedPayloadV1) = *testPayload
						return nil
					},
				)

				// The handler converts RoundEntityUpdatedPayload to RoundScheduleUpdatePayload
				expectedSchedulePayload := roundevents.RoundScheduleUpdatePayloadV1{
					RoundID:   testPayload.Round.ID,
					Title:     testPayload.Round.Title,
					StartTime: testPayload.Round.StartTime,
					Location:  testPayload.Round.Location,
				}

				mockRoundService.EXPECT().UpdateScheduledRoundEvents(
					gomock.Any(),
					expectedSchedulePayload,
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle RoundScheduleUpdate event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundEntityUpdatedPayloadV1) = *testPayload
						return nil
					},
				)

				// The handler converts RoundEntityUpdatedPayload to RoundScheduleUpdatePayload
				expectedSchedulePayload := roundevents.RoundScheduleUpdatePayloadV1{
					RoundID:   testPayload.Round.ID,
					Title:     testPayload.Round.Title,
					StartTime: testPayload.Round.StartTime,
					Location:  testPayload.Round.Location,
				}

				mockRoundService.EXPECT().UpdateScheduledRoundEvents(
					gomock.Any(),
					expectedSchedulePayload,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundScheduleUpdatePayloadV1{
							RoundID:   testRoundID,
							Title:     testTitle,
							StartTime: &testStartTime,
							Location:  &testLocation,
						},
					},
					nil,
				)

				// No CreateResultMessage expectation because the handler doesn't call it on success
			},
			msg:     testMsg,
			want:    []*message.Message{}, // Handler returns empty array on success
			wantErr: false,                // This should not error because CreateResultMessage is not called
		},
		{
			name: "Unknown result from UpdateScheduledRoundEvents",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundEntityUpdatedPayloadV1) = *testPayload
						return nil
					},
				)

				// The handler converts RoundEntityUpdatedPayload to RoundScheduleUpdatePayload
				expectedSchedulePayload := roundevents.RoundScheduleUpdatePayloadV1{
					RoundID:   testPayload.Round.ID,
					Title:     testPayload.Round.Title,
					StartTime: testPayload.Round.StartTime,
					Location:  testPayload.Round.Location,
				}

				mockRoundService.EXPECT().UpdateScheduledRoundEvents(
					gomock.Any(),
					expectedSchedulePayload,
				).Return(
					roundservice.RoundOperationResult{}, // Return empty result
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service",
		},
		{
			name: "Failure result from UpdateScheduledRoundEvents",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundEntityUpdatedPayloadV1) = *testPayload
						return nil
					},
				)

				// The handler converts RoundEntityUpdatedPayload to RoundScheduleUpdatePayload
				expectedSchedulePayload := roundevents.RoundScheduleUpdatePayloadV1{
					RoundID:   testPayload.Round.ID,
					Title:     testPayload.Round.Title,
					StartTime: testPayload.Round.StartTime,
					Location:  testPayload.Round.Location,
				}

				mockRoundService.EXPECT().UpdateScheduledRoundEvents(
					gomock.Any(),
					expectedSchedulePayload,
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundErrorPayload{
							RoundID: testRoundID,
							Error:   "some error",
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundUpdateErrorV1,
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
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, tracer, mockHelpers, metrics)
				},
			}

			got, err := h.HandleRoundScheduleUpdate(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundScheduleUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("Handle RoundScheduleUpdate() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundScheduleUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}
