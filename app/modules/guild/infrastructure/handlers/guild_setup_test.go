package guildhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guildservice "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application"
	guildmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestGuildHandlers_HandleGuildSetup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	setupTime := time.Now().UTC()
	validPayload := &guildevents.GuildConfigCreationRequestedPayloadV1{
		GuildID:              sharedtypes.GuildID("guild-1"),
		SignupChannelID:      "signup-chan",
		SignupMessageID:      "msg-1",
		EventChannelID:       "event-chan",
		LeaderboardChannelID: "leaderboard-chan",
		UserRoleID:           "role-1",
		EditorRoleID:         "role-2",
		AdminRoleID:          "role-3",
		SignupEmoji:          ":frolf:",
		AutoSetupCompleted:   true,
		SetupCompletedAt:     &setupTime,
	}

	expectedConfig := &guildtypes.GuildConfig{
		GuildID:              sharedtypes.GuildID("guild-1"),
		SignupChannelID:      "signup-chan",
		SignupMessageID:      "msg-1",
		EventChannelID:       "event-chan",
		LeaderboardChannelID: "leaderboard-chan",
		UserRoleID:           "role-1",
		EditorRoleID:         "role-2",
		AdminRoleID:          "role-3",
		SignupEmoji:          ":frolf:",
		AutoSetupCompleted:   true,
		SetupCompletedAt:     &setupTime,
	}

	payloadBytes, _ := json.Marshal(validPayload)
	testMsg := message.NewMessage(uuid.New().String(), payloadBytes)
	testMsg.Metadata.Set("guild_id", "guild-1")
	invalidMsg := message.NewMessage(uuid.New().String(), []byte("invalid json"))

	mockService := guildmocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &guildmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockSetup      func()
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "success",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*guildevents.GuildConfigCreationRequestedPayloadV1) = *validPayload
						return nil
					},
				)
				mockService.EXPECT().CreateGuildConfig(gomock.Any(), expectedConfig).Return(guildservice.GuildOperationResult{
					Success: &guildevents.GuildConfigCreatedPayload{
						GuildID: sharedtypes.GuildID("guild-1"),
						Config:  *expectedConfig,
					},
				}, nil)
				successMsg := message.NewMessage(uuid.New().String(), []byte("success"))
				successMsg.Metadata.Set("guild_id", "guild-1")
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&guildevents.GuildConfigCreatedPayload{
						GuildID: sharedtypes.GuildID("guild-1"),
						Config:  *expectedConfig,
					},
					guildevents.GuildConfigCreatedV1,
				).Return(successMsg, nil)
			},
			msg:     testMsg,
			wantErr: false,
		},
		{
			name: "fail to unmarshal payload",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload",
		},
		{
			name: "service returns error",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*guildevents.GuildConfigCreationRequestedPayloadV1) = *validPayload
						return nil
					},
				)
				mockService.EXPECT().CreateGuildConfig(gomock.Any(), expectedConfig).Return(guildservice.GuildOperationResult{}, fmt.Errorf("database error"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle guild.setup event: database error",
		},
		{
			name: "service returns failure payload",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*guildevents.GuildConfigCreationRequestedPayloadV1) = *validPayload
						return nil
					},
				)
				mockService.EXPECT().CreateGuildConfig(gomock.Any(), expectedConfig).Return(guildservice.GuildOperationResult{
					Failure: &guildevents.GuildConfigCreationFailedPayload{
						GuildID: sharedtypes.GuildID("guild-1"),
						Reason:  "config already exists",
					},
				}, nil)
				failureMsg := message.NewMessage(uuid.New().String(), []byte("failure"))
				failureMsg.Metadata.Set("guild_id", "guild-1")
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&guildevents.GuildConfigCreationFailedPayload{
						GuildID: sharedtypes.GuildID("guild-1"),
						Reason:  "config already exists",
					},
					guildevents.GuildConfigCreationFailedV1,
				).Return(failureMsg, nil)
			},
			msg:     testMsg,
			wantErr: false,
		},
		{
			name: "error creating success message",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*guildevents.GuildConfigCreationRequestedPayloadV1) = *validPayload
						return nil
					},
				)
				mockService.EXPECT().CreateGuildConfig(gomock.Any(), expectedConfig).Return(guildservice.GuildOperationResult{
					Success: &guildevents.GuildConfigCreatedPayload{
						GuildID: sharedtypes.GuildID("guild-1"),
						Config:  *expectedConfig,
					},
				}, nil)
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					guildevents.GuildConfigCreatedV1,
				).Return(nil, fmt.Errorf("message creation error"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: message creation error",
		},
		{
			name: "error creating failure message",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*guildevents.GuildConfigCreationRequestedPayloadV1) = *validPayload
						return nil
					},
				)
				mockService.EXPECT().CreateGuildConfig(gomock.Any(), expectedConfig).Return(guildservice.GuildOperationResult{
					Failure: &guildevents.GuildConfigCreationFailedPayload{
						GuildID: sharedtypes.GuildID("guild-1"),
						Reason:  "config already exists",
					},
				}, nil)
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					guildevents.GuildConfigCreationFailedV1,
				).Return(nil, fmt.Errorf("message creation error"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create failure message: message creation error",
		},
		{
			name: "unexpected result from service",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*guildevents.GuildConfigCreationRequestedPayloadV1) = *validPayload
						return nil
					},
				)
				mockService.EXPECT().CreateGuildConfig(gomock.Any(), expectedConfig).Return(guildservice.GuildOperationResult{}, nil)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockSetup != nil {
				tt.mockSetup()
			}
			h := &GuildHandlers{
				guildService: mockService,
				logger:       logger,
				tracer:       tracer,
				metrics:      metrics,
				helpers:      mockHelpers,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, metrics, tracer, mockHelpers)
				},
			}
			got, err := h.HandleGuildSetup(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildSetup() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err != nil && err.Error() != tt.expectedErrMsg {
				t.Errorf("expected error message %q, got %q", tt.expectedErrMsg, err.Error())
			}
			if tt.want != nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleGuildSetup() = %v, want %v", got, tt.want)
			}
		})
	}
}
