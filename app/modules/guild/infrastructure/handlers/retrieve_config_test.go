package guildhandlers

import (
	"context"
	"testing"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	guildservice "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application"
	guildmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestGuildHandlers_HandleRetrieveGuildConfig(t *testing.T) {
	tests := []struct {
		name      string
		payload   *guildevents.GuildConfigRetrievalRequestedPayloadV1
		mockSetup func(*guildmocks.MockService)
		wantErr   bool
		wantTopic string
		wantLen   int
	}{
		{
			name: "success - guild config retrieved",
			payload: &guildevents.GuildConfigRetrievalRequestedPayloadV1{
				GuildID: sharedtypes.GuildID("guild-1"),
			},
			mockSetup: func(m *guildmocks.MockService) {
				m.EXPECT().GetGuildConfig(gomock.Any(), sharedtypes.GuildID("guild-1")).Return(results.SuccessResult(&guildevents.GuildConfigRetrievedPayloadV1{
					GuildID: sharedtypes.GuildID("guild-1"),
					Config: guildtypes.GuildConfig{
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
					},
				}), nil)
			},
			wantErr:   false,
			wantTopic: guildevents.GuildConfigRetrievedV1,
			wantLen:   1,
		},
		{
			name: "failure - guild config not found",
			payload: &guildevents.GuildConfigRetrievalRequestedPayloadV1{
				GuildID: sharedtypes.GuildID("guild-1"),
			},
			mockSetup: func(m *guildmocks.MockService) {
				m.EXPECT().GetGuildConfig(gomock.Any(), sharedtypes.GuildID("guild-1")).Return(results.FailureResult(&guildevents.GuildConfigRetrievalFailedPayloadV1{
					GuildID: sharedtypes.GuildID("guild-1"),
					Reason:  guildservice.ErrGuildConfigNotFound.Error(),
				}), nil)
			},
			wantErr:   false,
			wantTopic: guildevents.GuildConfigRetrievalFailedV1,
			wantLen:   1,
		},
		{
			name:    "error - nil payload",
			payload: nil,
			wantErr: true,
			wantLen: 0,
		},
		{
			name: "error - service error",
			payload: &guildevents.GuildConfigRetrievalRequestedPayloadV1{
				GuildID: sharedtypes.GuildID("guild-1"),
			},
			mockSetup: func(m *guildmocks.MockService) {
				m.EXPECT().GetGuildConfig(gomock.Any(), sharedtypes.GuildID("guild-1")).Return(results.OperationResult{}, context.DeadlineExceeded)
			},
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := guildmocks.NewMockService(ctrl)
			if tt.mockSetup != nil {
				tt.mockSetup(mockService)
			}

			logger := loggerfrolfbot.NoOpLogger
			tracer := noop.NewTracerProvider().Tracer("test")
			metrics := &guildmetrics.NoOpMetrics{}

			h := NewGuildHandlers(mockService, logger, tracer, nil, metrics)
			results, err := h.HandleRetrieveGuildConfig(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, want error %v", err, tt.wantErr)
			}

			if len(results) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(results), tt.wantLen)
			}

			if len(results) > 0 && results[0].Topic != tt.wantTopic {
				t.Errorf("got topic %s, want %s", results[0].Topic, tt.wantTopic)
			}
		})
	}
}
