package guildservice

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

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

func TestGuildService_CreateGuildConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	mockRepo := guilddbmocks.NewMockRepository(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	metrics := &guildmetrics.NoOpMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	setupTime := time.Now().UTC()

	validConfig := &guildtypes.GuildConfig{
		GuildID:              "guild-1",
		SignupChannelID:      "signup-chan",
		EventChannelID:       "event-chan",
		LeaderboardChannelID: "leaderboard-chan",
		UserRoleID:           "role-1",
		SignupEmoji:          ":frolf:",
		AutoSetupCompleted:   true,
		SetupCompletedAt:     &setupTime,
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
				m.EXPECT().GetConfig(gomock.Any(), sharedtypes.GuildID("guild-1")).Return(nil, guilddb.ErrNotFound)
				m.EXPECT().SaveConfig(gomock.Any(), validConfig).Return(nil)
			},
			config:      validConfig,
			wantSuccess: true,
		},
		{
			name: "db error on save",
			mockSetup: func(m *guilddbmocks.MockRepository) {
				m.EXPECT().GetConfig(gomock.Any(), sharedtypes.GuildID("guild-2")).Return(nil, guilddb.ErrNotFound)
				m.EXPECT().SaveConfig(gomock.Any(), gomock.Any()).Return(errors.New("db error"))
			},
			config: &guildtypes.GuildConfig{
				GuildID:              "guild-2",
				SignupChannelID:      "signup-chan",
				EventChannelID:       "event-chan",
				LeaderboardChannelID: "leaderboard-chan",
				UserRoleID:           "role-1",
				SignupEmoji:          ":frolf:",
			},
			wantFailure: true,
			wantErr:     errors.New("db error"),
			failReason:  "db error",
		},
		{
			name: "idempotent when config matches",
			mockSetup: func(m *guilddbmocks.MockRepository) {
				existing := *validConfig
				m.EXPECT().GetConfig(gomock.Any(), sharedtypes.GuildID("guild-1")).Return(&existing, nil)
			},
			config:      validConfig,
			wantSuccess: true,
		},
		{
			name: "already exists with different settings",
			mockSetup: func(m *guilddbmocks.MockRepository) {
				existing := *validConfig
				existing.SignupChannelID = "another-chan"
				m.EXPECT().GetConfig(gomock.Any(), sharedtypes.GuildID("guild-3")).Return(&existing, nil)
			},
			config: &guildtypes.GuildConfig{
				GuildID:              "guild-3",
				SignupChannelID:      "signup-chan",
				EventChannelID:       "event-chan",
				LeaderboardChannelID: "leaderboard-chan",
				UserRoleID:           "role-1",
				SignupEmoji:          ":frolf:",
				AutoSetupCompleted:   true,
				SetupCompletedAt:     &setupTime,
			},
			wantFailure: true,
			failReason:  ErrGuildConfigConflict.Error(),
		},
		{
			name:        "missing required field - signup channel",
			mockSetup:   func(m *guilddbmocks.MockRepository) {},
			config:      &guildtypes.GuildConfig{GuildID: "guild-4"},
			wantFailure: true,
			failReason:  "signup channel ID required",
		},
		{
			name:        "nil config",
			mockSetup:   func(m *guilddbmocks.MockRepository) {},
			config:      nil,
			wantFailure: true,
			failReason:  ErrNilConfig.Error(),
		},
		{
			name:      "empty guild ID",
			mockSetup: func(m *guilddbmocks.MockRepository) {},
			config: &guildtypes.GuildConfig{
				GuildID:              "",
				SignupChannelID:      "signup-chan",
				EventChannelID:       "event-chan",
				LeaderboardChannelID: "leaderboard-chan",
				UserRoleID:           "role-1",
				SignupEmoji:          ":frolf:",
			},
			wantFailure: true,
			failReason:  ErrInvalidGuildID.Error(),
		},
		{
			name: "GetConfig returns database error",
			mockSetup: func(m *guilddbmocks.MockRepository) {
				m.EXPECT().GetConfig(gomock.Any(), sharedtypes.GuildID("guild-9")).Return(nil, errors.New("db lookup error"))
			},
			config: &guildtypes.GuildConfig{
				GuildID:              "guild-9",
				SignupChannelID:      "signup-chan",
				EventChannelID:       "event-chan",
				LeaderboardChannelID: "leaderboard-chan",
				UserRoleID:           "role-1",
				SignupEmoji:          ":frolf:",
			},
			wantFailure: true,
			wantErr:     errors.New("db lookup error"),
			failReason:  "db lookup error",
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

			got, err := s.CreateGuildConfig(ctx, tt.config)

			// Error check
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr.Error())
				} else if !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("expected error containing %q, got %q", tt.wantErr.Error(), err.Error())
				}
			}

			// Success check
			if tt.wantSuccess {
				if got.Success == nil {
					t.Errorf("expected success, got nil")
					return
				}
				actual := got.Success.(*guildevents.GuildConfigCreatedPayloadV1)
				if actual.Config.SignupEmoji != validConfig.SignupEmoji {
					t.Errorf("expected Emoji %q, got %q", validConfig.SignupEmoji, actual.Config.SignupEmoji)
				}
			}

			// Failure check
			if tt.wantFailure {
				if got.Failure == nil {
					t.Errorf("expected failure payload, got nil")
					return
				}
				actual := got.Failure.(*guildevents.GuildConfigCreationFailedPayloadV1)
				if actual.Reason != tt.failReason {
					t.Errorf("expected failure Reason %q, got %q", tt.failReason, actual.Reason)
				}
			}
		})
	}
}
