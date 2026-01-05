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
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboardmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleGetLeaderboardRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testMsg := message.NewMessage("test-id", []byte{})

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
			name: "Successfully handle GetLeaderboardRequest",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.GetLeaderboardRequestedPayloadV1) = leaderboardevents.GetLeaderboardRequestedPayloadV1{}
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetLeaderboard(
					gomock.Any(),
					gomock.Any(), // guildID argument required
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.GetLeaderboardResponsePayloadV1{
							Leaderboard: []leaderboardtypes.LeaderboardEntry{},
						},
					},
					nil,
				)

				updateResultPayload := &leaderboardevents.GetLeaderboardResponsePayloadV1{
					Leaderboard: []leaderboardtypes.LeaderboardEntry{},
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					leaderboardevents.GetLeaderboardResponseV1,
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
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "transient unmarshal error: invalid payload",
		},
		{
			name: "Service failure in GetLeaderboard",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.GetLeaderboardRequestedPayloadV1) = leaderboardevents.GetLeaderboardRequestedPayloadV1{}
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetLeaderboard(
					gomock.Any(),
					gomock.Any(), // guildID
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to get leaderboard: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.GetLeaderboardRequestedPayloadV1) = leaderboardevents.GetLeaderboardRequestedPayloadV1{}
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetLeaderboard(
					gomock.Any(),
					gomock.Any(), // guildID
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.GetLeaderboardResponsePayloadV1{
							Leaderboard: []leaderboardtypes.LeaderboardEntry{},
						},
					},
					nil,
				)

				updateResultPayload := &leaderboardevents.GetLeaderboardResponsePayloadV1{
					Leaderboard: []leaderboardtypes.LeaderboardEntry{},
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					leaderboardevents.GetLeaderboardResponseV1,
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
						*out.(*leaderboardevents.GetLeaderboardRequestedPayloadV1) = leaderboardevents.GetLeaderboardRequestedPayloadV1{}
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetLeaderboard(
					gomock.Any(),
					gomock.Any(), // guildID
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.GetLeaderboardFailedPayloadV1{
							Reason: "non-error failure",
						},
					},
					nil,
				)

				failureResultPayload := &leaderboardevents.GetLeaderboardFailedPayloadV1{
					Reason: "non-error failure",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					leaderboardevents.GetLeaderboardFailedV1,
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
						*out.(*leaderboardevents.GetLeaderboardRequestedPayloadV1) = leaderboardevents.GetLeaderboardRequestedPayloadV1{}
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetLeaderboard(
					gomock.Any(),
					gomock.Any(), // guildID
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.GetLeaderboardFailedPayloadV1{
							Reason: "internal service error",
						},
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to get leaderboard: internal service error",
		},
		{
			name: "Unknown result from GetLeaderboard",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.GetLeaderboardRequestedPayloadV1) = leaderboardevents.GetLeaderboardRequestedPayloadV1{}
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetLeaderboard(
					gomock.Any(),
					gomock.Any(), // guildID
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					nil,
				)
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

			got, err := h.HandleGetLeaderboardRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetLeaderboardRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleGetLeaderboardRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleGetLeaderboardRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleRoundGetTagRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUserID := sharedtypes.DiscordID("12345678901234567")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testResponse := roundtypes.ResponseAccept
	testJoinedLate := true

	testPayload := &sharedevents.RoundTagLookupRequestedPayloadV1{
		UserID:     testUserID,
		RoundID:    testRoundID,
		Response:   testResponse,
		JoinedLate: &testJoinedLate,
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
			name: "Successfully handle RoundGetTagRequest - Tag Found",
			mockSetup: func() {
				// Mock UnmarshalPayload to succeed with the test payload
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.RoundTagLookupRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				// Mock the service call to return a successful result (Tag Found)
				testTagNumber := sharedtypes.TagNumber(5) // Example found tag
				mockLeaderboardService.EXPECT().RoundGetTagByUserID(
					gomock.Any(),
					gomock.Any(), // guildID
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &sharedevents.RoundTagLookupResultPayloadV1{
							UserID:    testUserID,
							RoundID:   testRoundID,
							TagNumber: &testTagNumber,
							Found:     true,
						},
					},
					nil, // No system error from service
				)

				// Mock CreateResultMessage for the success case
				successResultPayload := &sharedevents.RoundTagLookupResultPayloadV1{
					UserID:    testUserID,
					RoundID:   testRoundID,
					TagNumber: &testTagNumber,
					Found:     true,
				}
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					successResultPayload,
					sharedevents.RoundTagLookupFoundV1, // Expected event type for Tag Found
				).Return(testMsg, nil) // Return a mock message and no error
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg}, // Expect the mock success message
			wantErr: false,
		},
		{
			name: "Successfully handle RoundGetTagRequest - Tag Not Found",
			mockSetup: func() {
				// Mock UnmarshalPayload to succeed with the test payload
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.RoundTagLookupRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				// Mock the service call to return a successful result (Tag Not Found)
				mockLeaderboardService.EXPECT().RoundGetTagByUserID(
					gomock.Any(),
					gomock.Any(), // guildID
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &sharedevents.RoundTagLookupResultPayloadV1{
							UserID:    testUserID,
							RoundID:   testRoundID,
							TagNumber: nil, // Tag not found
							Found:     false,
						},
					},
					nil, // No system error from service
				)

				// Mock CreateResultMessage for the not found case
				notFoundResultPayload := &sharedevents.RoundTagLookupResultPayloadV1{
					UserID:    testUserID,
					RoundID:   testRoundID,
					TagNumber: nil,
					Found:     false,
				}
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					notFoundResultPayload,
					sharedevents.RoundTagLookupNotFoundV1, // Expected event type for Tag Not Found
				).Return(testMsg, nil) // Return a mock message and no error
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg}, // Expect the mock not found message
			wantErr: false,
		},
		{
			name: "Fail to unmarshal payload",
			mockSetup: func() {
				// Mock UnmarshalPayload to return an error
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload"))
				// No service or CreateResultMessage calls expected
				mockLeaderboardService.EXPECT().RoundGetTagByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			msg:            invalidMsg, // Use an invalid message
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "transient unmarshal error: invalid payload",
		},
		{
			name: "Service returns unexpected system error",
			mockSetup: func() {
				// Mock UnmarshalPayload to succeed
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.RoundTagLookupRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				// Mock the service call to return a system error
				mockLeaderboardService.EXPECT().RoundGetTagByUserID(
					gomock.Any(),
					gomock.Any(), // guildID
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{}, // No Success, Failure, or Error fields set
					fmt.Errorf("internal service error"),            // Return a system error
				)

				// No CreateResultMessage call expected
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed during RoundGetTagByUserID service call: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails (Tag Found)",
			mockSetup: func() {
				// Mock UnmarshalPayload to succeed
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.RoundTagLookupRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				// Mock the service call to return a successful result (Tag Found)
				testTagNumber := sharedtypes.TagNumber(5)
				mockLeaderboardService.EXPECT().RoundGetTagByUserID(
					gomock.Any(),
					gomock.Any(), // guildID
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &sharedevents.RoundTagLookupResultPayloadV1{
							UserID:    testUserID,
							RoundID:   testRoundID,
							TagNumber: &testTagNumber,
							Found:     true,
						},
					},
					nil,
				)

				// Mock CreateResultMessage to return an error
				successResultPayload := &sharedevents.RoundTagLookupResultPayloadV1{
					UserID:    testUserID,
					RoundID:   testRoundID,
					TagNumber: &testTagNumber,
					Found:     true,
				}
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					successResultPayload,
					sharedevents.RoundTagLookupFoundV1,
				).Return(nil, fmt.Errorf("failed to create result message")) // Return error from CreateResultMessage
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Service success but CreateResultMessage fails (Tag Not Found)",
			mockSetup: func() {
				// Mock UnmarshalPayload to succeed
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.RoundTagLookupRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				// Mock the service call to return a successful result (Tag Not Found)
				mockLeaderboardService.EXPECT().RoundGetTagByUserID(
					gomock.Any(),
					gomock.Any(), // guildID
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &sharedevents.RoundTagLookupResultPayloadV1{
							UserID:    testUserID,
							RoundID:   testRoundID,
							TagNumber: nil,
							Found:     false,
						},
					},
					nil,
				)

				// Mock CreateResultMessage to return an error
				notFoundResultPayload := &sharedevents.RoundTagLookupResultPayloadV1{
					UserID:    testUserID,
					RoundID:   testRoundID,
					TagNumber: nil,
					Found:     false,
				}
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					notFoundResultPayload,
					sharedevents.RoundTagLookupNotFoundV1,
				).Return(nil, fmt.Errorf("failed to create result message")) // Return error from CreateResultMessage
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Service returns business failure (Failure field)",
			mockSetup: func() {
				// Mock UnmarshalPayload to succeed
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.RoundTagLookupRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				// Mock the service call to return a business failure
				mockLeaderboardService.EXPECT().RoundGetTagByUserID(
					gomock.Any(),
					gomock.Any(), // guildID
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &sharedevents.RoundTagLookupFailedPayloadV1{
							UserID:  testUserID,
							RoundID: testRoundID,
							Reason:  "No active round found",
						},
					},
					nil, // No system error from service
				)

				// Mock CreateResultMessage for the failure case
				failureResultPayload := &sharedevents.RoundTagLookupFailedPayloadV1{
					UserID:  testUserID,
					RoundID: testRoundID,
					Reason:  "No active round found",
				}
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					leaderboardevents.GetTagNumberFailedV1, // Expected event type for failure
				).Return(testMsg, nil) // Return a mock failure message and no error
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg}, // Expect the mock failure message
			wantErr: false,
		},
		{
			name: "Service returns business failure but CreateResultMessage fails",
			mockSetup: func() {
				// Mock UnmarshalPayload to succeed
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.RoundTagLookupRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				// Mock the service call to return a business failure
				mockLeaderboardService.EXPECT().RoundGetTagByUserID(
					gomock.Any(),
					gomock.Any(), // guildID
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &sharedevents.RoundTagLookupFailedPayloadV1{
							UserID:  testUserID,
							RoundID: testRoundID,
							Reason:  "No active round found",
						},
					},
					nil,
				)

				// Mock CreateResultMessage to return an error
				failureResultPayload := &sharedevents.RoundTagLookupFailedPayloadV1{
					UserID:  testUserID,
					RoundID: testRoundID,
					Reason:  "No active round found",
				}
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					leaderboardevents.GetTagNumberFailedV1,
				).Return(nil, fmt.Errorf("failed to create failure message")) // Return error from CreateResultMessage
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create failure message: failed to create failure message",
		},
		{
			name: "Service returns system error within result (Error field)",
			mockSetup: func() {
				// Mock UnmarshalPayload to succeed
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.RoundTagLookupRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				// Mock the service call to return a system error within the result struct
				serviceErr := fmt.Errorf("database connection failed")
				mockLeaderboardService.EXPECT().RoundGetTagByUserID(
					gomock.Any(),
					gomock.Any(), // guildID
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Error: serviceErr, // Return system error in the Error field
					},
					nil, // No system error from service call itself
				)

				// Mock CreateResultMessage for the failure case due to system error
				failureResultPayload := sharedevents.RoundTagLookupFailedPayloadV1{
					UserID:  testUserID,
					RoundID: testRoundID,
					Reason:  serviceErr.Error(), // Reason should be the error message
				}
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					leaderboardevents.GetTagNumberFailedV1,
				).Return(testMsg, nil) // Return a mock failure message and no error
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg}, // Expect the mock failure message
			wantErr: false,
		},
		{
			name: "Service returns system error within result but CreateResultMessage fails",
			mockSetup: func() {
				// Mock UnmarshalPayload to succeed
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.RoundTagLookupRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				// Mock the service call to return a system error within the result struct
				serviceErr := fmt.Errorf("database connection failed")
				mockLeaderboardService.EXPECT().RoundGetTagByUserID(
					gomock.Any(),
					gomock.Any(), // guildID
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Error: serviceErr,
					},
					nil,
				)

				// Mock CreateResultMessage to return an error
				failureResultPayload := sharedevents.RoundTagLookupFailedPayloadV1{
					UserID:  testUserID,
					RoundID: testRoundID,
					Reason:  serviceErr.Error(),
				}
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					leaderboardevents.GetTagNumberFailedV1,
				).Return(nil, fmt.Errorf("failed to create failure message")) // Return error from CreateResultMessage
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create failure message: failed to create failure message",
		},
		{
			name: "Service returns unexpected nil result fields",
			mockSetup: func() {
				// Mock UnmarshalPayload to succeed
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.RoundTagLookupRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				// Mock the service call to return a result with no fields set
				mockLeaderboardService.EXPECT().RoundGetTagByUserID(
					gomock.Any(),
					gomock.Any(), // guildID
					*testPayload,
				).Return(
					leaderboardservice.LeaderboardOperationResult{}, // All result fields are nil
					nil, // No system error from service
				)

				// No CreateResultMessage call expected
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "RoundGetTagByUserID service returned unexpected nil result fields",
		},
		{
			name: "Invalid payload type from UnmarshalPayload",
			mockSetup: func() {
				// Mock UnmarshalPayload to return an error indicating wrong type
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload type: expected RoundTagLookupRequestPayload"))
				// No service or CreateResultMessage calls expected
				mockLeaderboardService.EXPECT().RoundGetTagByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			msg:            invalidMsg, // Use an invalid message
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "transient unmarshal error: invalid payload type: expected RoundTagLookupRequestPayload",
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

			// Call the specific handler method being tested
			got, err := h.HandleRoundGetTagRequest(tt.msg)

			// Assert error
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundGetTagRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundGetTagRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			// Assert returned messages (only if no error is expected)
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				// DeepEqual can be tricky with message.Message due to internal fields.
				// A more robust check might compare message UUIDs, topics, and payloads.
				// For simplicity here, we'll do a basic check, but be aware of DeepEqual limitations.
				t.Errorf("HandleRoundGetTagRequest() = %v, want %v", got, tt.want)
			} else if tt.wantErr && got != nil {
				t.Errorf("HandleRoundGetTagRequest() returned messages %v, want nil for error case", got)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleGetTagByUserIDRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRqstUserID := sharedtypes.DiscordID("8675309")
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testTag := sharedtypes.TagNumber(1)

	testPayload := &sharedevents.DiscordTagLookupRequestedPayloadV1{
		UserID: testUserID,
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
			name: "Successfully handle GetTagByUserIDRequest - Tag Found",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.DiscordTagLookupRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				// Update: Now expecting the service to be called with just the UserID
				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					gomock.Any(), // guildID
					testUserID,   // Just passing the UserID rather than payload struct
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &sharedevents.DiscordTagLookupResultPayloadV1{
							TagNumber:        &testTag,
							UserID:           testUserID,
							RequestingUserID: testRqstUserID,
							Found:            true,
						},
					},
					nil,
				)

				successResponsePayload := &sharedevents.DiscordTagLookupResultPayloadV1{
					TagNumber:        &testTag,
					UserID:           testUserID,
					RequestingUserID: testRqstUserID,
					Found:            true,
				}

				// Updated event type to match the refactored handler
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					successResponsePayload,
					sharedevents.DiscordTagLookupSucceededV1,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Successfully handle GetTagByUserIDRequest - Tag Not Found",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.DiscordTagLookupRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					gomock.Any(), // guildID
					testUserID,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &sharedevents.DiscordTagLookupResultPayloadV1{
							TagNumber:        nil,
							UserID:           testUserID,
							RequestingUserID: testRqstUserID,
							Found:            false,
						},
					},
					nil,
				)

				notFoundResponsePayload := &sharedevents.DiscordTagLookupResultPayloadV1{
					TagNumber:        nil,
					UserID:           testUserID,
					RequestingUserID: testRqstUserID,
					Found:            false,
				}

				// Updated event type to match the refactored handler
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					notFoundResponsePayload,
					sharedevents.DiscordTagLookupNotFoundV1,
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
			name: "Service returns unexpected system error",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.DiscordTagLookupRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					gomock.Any(), // guildID
					testUserID,
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed during GetTagByUserID service call: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.DiscordTagLookupRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					gomock.Any(), // guildID
					testUserID,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &sharedevents.DiscordTagLookupResultPayloadV1{
							TagNumber:        &testTag,
							UserID:           testUserID,
							RequestingUserID: testRqstUserID,
							Found:            true,
						},
					},
					nil,
				)

				successResponsePayload := &sharedevents.DiscordTagLookupResultPayloadV1{
					TagNumber:        &testTag,
					UserID:           testUserID,
					RequestingUserID: testRqstUserID,
					Found:            true,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					successResponsePayload,
					sharedevents.DiscordTagLookupSucceededV1,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success/not found message: failed to create result message",
		},
		{
			name: "Service returns business failure (Failure field)",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.DiscordTagLookupRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					gomock.Any(), // guildID
					testUserID,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &sharedevents.DiscordTagLookupFailedPayloadV1{
							UserID: testUserID,
							Reason: "No active leaderboard found",
						},
					},
					nil,
				)

				failureResultPayload := &sharedevents.DiscordTagLookupFailedPayloadV1{
					UserID: testUserID,
					Reason: "No active leaderboard found",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					sharedevents.DiscordTagLookupFailedV1,
				).Return(testMsg, nil)
			},
			msg:            testMsg,
			want:           []*message.Message{testMsg},
			wantErr:        false,
			expectedErrMsg: "",
		},
		{
			name: "Service returns system error within result (Error field)",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.DiscordTagLookupRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					gomock.Any(), // guildID
					testUserID,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Error: fmt.Errorf("database connection failed"),
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					sharedevents.DiscordTagLookupFailedPayloadV1{
						UserID: testUserID,
						Reason: "database connection failed",
					},
					sharedevents.DiscordTagLookupFailedV1,
				).Return(testMsg, nil)
			},
			msg:            testMsg,
			want:           []*message.Message{testMsg},
			wantErr:        false,
			expectedErrMsg: "",
		},
		{
			name: "Service returns unexpected nil result fields",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.DiscordTagLookupRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					gomock.Any(), // guildID
					testUserID,
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "service returned unexpected nil result: GetTagByUserID service returned unexpected nil result fields",
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

			got, err := h.HandleGetTagByUserIDRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetTagByUserIDRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleGetTagByUserIDRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if len(got) != len(tt.want) {
				t.Errorf("HandleGetTagByUserIDRequest() returned %d messages, want %d", len(got), len(tt.want))
			}
		})
	}
}
