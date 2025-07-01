package guildhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	guildservice "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application"
	guildmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestGuildHandlers_HandleUpdateGuildConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	validPayload := &guildevents.GuildConfigUpdateRequestedPayload{
		GuildID:              "guild-1",
		SignupChannelID:      "signup-chan",
		SignupMessageID:      "msg-1",
		EventChannelID:       "event-chan",
		LeaderboardChannelID: "leaderboard-chan",
		UserRoleID:           "role-1",
		EditorRoleID:         "role-2",
		AdminRoleID:          "role-3",
		SignupEmoji:          ":frolf:",
		AutoSetupCompleted:   true,
	}
	validConfig := &guildtypes.GuildConfig{
		GuildID:              "guild-1",
		SignupChannelID:      "signup-chan",
		SignupMessageID:      "msg-1",
		EventChannelID:       "event-chan",
		LeaderboardChannelID: "leaderboard-chan",
		UserRoleID:           "role-1",
		EditorRoleID:         "role-2",
		AdminRoleID:          "role-3",
		SignupEmoji:          ":frolf:",
		AutoSetupCompleted:   true,
	}
	payloadBytes, _ := json.Marshal(validPayload)
	testMsg := message.NewMessage(uuid.New().String(), payloadBytes)
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
						*out.(*guildevents.GuildConfigUpdateRequestedPayload) = *validPayload
						return nil
					},
				)
				mockService.EXPECT().UpdateGuildConfig(gomock.Any(), validConfig).Return(guildservice.GuildOperationResult{
					Success: &guildevents.GuildConfigUpdatedPayload{
						GuildID: "guild-1",
						Config:  *validConfig,
					},
				}, nil)
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&guildevents.GuildConfigUpdatedPayload{
						GuildID: "guild-1",
						Config:  *validConfig,
					},
					guildevents.GuildConfigUpdated,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
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
			name: "service failure",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*guildevents.GuildConfigUpdateRequestedPayload) = *validPayload
						return nil
					},
				)
				mockService.EXPECT().UpdateGuildConfig(gomock.Any(), validConfig).Return(guildservice.GuildOperationResult{}, fmt.Errorf("internal service error"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle GuildConfigUpdateRequested event: internal service error",
		},
		{
			name: "failure payload",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*guildevents.GuildConfigUpdateRequestedPayload) = *validPayload
						return nil
					},
				)
				mockService.EXPECT().UpdateGuildConfig(gomock.Any(), validConfig).Return(guildservice.GuildOperationResult{
					Failure: &guildevents.GuildConfigUpdateFailedPayload{
						GuildID: "guild-1",
						Reason:  "some failure",
					},
				}, nil)
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&guildevents.GuildConfigUpdateFailedPayload{
						GuildID: "guild-1",
						Reason:  "some failure",
					},
					guildevents.GuildConfigUpdateFailed,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "unexpected result",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*guildevents.GuildConfigUpdateRequestedPayload) = *validPayload
						return nil
					},
				)
				mockService.EXPECT().UpdateGuildConfig(gomock.Any(), validConfig).Return(guildservice.GuildOperationResult{}, nil)
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
			got, err := h.HandleUpdateGuildConfig(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUpdateGuildConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err != nil && err.Error() != tt.expectedErrMsg {
				t.Errorf("expected error message %q, got %q", tt.expectedErrMsg, err.Error())
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleUpdateGuildConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
