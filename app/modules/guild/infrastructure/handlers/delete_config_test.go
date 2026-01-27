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
	"go.opentelemetry.io/otel/trace/noop"
)

func TestGuildHandlers_HandleDeleteGuildConfig(t *testing.T) {
	tests := []struct {
		name      string
		payload   *guildevents.GuildConfigDeletionRequestedPayloadV1
		setupFake func(*FakeGuildService)
		wantErr   bool
		wantTopic string
		wantLen   int
	}{
		{
			name: "success - guild config deleted",
			payload: &guildevents.GuildConfigDeletionRequestedPayloadV1{
				GuildID: sharedtypes.GuildID("guild-1"),
			},
			setupFake: func(f *FakeGuildService) {
				f.DeleteGuildConfigFunc = func(ctx context.Context, guildID sharedtypes.GuildID) (guildservice.GuildConfigResult, error) {
					return results.SuccessResult[*guildtypes.GuildConfig, error](nil), nil
				}
			},
			wantErr:   false,
			wantTopic: guildevents.GuildConfigDeletedV1,
			wantLen:   1,
		},
		{
			name: "failure - guild config not found",
			payload: &guildevents.GuildConfigDeletionRequestedPayloadV1{
				GuildID: sharedtypes.GuildID("guild-1"),
			},
			setupFake: func(f *FakeGuildService) {
				f.DeleteGuildConfigFunc = func(ctx context.Context, guildID sharedtypes.GuildID) (guildservice.GuildConfigResult, error) {
					// Return domain failure result
					return results.FailureResult[*guildtypes.GuildConfig, error](guildservice.ErrGuildConfigNotFound), nil
				}
			},
			wantErr:   false,
			wantTopic: guildevents.GuildConfigDeletionFailedV1,
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
			payload: &guildevents.GuildConfigDeletionRequestedPayloadV1{
				GuildID: sharedtypes.GuildID("guild-1"),
			},
			setupFake: func(f *FakeGuildService) {
				f.DeleteGuildConfigFunc = func(ctx context.Context, guildID sharedtypes.GuildID) (guildservice.GuildConfigResult, error) {
					// Return infrastructure error
					return guildservice.GuildConfigResult{}, context.DeadlineExceeded
				}
			},
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := NewFakeGuildService()
			if tt.setupFake != nil {
				tt.setupFake(fakeService)
			}

			logger := loggerfrolfbot.NoOpLogger
			tracer := noop.NewTracerProvider().Tracer("test")
			metrics := &guildmetrics.NoOpMetrics{}

			h := NewGuildHandlers(fakeService, logger, tracer, nil, metrics)
			res, err := h.HandleDeleteGuildConfig(context.Background(), tt.payload)

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
