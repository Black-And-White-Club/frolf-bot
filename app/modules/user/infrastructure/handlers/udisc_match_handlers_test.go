package userhandlers

import (
	"encoding/json"
	"errors"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	utilmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	usermocks "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleScorecardParsed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testImportID := "import-123"
	testGuildID := sharedtypes.GuildID("guild-123")
	testRoundID := sharedtypes.RoundID(uuid.New())

	createParsedScorecardMessage := func() *message.Message {
		payload := &roundevents.ParsedScorecardPayload{
			ImportID: testImportID,
			GuildID:  testGuildID,
			RoundID:  testRoundID,
		}
		payloadBytes, _ := json.Marshal(payload)
		return message.NewMessage("test-id", payloadBytes)
	}

	mockUserService := usermocks.NewMockService(ctrl)
	mockHelpers := utilmocks.NewMockHelpers(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	tests := []struct {
		name      string
		msg       *message.Message
		mockSetup func()
		want      []*message.Message
		wantErr   bool
	}{
		{
			name: "Match confirmation required",
			msg:  createParsedScorecardMessage(),
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParsedScorecardPayload) = roundevents.ParsedScorecardPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
						}
						return nil
					},
				)
				mockUserService.EXPECT().MatchParsedScorecard(gomock.Any(), gomock.Any()).
					Return(userservice.UserOperationResult{
						Success: &userevents.UDiscMatchConfirmationRequiredPayloadV1{
							ImportID: testImportID,
						},
					}, nil)
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), userevents.UDiscMatchConfirmationRequiredV1).
					Return(message.NewMessage("out-id", []byte{}), nil)
			},
			want:    []*message.Message{message.NewMessage("out-id", []byte{})},
			wantErr: false,
		},
		{
			name: "Match confirmed",
			msg:  createParsedScorecardMessage(),
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParsedScorecardPayload) = roundevents.ParsedScorecardPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
						}
						return nil
					},
				)
				mockUserService.EXPECT().MatchParsedScorecard(gomock.Any(), gomock.Any()).
					Return(userservice.UserOperationResult{
						Success: &userevents.UDiscMatchConfirmedPayloadV1{
							ImportID: testImportID,
						},
					}, nil)
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), userevents.UDiscMatchConfirmedV1).
					Return(message.NewMessage("out-id", []byte{}), nil)
			},
			want:    []*message.Message{message.NewMessage("out-id", []byte{})},
			wantErr: false,
		},
		{
			name: "Service error",
			msg:  createParsedScorecardMessage(),
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParsedScorecardPayload) = roundevents.ParsedScorecardPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
						}
						return nil
					},
				)
				mockUserService.EXPECT().MatchParsedScorecard(gomock.Any(), gomock.Any()).
					Return(userservice.UserOperationResult{}, errors.New("service error"))
			},
			wantErr: true,
		},
		{
			name: "Failure result",
			msg:  createParsedScorecardMessage(),
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParsedScorecardPayload) = roundevents.ParsedScorecardPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
						}
						return nil
					},
				)
				mockUserService.EXPECT().MatchParsedScorecard(gomock.Any(), gomock.Any()).
					Return(userservice.UserOperationResult{
						Failure: errors.New("some failure"),
					}, nil)
			},
			wantErr: true,
		},
		{
			name: "Unexpected success type",
			msg:  createParsedScorecardMessage(),
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParsedScorecardPayload) = roundevents.ParsedScorecardPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
						}
						return nil
					},
				)
				mockUserService.EXPECT().MatchParsedScorecard(gomock.Any(), gomock.Any()).
					Return(userservice.UserOperationResult{
						Success: "unexpected type",
					}, nil)
			},
			wantErr: true,
		},
		{
			name: "Failure to create confirmation required message",
			msg:  createParsedScorecardMessage(),
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParsedScorecardPayload) = roundevents.ParsedScorecardPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
						}
						return nil
					},
				)
				mockUserService.EXPECT().MatchParsedScorecard(gomock.Any(), gomock.Any()).
					Return(userservice.UserOperationResult{
						Success: &userevents.UDiscMatchConfirmationRequiredPayloadV1{
							ImportID: testImportID,
						},
					}, nil)
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), userevents.UDiscMatchConfirmationRequiredV1).
					Return(nil, errors.New("create message error"))
			},
			wantErr: true,
		},
		{
			name: "Failure to create match confirmed message",
			msg:  createParsedScorecardMessage(),
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.ParsedScorecardPayload) = roundevents.ParsedScorecardPayload{
							ImportID: testImportID,
							GuildID:  testGuildID,
							RoundID:  testRoundID,
						}
						return nil
					},
				)
				mockUserService.EXPECT().MatchParsedScorecard(gomock.Any(), gomock.Any()).
					Return(userservice.UserOperationResult{
						Success: &userevents.UDiscMatchConfirmedPayloadV1{
							ImportID: testImportID,
						},
					}, nil)
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), userevents.UDiscMatchConfirmedV1).
					Return(nil, errors.New("create message error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			h := NewUserHandlers(mockUserService, logger, tracer, mockHelpers, metrics)
			got, err := h.HandleScorecardParsed(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScorecardParsed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != len(tt.want) {
				t.Errorf("HandleScorecardParsed() got = %v, want %v", got, tt.want)
			}
		})
	}
}
