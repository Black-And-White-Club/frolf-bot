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

func scorePtr(s sharedtypes.Score) *sharedtypes.Score {
	return &s
}

func TestRoundHandlers_HandleAllScoresSubmitted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRoundID := sharedtypes.RoundID(uuid.New())
	testStartTime := sharedtypes.StartTime(time.Now())
	testTitle := roundtypes.Title("Test Round")
	testLocation := roundtypes.Location("Test Location")

	// Create a more complete test payload that matches what the implementation expects
	testPayload := &roundevents.AllScoresSubmittedPayloadV1{
		RoundID:        testRoundID,
		EventMessageID: "test-message-id",
		RoundData: roundtypes.Round{
			ID:             testRoundID,
			Title:          testTitle,
			Location:       &testLocation,
			StartTime:      &testStartTime,
			EventMessageID: "test-message-id",
			Participants: []roundtypes.Participant{
				{
					UserID:   sharedtypes.DiscordID("user1"),
					Response: roundtypes.ResponseAccept,
					Score:    scorePtr(sharedtypes.Score(60)),
				},
				{
					UserID:   sharedtypes.DiscordID("user2"),
					Response: roundtypes.ResponseAccept,
					Score:    scorePtr(sharedtypes.Score(65)),
				},
			},
		},
		Participants: []roundtypes.Participant{
			{
				UserID:   sharedtypes.DiscordID("user1"),
				Response: roundtypes.ResponseAccept,
				Score:    scorePtr(sharedtypes.Score(60)),
			},
			{
				UserID:   sharedtypes.DiscordID("user2"),
				Response: roundtypes.ResponseAccept,
				Score:    scorePtr(sharedtypes.Score(65)),
			},
		},
	}

	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)
	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	// ✅ Define the expected messages at the test level
	discordMsg := message.NewMessage("discord-msg-id", []byte("discord-message"))
	backendMsg := message.NewMessage("backend-msg-id", []byte("backend-message"))

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
			name: "Successfully handle AllScoresSubmitted",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.AllScoresSubmittedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().FinalizeRound(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundFinalizedPayloadV1{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				// ✅ First call: Discord finalization message
				expectedDiscordPayload := &roundevents.RoundFinalizedDiscordPayloadV1{
					RoundID:        testRoundID,
					Title:          testTitle,
					StartTime:      &testStartTime,
					Location:       &testLocation,
					Participants:   testPayload.Participants,
					EventMessageID: "test-message-id",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					expectedDiscordPayload,
					roundevents.RoundFinalizedDiscordV1,
				).Return(discordMsg, nil) // ✅ Use the pre-defined message

				// ✅ Second call: Backend finalization message
				expectedBackendPayload := &roundevents.RoundFinalizedPayloadV1{
					RoundID: testRoundID,
					RoundData: roundtypes.Round{
						ID:             testRoundID,
						Title:          testTitle,
						Description:    nil,
						Location:       &testLocation,
						StartTime:      &testStartTime,
						EventMessageID: "test-message-id",
						CreatedBy:      "",
						State:          "",
						Participants:   testPayload.Participants, // Uses participants from payload
					},
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					expectedBackendPayload,
					roundevents.RoundFinalizedV1,
				).Return(backendMsg, nil) // ✅ Use the pre-defined message
			},
			msg:     testMsg,
			want:    []*message.Message{discordMsg, backendMsg}, // ✅ Now these are in scope
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
			name: "Service failure in FinalizeRound",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.AllScoresSubmittedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().FinalizeRound(
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
			expectedErrMsg: "failed during backend FinalizeRound service call: internal service error",
		},
		{
			name: "Service failure with non-error result",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.AllScoresSubmittedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().FinalizeRound(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundFinalizationErrorPayloadV1{
							RoundID: testRoundID,
							Error:   "non-error failure",
						},
					},
					nil,
				)

				// The implementation now creates a failure message instead of continuing with success flow
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&roundevents.RoundFinalizationErrorPayloadV1{
						RoundID: testRoundID,
						Error:   "non-error failure",
					},
					roundevents.RoundFinalizationErrorV1,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "CreateResultMessage fails for Discord finalization",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.AllScoresSubmittedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().FinalizeRound(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundFinalizedPayloadV1{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundFinalizedDiscordV1,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create RoundFinalized message: failed to create result message",
		},
		{
			name: "CreateResultMessage fails for Backend finalization",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.AllScoresSubmittedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().FinalizeRound(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundFinalizedPayloadV1{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				// First call succeeds (Discord topic)
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundFinalizedDiscordV1,
				).Return(discordMsg, nil)

				// Second call fails (Backend topic)
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundFinalizedV1,
				).Return(nil, fmt.Errorf("failed to create backend message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create RoundFinalized message: failed to create backend message",
		},
		{
			name: "Invalid payload type",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload type: expected AllScoresSubmittedPayload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload type: expected AllScoresSubmittedPayload",
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

			got, err := h.HandleAllScoresSubmitted(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleAllScoresSubmitted() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleAllScoresSubmitted() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleAllScoresSubmitted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundFinalized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRoundID := sharedtypes.RoundID(uuid.New())

	testPayload := &roundevents.RoundFinalizedPayloadV1{
		RoundID: testRoundID,
		RoundData: roundtypes.Round{
			ID: testRoundID,
			Participants: []roundtypes.Participant{
				{
					UserID:   sharedtypes.DiscordID("user1"),
					Response: roundtypes.ResponseAccept,
					Score:    scorePtr(sharedtypes.Score(60)),
				},
				{
					UserID:   sharedtypes.DiscordID("user2"),
					Response: roundtypes.ResponseAccept,
					Score:    scorePtr(sharedtypes.Score(65)),
				},
			},
		},
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
			name: "Successfully handle RoundFinalized",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundFinalizedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().NotifyScoreModule(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.ProcessRoundScoresRequestPayloadV1{
							RoundID: testRoundID,
							Scores: []roundevents.ParticipantScoreV1{
								{UserID: sharedtypes.DiscordID("user1"), Score: sharedtypes.Score(60)},
								{UserID: sharedtypes.DiscordID("user2"), Score: sharedtypes.Score(65)},
							},
						},
					},
					nil,
				)

				expectedPayload := &roundevents.ProcessRoundScoresRequestPayloadV1{
					RoundID: testRoundID,
					Scores: []roundevents.ParticipantScoreV1{
						{UserID: sharedtypes.DiscordID("user1"), Score: sharedtypes.Score(60)},
						{UserID: sharedtypes.DiscordID("user2"), Score: sharedtypes.Score(65)},
					},
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					expectedPayload,
					roundevents.ProcessRoundScoresRequestedV1,
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
			name: "Service failure in NotifyScoreModule",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundFinalizedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().NotifyScoreModule(
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
			expectedErrMsg: "failed to handle RoundFinalized event: internal service error",
		},
		{
			name: "Service failure but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundFinalizedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().NotifyScoreModule(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundFinalizationErrorPayloadV1{
							RoundID: testRoundID,
							Error:   "database connection failed",
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundFinalizationErrorV1,
				).Return(nil, fmt.Errorf("failed to create failure message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create failure message: failed to create failure message",
		},
		{
			name: "Service failure with non-error result",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundFinalizedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().NotifyScoreModule(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundFinalizationErrorPayloadV1{
							RoundID: testRoundID,
							Error:   "no participants with submitted scores found",
						},
					},
					nil,
				)

				failurePayload := &roundevents.RoundFinalizationErrorPayloadV1{
					RoundID: testRoundID,
					Error:   "no participants with submitted scores found",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failurePayload,
					roundevents.RoundFinalizationErrorV1,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Unknown result from NotifyScoreModule",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundFinalizedPayloadV1) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().NotifyScoreModule(
					gomock.Any(),
					*testPayload,
				).Return(
					roundservice.RoundOperationResult{}, // Empty result
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service (neither success nor failure)", // ✅ Fixed to match actual error
		},
		{
			name: "Invalid payload type",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload type: expected RoundFinalizedPayload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload type: expected RoundFinalizedPayload",
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

			got, err := h.HandleRoundFinalized(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundFinalized() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundFinalized() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundFinalized() = %v, want %v", got, tt.want)
			}
		})
	}
}
