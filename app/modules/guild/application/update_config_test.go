package guildservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	guilddbmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestGuildService_UpdateGuildConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := guilddbmocks.NewMockRepository(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	metrics := &guildmetrics.NoOpMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	validConfig := &guildtypes.GuildConfig{
		GuildID:         "guild-1",
		SignupChannelID: "signup-chan",
	}

	tests := []struct {
		name        string
		mockSetup   func(*guilddbmocks.MockRepository)
		config      *guildtypes.GuildConfig
		wantSuccess bool
		wantFailure bool
		wantErr     error
		failReason  string
	}{
		{
			name: "success",
			mockSetup: func(m *guilddbmocks.MockRepository) {
				m.EXPECT().UpdateConfig(gomock.Any(), sharedtypes.GuildID("guild-1"), gomock.Any()).Return(nil)
			},
			config:      validConfig,
			wantSuccess: true,
		},
		{
			name: "not found",
			mockSetup: func(m *guilddbmocks.MockRepository) {
				m.EXPECT().UpdateConfig(gomock.Any(), sharedtypes.GuildID("guild-2"), gomock.Any()).Return(guilddb.ErrNoRowsAffected)
			},
			config:      &guildtypes.GuildConfig{GuildID: "guild-2"},
			wantFailure: true,
			failReason:  ErrGuildConfigNotFound.Error(),
		},
		{
			name: "db error on update",
			mockSetup: func(m *guilddbmocks.MockRepository) {
				m.EXPECT().UpdateConfig(gomock.Any(), sharedtypes.GuildID("guild-4"), gomock.Any()).Return(errors.New("update error"))
			},
			config:      &guildtypes.GuildConfig{GuildID: "guild-4"},
			wantFailure: true,
			wantErr:     errors.New("update error"),
			failReason:  "update error",
		},
		{
			name:        "invalid guildID",
			mockSetup:   func(m *guilddbmocks.MockRepository) {},
			config:      &guildtypes.GuildConfig{GuildID: ""},
			wantFailure: true,
			failReason:  ErrInvalidGuildID.Error(),
		},
		{
			name:        "nil config",
			mockSetup:   func(m *guilddbmocks.MockRepository) {},
			config:      nil,
			wantFailure: true,
			failReason:  ErrNilConfig.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup(mockRepo)
			s := &GuildService{
				repo:    mockRepo,
				logger:  logger,
				metrics: metrics,
				tracer:  tracer,
			}

			got, err := s.UpdateGuildConfig(ctx, tt.config)

			// Error check
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr.Error())
				} else if !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("expected error containing %q, got %q", tt.wantErr.Error(), err.Error())
				}
				return
			}

			if err != nil && tt.wantErr == nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Success check
			if tt.wantSuccess {
				if got.Success == nil {
					t.Errorf("expected success payload, got nil")
				}
			}

			// Failure check
			if tt.wantFailure {
				if got.Failure == nil {
					t.Errorf("expected failure payload, got nil")
					return
				}
				actual := got.Failure.(*guildevents.GuildConfigUpdateFailedPayloadV1)
				if actual.Reason != tt.failReason {
					t.Errorf("expected reason %q, got %q", tt.failReason, actual.Reason)
				}
			}
		})
	}
}
