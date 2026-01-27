package guildhandlers

import (
	"context"
	"testing"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	guildservice "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestGuildHandlers_HandleUpdateGuildConfig(t *testing.T) {
	tests := []struct {
		name      string
		payload   *guildevents.GuildConfigUpdateRequestedPayloadV1
		setupFake func(*FakeGuildService)
		wantErr   bool
		wantTopic string
		wantLen   int
	}{
		{
			name: "success - guild config updated",
			payload: &guildevents.GuildConfigUpdateRequestedPayloadV1{
				GuildID:         "guild-1",
				SignupChannelID: "signup-chan",
				SignupEmoji:     ":frolf:",
			},
			setupFake: func(f *FakeGuildService) {
				f.UpdateGuildConfigFunc = func(ctx context.Context, config *guildtypes.GuildConfig) (guildservice.GuildConfigResult, error) {
					// Return success result with the config passed in
					return results.SuccessResult[*guildtypes.GuildConfig, error](config), nil
				}
			},
			wantErr:   false,
			wantTopic: guildevents.GuildConfigUpdatedV1,
			wantLen:   1,
		},
		{
			name: "failure - validation error",
			payload: &guildevents.GuildConfigUpdateRequestedPayloadV1{
				GuildID: "", // Invalid ID
			},
			setupFake: func(f *FakeGuildService) {
				f.UpdateGuildConfigFunc = func(ctx context.Context, config *guildtypes.GuildConfig) (guildservice.GuildConfigResult, error) {
					// Simulate domain failure (validation)
					return results.FailureResult[*guildtypes.GuildConfig, error](guildservice.ErrInvalidGuildID), nil
				}
			},
			wantErr:   false,
			wantTopic: guildevents.GuildConfigUpdateFailedV1,
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
			payload: &guildevents.GuildConfigUpdateRequestedPayloadV1{
				GuildID: "guild-1",
			},
			setupFake: func(f *FakeGuildService) {
				f.UpdateGuildConfigFunc = func(ctx context.Context, config *guildtypes.GuildConfig) (guildservice.GuildConfigResult, error) {
					// Simulate infrastructure/network error
					return guildservice.GuildConfigResult{}, context.DeadlineExceeded
				}
			},
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize the manual fake service
			fakeService := NewFakeGuildService()
			if tt.setupFake != nil {
				tt.setupFake(fakeService)
			}

			logger := loggerfrolfbot.NoOpLogger
			tracer := noop.NewTracerProvider().Tracer("test")
			metrics := &guildmetrics.NoOpMetrics{}

			// Inject the fake service into the handler
			h := NewGuildHandlers(fakeService, logger, tracer, nil, metrics)
			res, err := h.HandleUpdateGuildConfig(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, want error %v", err, tt.wantErr)
			}

			if len(res) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(res), tt.wantLen)
			}

			if len(res) > 0 && res[0].Topic != tt.wantTopic {
				t.Errorf("got topic %s, want %s", res[0].Topic, tt.wantTopic)
			}
		})
	}
}
