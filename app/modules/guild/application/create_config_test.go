package guildservice

import (
	"context"
	"errors"
	"testing"
	"time"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestGuildService_CreateGuildConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Mocks
	mockDB := guilddb.NewMockGuildDB(ctrl)
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
		mockDBSetup func(*guilddb.MockGuildDB)
		config      *guildtypes.GuildConfig
		wantResult  GuildOperationResult
		wantErr     error
	}{
		{
			name: "success",
			mockDBSetup: func(m *guilddb.MockGuildDB) {
				m.EXPECT().GetConfig(gomock.Any(), sharedtypes.GuildID("guild-1")).Return(nil, nil)
				m.EXPECT().SaveConfig(gomock.Any(), validConfig).Return(nil)
			},
			config: validConfig,
			wantResult: GuildOperationResult{
				Success: &guildevents.GuildConfigCreatedPayload{
					GuildID: "guild-1",
					Config:  *validConfig,
				},
			},
			wantErr: nil,
		},
		{
			name: "db error",
			mockDBSetup: func(m *guilddb.MockGuildDB) {
				m.EXPECT().GetConfig(gomock.Any(), sharedtypes.GuildID("guild-2")).Return(nil, nil)
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
			wantResult: GuildOperationResult{
				Failure: &guildevents.GuildConfigCreationFailedPayload{
					GuildID: "guild-2",
					Reason:  "db error",
				},
			},
			wantErr: errors.New("db error"),
		},
		{
			name: "idempotent when config matches",
			mockDBSetup: func(m *guilddb.MockGuildDB) {
				existing := *validConfig
				m.EXPECT().GetConfig(gomock.Any(), sharedtypes.GuildID("guild-1")).Return(&existing, nil)
			},
			config: validConfig,
			wantResult: GuildOperationResult{
				Success: &guildevents.GuildConfigCreatedPayload{
					GuildID: "guild-1",
					Config:  *validConfig,
				},
			},
			wantErr: nil,
		},
		{
			name: "already exists with different settings",
			mockDBSetup: func(m *guilddb.MockGuildDB) {
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
			wantResult: GuildOperationResult{
				Failure: &guildevents.GuildConfigCreationFailedPayload{
					GuildID: "guild-3",
					Reason:  ErrGuildConfigConflict.Error(),
				},
			},
			wantErr: ErrGuildConfigConflict,
		},
		{
			name:        "missing required field",
			mockDBSetup: func(m *guilddb.MockGuildDB) {},
			config: &guildtypes.GuildConfig{
				GuildID: "guild-4",
				// Missing SignupChannelID, etc.
			},
			wantResult: GuildOperationResult{
				Failure: &guildevents.GuildConfigCreationFailedPayload{
					GuildID: "guild-4",
					Reason:  "signup channel ID required",
				},
			},
			wantErr: errors.New("signup channel ID required"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockDBSetup(mockDB)
			s := &GuildService{
				GuildDB: mockDB,
				logger:  logger,
				metrics: metrics,
				tracer:  tracer,
				serviceWrapper: func(ctx context.Context, operationName string, guildID sharedtypes.GuildID, serviceFunc func(ctx context.Context) (GuildOperationResult, error)) (GuildOperationResult, error) {
					return serviceFunc(ctx)
				},
			}
			got, err := s.CreateGuildConfig(ctx, tt.config)
			if tt.wantErr != nil {
				if err == nil || err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			if tt.wantResult.Success != nil {
				if got.Success == nil {
					t.Errorf("expected success, got nil")
					return
				}
				// Compare payloads
				exp := tt.wantResult.Success.(*guildevents.GuildConfigCreatedPayload)
				actual := got.Success.(*guildevents.GuildConfigCreatedPayload)
				if exp.GuildID != actual.GuildID {
					t.Errorf("expected GuildID %q, got %q", exp.GuildID, actual.GuildID)
				}
				if exp.Config != actual.Config {
					t.Errorf("expected Config %v, got %v", exp.Config, actual.Config)
				}
			}
			if tt.wantResult.Failure != nil {
				if got.Failure == nil {
					t.Errorf("expected failure, got nil")
					return
				}
				exp := tt.wantResult.Failure.(*guildevents.GuildConfigCreationFailedPayload)
				actual := got.Failure.(*guildevents.GuildConfigCreationFailedPayload)
				if exp.GuildID != actual.GuildID {
					t.Errorf("expected failure GuildID %q, got %q", exp.GuildID, actual.GuildID)
				}
				if exp.Reason != actual.Reason {
					t.Errorf("expected failure Reason %q, got %q", exp.Reason, actual.Reason)
				}
			}
		})
	}
}
