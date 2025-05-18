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
						*out.(*leaderboardevents.GetLeaderboardRequestPayload) = leaderboardevents.GetLeaderboardRequestPayload{}
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetLeaderboard(
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.GetLeaderboardResponsePayload{
							Leaderboard: []leaderboardtypes.LeaderboardEntry{},
						},
					},
					nil,
				)

				updateResultPayload := &leaderboardevents.GetLeaderboardResponsePayload{
					Leaderboard: []leaderboardtypes.LeaderboardEntry{},
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					leaderboardevents.GetLeaderboardResponse,
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
			expectedErrMsg: "failed to unmarshal payload: invalid payload",
		},
		{
			name: "Service failure in GetLeaderboard",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*leaderboardevents.GetLeaderboardRequestPayload) = leaderboardevents.GetLeaderboardRequestPayload{}
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetLeaderboard(
					gomock.Any(),
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
						*out.(*leaderboardevents.GetLeaderboardRequestPayload) = leaderboardevents.GetLeaderboardRequestPayload{}
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetLeaderboard(
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.GetLeaderboardResponsePayload{
							Leaderboard: []leaderboardtypes.LeaderboardEntry{},
						},
					},
					nil,
				)

				updateResultPayload := &leaderboardevents.GetLeaderboardResponsePayload{
					Leaderboard: []leaderboardtypes.LeaderboardEntry{},
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					leaderboardevents.GetLeaderboardResponse,
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
						*out.(*leaderboardevents.GetLeaderboardRequestPayload) = leaderboardevents.GetLeaderboardRequestPayload{}
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetLeaderboard(
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.GetLeaderboardFailedPayload{
							Reason: "non-error failure",
						},
					},
					nil,
				)

				failureResultPayload := &leaderboardevents.GetLeaderboardFailedPayload{
					Reason: "non-error failure",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					leaderboardevents.GetLeaderboardFailed,
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
						*out.(*leaderboardevents.GetLeaderboardRequestPayload) = leaderboardevents.GetLeaderboardRequestPayload{}
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetLeaderboard(
					gomock.Any(),
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &leaderboardevents.GetLeaderboardFailedPayload{
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
						*out.(*leaderboardevents.GetLeaderboardRequestPayload) = leaderboardevents.GetLeaderboardRequestPayload{}
						return nil
					},
				)

				mockLeaderboardService.EXPECT().GetLeaderboard(
					gomock.Any(),
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

func TestLeaderboardHandlers_HandleGetTagByUserIDRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUserID := sharedtypes.DiscordID("12345678901234567")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testTag := sharedtypes.TagNumber(1)

	testPayload := &sharedevents.DiscordTagLookupRequestPayload{
		UserID: testUserID,
	}
	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	// Mock dependencies
	mockLeaderboardService := leaderboardmocks.NewMockService(ctrl) // Use the correct mock type
	mockHelpers := mocks.NewMockHelpers(ctrl)                       // Use the correct mock type

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
			name: "Successfully handle GetTagByUserIDRequest - Tag Found", // Renamed for clarity
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.DiscordTagLookupRequestPayload) = *testPayload
						return nil
					},
				)

				// Update the service call expectation to use the payload struct
				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					*testPayload, // Pass the payload struct
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.GetTagNumberResponsePayload{
							TagNumber: &testTag,
							UserID:    testUserID,
							RoundID:   testRoundID,
							Found:     true, // Expect Found: true
						},
					},
					nil,
				)

				successResponsePayload := &leaderboardevents.GetTagNumberResponsePayload{
					TagNumber: &testTag,
					UserID:    testUserID,
					RoundID:   testRoundID,
					Found:     true,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					successResponsePayload,
					leaderboardevents.GetTagNumberResponse, // Expect GetTagNumberResponse event
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Successfully handle GetTagByUserIDRequest - Tag Not Found", // New test case for Tag Not Found
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.DiscordTagLookupRequestPayload) = *testPayload
						return nil
					},
				)

				// Update the service call expectation to use the payload struct
				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					*testPayload, // Pass the payload struct
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.GetTagNumberResponsePayload{
							TagNumber: nil, // TagNumber is nil
							UserID:    testUserID,
							RoundID:   testRoundID,
							Found:     false, // Expect Found: false
						},
					},
					nil,
				)

				notFoundResponsePayload := &leaderboardevents.GetTagNumberResponsePayload{
					TagNumber: nil,
					UserID:    testUserID,
					RoundID:   testRoundID,
					Found:     false,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					notFoundResponsePayload,
					leaderboardevents.GetTagByUserIDNotFound, // Expect GetTagByUserIDNotFound event
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
			name: "Service returns unexpected system error", // Renamed for clarity
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.DiscordTagLookupRequestPayload) = *testPayload
						return nil
					},
				)

				// Update the service call expectation to use the payload struct
				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					*testPayload, // Pass the payload struct
				).Return(
					leaderboardservice.LeaderboardOperationResult{}, // Result is empty
					fmt.Errorf("internal service error"),            // Error is returned directly
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed during GetTagByUserID service call: internal service error", // Match the handler's error wrapping
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.DiscordTagLookupRequestPayload) = *testPayload
						return nil
					},
				)

				// Update the service call expectation to use the payload struct
				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					*testPayload, // Pass the payload struct
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: &leaderboardevents.GetTagNumberResponsePayload{
							TagNumber: &testTag,
							UserID:    testUserID,
							RoundID:   testRoundID,
							Found:     true,
						},
					},
					nil,
				)

				successResponsePayload := &leaderboardevents.GetTagNumberResponsePayload{
					TagNumber: &testTag,
					UserID:    testUserID,
					RoundID:   testRoundID,
					Found:     true,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					successResponsePayload,
					leaderboardevents.GetTagNumberResponse,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Service returns business failure (Failure field)", // Renamed for clarity
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.DiscordTagLookupRequestPayload) = *testPayload
						return nil
					},
				)

				// Update the service call expectation to use the payload struct
				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					*testPayload, // Pass the payload struct
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: &sharedevents.DiscordTagLookupByUserIDFailedPayload{
							Reason: "No active leaderboard found", // Match the service's failure reason
						},
					},
					nil,
				)

				failureResultPayload := &sharedevents.DiscordTagLookupByUserIDFailedPayload{
					Reason: "No active leaderboard found",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					leaderboardevents.GetTagNumberFailed,
				).Return(testMsg, nil)
			},
			msg:            testMsg,
			want:           []*message.Message{testMsg},
			wantErr:        false,
			expectedErrMsg: "",
		},
		{
			name: "Service returns system error within result (Error field)", // New test case for Error field
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.DiscordTagLookupRequestPayload) = *testPayload
						return nil
					},
				)

				// Update the service call expectation to use the payload struct
				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					*testPayload, // Pass the payload struct
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Error: fmt.Errorf("database connection failed"), // Error is in the result struct
					},
					nil, // No error returned directly
				)

				failureResultPayload := &sharedevents.DiscordTagLookupByUserIDFailedPayload{
					Reason: "database connection failed", // Reason is the error message
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					leaderboardevents.GetTagNumberFailed,
				).Return(testMsg, nil)
			},
			msg:            testMsg,
			want:           []*message.Message{testMsg},
			wantErr:        false, // Handler returns a message, not an error for this case
			expectedErrMsg: "",
		},
		{
			name: "Service returns unexpected nil result fields", // Renamed for clarity
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.DiscordTagLookupRequestPayload) = *testPayload
						return nil
					},
				)

				// Update the service call expectation to use the payload struct
				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					*testPayload, // Pass the payload struct
				).Return(
					leaderboardservice.LeaderboardOperationResult{}, // Empty result
					nil, // No error returned directly
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "GetTagByUserID service returned unexpected nil result fields", // Match the handler's error
		},
		{
			name: "Service returns success with unexpected payload type", // New test case for unexpected success payload
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.DiscordTagLookupRequestPayload) = *testPayload
						return nil
					},
				)

				// Update the service call expectation to use the payload struct
				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					*testPayload, // Pass the payload struct
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Success: "unexpected string payload", // Return a string instead of the expected struct
					},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected success payload type from GetTagByUserID: expected *leaderboardevents.GetTagNumberResponsePayload, got string", // Match the handler's error
		},
		{
			name: "Service returns failure with unexpected payload type", // New test case for unexpected failure payload
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.DiscordTagLookupRequestPayload) = *testPayload
						return nil
					},
				)

				// Update the service call expectation to use the payload struct
				mockLeaderboardService.EXPECT().GetTagByUserID(
					gomock.Any(),
					*testPayload, // Pass the payload struct
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Failure: "unexpected string payload", // Return a string instead of the expected struct
					},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected failure payload type from GetTagByUserID: expected *sharedevents.DiscordTagLookupByUserIDFailedPayload, got string", // Match the handler's error
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
