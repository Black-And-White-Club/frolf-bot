package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
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

func TestRoundHandlers_HandleTagNumberFound(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testTagNumber := sharedtypes.TagNumber(42)
	testDiscordMessageID := "12345"
	testOriginalResponse := roundtypes.ResponseAccept
	testOriginalJoinedLate := true // Example value

	// Define the expected ParticipantJoinedPayload once for consistency
	expectedParticipantJoinedPayload := &roundevents.ParticipantJoinedPayload{
		RoundID:               testRoundID,
		AcceptedParticipants:  []roundtypes.Participant{{UserID: testUserID, Response: roundtypes.ResponseAccept, TagNumber: &testTagNumber, Score: nil}},
		DeclinedParticipants:  []roundtypes.Participant{},
		TentativeParticipants: []roundtypes.Participant{},
		EventMessageID:        testDiscordMessageID,
		JoinedLate:            nil,
	}

	// Correct testPayload type to match what HandleTagNumberFound expects
	testPayload := &sharedevents.RoundTagLookupResultPayload{
		RoundID:            testRoundID,
		UserID:             testUserID,
		TagNumber:          &testTagNumber,
		OriginalResponse:   testOriginalResponse,
		OriginalJoinedLate: &testOriginalJoinedLate,
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
			name: "Successfully handle TagNumberFound",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						// Correct type assertion for the out parameter
						*out.(*sharedevents.RoundTagLookupResultPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateParticipantStatus(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:    testRoundID,
						UserID:     testUserID,
						Response:   testOriginalResponse,
						TagNumber:  &testTagNumber,
						JoinedLate: &testOriginalJoinedLate,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: expectedParticipantJoinedPayload, // Use the consistently defined payload
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					expectedParticipantJoinedPayload, // Use the consistently defined payload
					roundevents.RoundParticipantJoined,
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
			name: "Service failure in handleParticipantUpdate",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.RoundTagLookupResultPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateParticipantStatus(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:    testRoundID,
						UserID:     testUserID,
						Response:   testOriginalResponse,
						TagNumber:  &testTagNumber,
						JoinedLate: &testOriginalJoinedLate,
					},
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "UpdateParticipantStatus service failed in helper: internal service error", // Corrected error message
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.RoundTagLookupResultPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateParticipantStatus(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:    testRoundID,
						UserID:     testUserID,
						Response:   testOriginalResponse,
						TagNumber:  &testTagNumber,
						JoinedLate: &testOriginalJoinedLate,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: expectedParticipantJoinedPayload, // Use the consistently defined payload
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					expectedParticipantJoinedPayload, // This is now expected to be `expectedParticipantJoinedPayload`
					roundevents.RoundParticipantJoined,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Unknown result from UpdateParticipantStatus",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.RoundTagLookupResultPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateParticipantStatus(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:    testRoundID,
						UserID:     testUserID,
						Response:   testOriginalResponse,
						TagNumber:  &testTagNumber,
						JoinedLate: &testOriginalJoinedLate,
					},
				).Return(
					roundservice.RoundOperationResult{}, // Return empty result
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "UpdateParticipantStatus service returned unexpected nil result in helper", // Corrected error message
		},
		{
			name: "Invalid payload type",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload type: expected RoundTagLookupResultPayload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload type: expected RoundTagLookupResultPayload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Re-initialize mocks for each subtest to ensure isolation
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

			got, err := h.HandleTagNumberFound(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagNumberFound() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleTagNumberFound() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleTagNumberFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleTagNumberNotFound(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testDiscordMessageID := "12345"
	testOriginalResponse := roundtypes.ResponseAccept // Assuming a default response for not found
	testOriginalJoinedLate := false                   // Assuming a default joined late for not found

	// Define the expected ParticipantJoinedPayload once for consistency
	expectedParticipantJoinedPayload := &roundevents.ParticipantJoinedPayload{
		RoundID:               testRoundID,
		AcceptedParticipants:  []roundtypes.Participant{{UserID: testUserID, Response: testOriginalResponse, TagNumber: nil, Score: nil}},
		DeclinedParticipants:  []roundtypes.Participant{},
		TentativeParticipants: []roundtypes.Participant{},
		EventMessageID:        testDiscordMessageID,
		JoinedLate:            nil, // Or &testOriginalJoinedLate if the service returns it
	}

	// Correct testPayload type to match what HandleTagNumberNotFound expects
	testPayload := &sharedevents.RoundTagLookupResultPayload{
		RoundID:            testRoundID,
		UserID:             testUserID,
		Found:              false, // Explicitly set to false for not found
		Error:              "tag not found",
		OriginalResponse:   testOriginalResponse,
		OriginalJoinedLate: &testOriginalJoinedLate,
	}

	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockSetup      func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers)
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle TagNumberNotFound",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						// Correct type assertion for the out parameter
						*out.(*sharedevents.RoundTagLookupResultPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateParticipantStatus(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:    testRoundID,
						UserID:     testUserID,
						Response:   testOriginalResponse,
						TagNumber:  nil, // Tag number is nil for not found
						JoinedLate: &testOriginalJoinedLate,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: expectedParticipantJoinedPayload, // Use the consistently defined payload
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					expectedParticipantJoinedPayload, // Use the consistently defined payload
					roundevents.RoundParticipantJoined,
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
			name: "Service failure in handleParticipantUpdate",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.RoundTagLookupResultPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateParticipantStatus(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:    testRoundID,
						UserID:     testUserID,
						Response:   testOriginalResponse,
						TagNumber:  nil,
						JoinedLate: &testOriginalJoinedLate,
					},
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "UpdateParticipantStatus service failed in helper: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.RoundTagLookupResultPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateParticipantStatus(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:    testRoundID,
						UserID:     testUserID,
						Response:   testOriginalResponse,
						TagNumber:  nil,
						JoinedLate: &testOriginalJoinedLate,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: expectedParticipantJoinedPayload,
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					expectedParticipantJoinedPayload,
					roundevents.RoundParticipantJoined,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Unknown result from UpdateParticipantStatus",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*sharedevents.RoundTagLookupResultPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateParticipantStatus(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:    testRoundID,
						UserID:     testUserID,
						Response:   testOriginalResponse,
						TagNumber:  nil,
						JoinedLate: &testOriginalJoinedLate,
					},
				).Return(
					roundservice.RoundOperationResult{}, // Return empty result
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "UpdateParticipantStatus service returned unexpected nil result in helper",
		},
		{
			name: "Invalid payload type",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload type: expected RoundTagLookupResultPayload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload type: expected RoundTagLookupResultPayload",
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

			got, err := h.HandleTagNumberNotFound(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagNumberNotFound() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleTagNumberNotFound() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleTagNumberNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleParticipantDeclined(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testDiscordMessageID := "12345"

	testPayload := &roundevents.ParticipantDeclinedPayload{
		RoundID: testRoundID,
		UserID:  testUserID,
	}

	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockSetup      func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers)
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle ParticipantDeclined",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantDeclinedPayload) = *testPayload
						return nil
					},
				)

				// Define the expected ParticipantJoinedPayload for the mock service return
				expectedServiceSuccessPayload := &roundevents.ParticipantJoinedPayload{
					RoundID:               testRoundID,
					AcceptedParticipants:  []roundtypes.Participant{},
					DeclinedParticipants:  []roundtypes.Participant{{UserID: testUserID, Response: roundtypes.ResponseDecline, TagNumber: nil, Score: nil}},
					TentativeParticipants: []roundtypes.Participant{},
					EventMessageID:        testDiscordMessageID,
					JoinedLate:            nil,
				}

				mockRoundService.EXPECT().UpdateParticipantStatus(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:    testRoundID,
						UserID:     testUserID,
						Response:   roundtypes.ResponseDecline,
						TagNumber:  nil,
						JoinedLate: nil,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: expectedServiceSuccessPayload,
					},
					nil,
				)

				// Fixed: Use RoundParticipantJoined topic (implementation always uses this for success)
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					expectedServiceSuccessPayload,
					roundevents.RoundParticipantJoined, // Changed from RoundParticipantDeclined
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
			name: "Service failure in UpdateParticipantStatus",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantDeclinedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateParticipantStatus(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:    testRoundID,
						UserID:     testUserID,
						Response:   roundtypes.ResponseDecline,
						TagNumber:  nil,
						JoinedLate: nil,
					},
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "UpdateParticipantStatus service failed in helper: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantDeclinedPayload) = *testPayload
						return nil
					},
				)

				// Define the expected ParticipantJoinedPayload for the mock service return
				expectedServiceSuccessPayload := &roundevents.ParticipantJoinedPayload{
					RoundID:               testRoundID,
					AcceptedParticipants:  []roundtypes.Participant{},
					DeclinedParticipants:  []roundtypes.Participant{{UserID: testUserID, Response: roundtypes.ResponseDecline, TagNumber: nil, Score: nil}},
					TentativeParticipants: []roundtypes.Participant{},
					EventMessageID:        testDiscordMessageID,
					JoinedLate:            nil,
				}

				mockRoundService.EXPECT().UpdateParticipantStatus(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:    testRoundID,
						UserID:     testUserID,
						Response:   roundtypes.ResponseDecline,
						TagNumber:  nil,
						JoinedLate: nil,
					},
				).Return(
					roundservice.RoundOperationResult{
						Success: expectedServiceSuccessPayload,
					},
					nil,
				)

				// Fixed: Use RoundParticipantJoined topic (implementation always uses this for success)
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					expectedServiceSuccessPayload,
					roundevents.RoundParticipantJoined, // Changed from RoundParticipantDeclined
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Unknown result from UpdateParticipantStatus",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParticipantDeclinedPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().UpdateParticipantStatus(
					gomock.Any(),
					roundevents.ParticipantJoinRequestPayload{
						RoundID:    testRoundID,
						UserID:     testUserID,
						Response:   roundtypes.ResponseDecline,
						TagNumber:  nil,
						JoinedLate: nil,
					},
				).Return(
					roundservice.RoundOperationResult{}, // Return empty result
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "UpdateParticipantStatus service returned unexpected nil result in helper",
		},
		{
			name: "Invalid payload type",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload type: expected ParticipantDeclinedPayload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload type: expected ParticipantDeclinedPayload",
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

			got, err := h.HandleParticipantDeclined(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleParticipantDeclined() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleParticipantDeclined() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleParticipantDeclined() = %v, want %v", got, tt.want)
			}
		})
	}
}
