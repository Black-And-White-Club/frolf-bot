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

func TestRoundHandlers_HandleRoundUpdateRequest(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testTitle := roundtypes.Title("Test Round")
	testUserID := sharedtypes.DiscordID("1234567890")

	testPayload := &roundevents.RoundUpdateRequestPayload{
		BaseRoundPayload: roundtypes.BaseRoundPayload{
			RoundID:     testRoundID,
			Title:       testTitle,
			Description: nil,
			Location:    nil,
			StartTime:   nil,
			UserID:      testUserID,
		},
		EventType: nil,
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
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateRoundUpdateRequest(
					gomock.Any(),
					roundevents.RoundUpdateRequestPayload{
						BaseRoundPayload: roundtypes.BaseRoundPayload{
							RoundID:     testRoundID,
							Title:       testTitle,
							Description: nil,
							Location:    nil,
							StartTime:   nil,
							UserID:      testUserID,
						},
						EventType: nil,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundUpdateValidatedPayload{
							RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
								BaseRoundPayload: roundtypes.BaseRoundPayload{
									RoundID:     testRoundID,
									Title:       testTitle,
									Description: nil,
									Location:    nil,
									StartTime:   nil,
									UserID:      testUserID,
								},
								EventType: nil,
							},
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundUpdateValidated,
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
			name: "Service failure in ValidateRoundUpdateRequest",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateRoundUpdateRequest(
					gomock.Any(),
					roundevents.RoundUpdateRequestPayload{
						BaseRoundPayload: roundtypes.BaseRoundPayload{
							RoundID:     testRoundID,
							Title:       testTitle,
							Description: nil,
							Location:    nil,
							StartTime:   nil,
							UserID:      testUserID,
						},
						EventType: nil,
					},
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle RoundUpdateRequest event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateRoundUpdateRequest(
					gomock.Any(),
					roundevents.RoundUpdateRequestPayload{
						BaseRoundPayload: roundtypes.BaseRoundPayload{
							RoundID:     testRoundID,
							Title:       testTitle,
							Description: nil,
							Location:    nil,
							StartTime:   nil,
							UserID:      testUserID,
						},
						EventType: nil,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundUpdateValidatedPayload{
							RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
								BaseRoundPayload: roundtypes.BaseRoundPayload{
									RoundID:     testRoundID,
									Title:       testTitle,
									Description: nil,
									Location:    nil,
									StartTime:   nil,
									UserID:      testUserID,
								},
								EventType: nil,
							},
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundUpdateValidated,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Unknown result from ValidateRoundUpdateRequest",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateRoundUpdateRequest(
					gomock.Any(),
					roundevents.RoundUpdateRequestPayload{
						BaseRoundPayload: roundtypes.BaseRoundPayload{
							RoundID:     testRoundID,
							Title:       testTitle,
							Description: nil,
							Location:    nil,
							StartTime:   nil,
							UserID:      testUserID,
						},
						EventType: nil,
					},
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
			name: "Failure result from ValidateRoundUpdateRequest",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ValidateRoundUpdateRequest(
					gomock.Any(),
					roundevents.RoundUpdateRequestPayload{
						BaseRoundPayload: roundtypes.BaseRoundPayload{
							RoundID:     testRoundID,
							Title:       testTitle,
							Description: nil,
							Location:    nil,
							StartTime:   nil,
							UserID:      testUserID,
						},
						EventType: nil,
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
					roundevents.RoundUpdateError,
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

	testPayload := &roundevents.RoundUpdateValidatedPayload{
		RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
			BaseRoundPayload: roundtypes.BaseRoundPayload{
				RoundID:     testRoundID,
				Title:       testTitle,
				Description: nil,
				Location:    nil,
				StartTime:   nil,
				UserID:      testUserID,
			},
			EventType: nil,
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
						*out.(*roundevents.RoundUpdateValidatedPayload) = roundevents.RoundUpdateValidatedPayload{
							RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
								BaseRoundPayload: roundtypes.BaseRoundPayload{
									RoundID:     testRoundID,
									Title:       testTitle,
									Description: nil,
									Location:    nil,
									StartTime:   nil,
									UserID:      testUserID,
								},
								EventType: nil,
							},
						}
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateRoundEntity(
					gomock.Any(),
					roundevents.RoundUpdateValidatedPayload{
						RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
							BaseRoundPayload: roundtypes.BaseRoundPayload{
								RoundID:     testRoundID,
								Title:       testTitle,
								Description: nil,
								Location:    nil,
								StartTime:   nil,
								UserID:      testUserID,
							},
							EventType: nil,
						},
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundEntityUpdatedPayload{
							Round: roundtypes.Round{
								ID:          testRoundID,
								Title:       testTitle,
								Description: nil,
								Location:    nil,
								StartTime:   nil,
								CreatedBy:   testUserID,
							},
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundUpdated,
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
						*out.(*roundevents.RoundUpdateValidatedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateRoundEntity(
					gomock.Any(),
					roundevents.RoundUpdateValidatedPayload{
						RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
							BaseRoundPayload: roundtypes.BaseRoundPayload{
								RoundID:     testRoundID,
								Title:       testTitle,
								Description: nil,
								Location:    nil,
								StartTime:   nil,
								UserID:      testUserID,
							},
							EventType: nil,
						},
					},
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
						*out.(*roundevents.RoundUpdateValidatedPayload) = roundevents.RoundUpdateValidatedPayload{
							RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
								BaseRoundPayload: roundtypes.BaseRoundPayload{
									RoundID:     testRoundID,
									Title:       testTitle,
									Description: nil,
									Location:    nil,
									StartTime:   nil,
									UserID:      testUserID,
								},
								EventType: nil,
							},
						}
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateRoundEntity(
					gomock.Any(),
					roundevents.RoundUpdateValidatedPayload{
						RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
							BaseRoundPayload: roundtypes.BaseRoundPayload{
								RoundID:     testRoundID,
								Title:       testTitle,
								Description: nil,
								Location:    nil,
								StartTime:   nil,
								UserID:      testUserID,
							},
							EventType: nil,
						},
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundEntityUpdatedPayload{
							Round: roundtypes.Round{
								ID:          testRoundID,
								Title:       testTitle,
								Description: nil,
								Location:    nil,
								StartTime:   nil,
								CreatedBy:   testUserID,
							},
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundUpdated,
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
						*out.(*roundevents.RoundUpdateValidatedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateRoundEntity(
					gomock.Any(),
					roundevents.RoundUpdateValidatedPayload{
						RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
							BaseRoundPayload: roundtypes.BaseRoundPayload{
								RoundID:     testRoundID,
								Title:       testTitle,
								Description: nil,
								Location:    nil,
								StartTime:   nil,
								UserID:      testUserID,
							},
							EventType: nil,
						},
					},
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
						*out.(*roundevents.RoundUpdateValidatedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateRoundEntity(
					gomock.Any(),
					roundevents.RoundUpdateValidatedPayload{
						RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
							BaseRoundPayload: roundtypes.BaseRoundPayload{
								RoundID:     testRoundID,
								Title:       testTitle,
								Description: nil,
								Location:    nil,
								StartTime:   nil,
								UserID:      testUserID,
							},
							EventType: nil,
						},
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
					roundevents.RoundUpdateError,
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

	testPayload := &roundevents.RoundScheduleUpdatePayload{
		RoundID: testRoundID,
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
						*out.(*roundevents.RoundScheduleUpdatePayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateScheduledRoundEvents(
					gomock.Any(),
					roundevents.RoundScheduleUpdatePayload{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundStoredPayload{
							Round: roundtypes.Round{
								ID: testRoundID,
							},
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundScheduleUpdate,
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
			name: "Service failure in UpdateScheduledRoundEvents",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundScheduleUpdatePayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateScheduledRoundEvents(
					gomock.Any(),
					roundevents.RoundScheduleUpdatePayload{
						RoundID: testRoundID,
					},
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
						*out.(*roundevents.RoundScheduleUpdatePayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateScheduledRoundEvents(
					gomock.Any(),
					roundevents.RoundScheduleUpdatePayload{
						RoundID: testRoundID,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundStoredPayload{
							Round: roundtypes.Round{
								ID: testRoundID,
							},
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundScheduleUpdate,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Unknown result from UpdateScheduledRoundEvents",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundScheduleUpdatePayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateScheduledRoundEvents(
					gomock.Any(),
					roundevents.RoundScheduleUpdatePayload{
						RoundID: testRoundID,
					},
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
						*out.(*roundevents.RoundScheduleUpdatePayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateScheduledRoundEvents(
					gomock.Any(),
					roundevents.RoundScheduleUpdatePayload{
						RoundID: testRoundID,
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
					roundevents.RoundUpdateError,
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
