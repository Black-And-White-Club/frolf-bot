package guildservice

import (
	"context"
	"errors"
	"testing"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestGuildService_UpdateGuildConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockDB := guilddb.NewMockGuildDB(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	metrics := &guildmetrics.NoOpMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	validConfig := &guildtypes.GuildConfig{
		GuildID:              "guild-1",
		SignupChannelID:      "signup-chan",
		EventChannelID:       "event-chan",
		LeaderboardChannelID: "leaderboard-chan",
		UserRoleID:           "role-1",
		SignupEmoji:          ":frolf:",
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
				m.EXPECT().GetConfig(gomock.Any(), sharedtypes.GuildID("guild-1")).Return(validConfig, nil)
				m.EXPECT().UpdateConfig(gomock.Any(), sharedtypes.GuildID("guild-1"), gomock.Any()).Return(nil)
			},
			config: validConfig,
			wantResult: GuildOperationResult{
				Success: &guildevents.GuildConfigUpdatedPayload{
					GuildID: "guild-1",
					Config:  *validConfig,
				},
			},
			wantErr: nil,
		},
		{
			name: "not found",
			mockDBSetup: func(m *guilddb.MockGuildDB) {
				m.EXPECT().GetConfig(gomock.Any(), sharedtypes.GuildID("guild-2")).Return(nil, nil)
			},
			config: &guildtypes.GuildConfig{
				GuildID: "guild-2",
			},
			wantResult: GuildOperationResult{
				Failure: &guildevents.GuildConfigUpdateFailedPayload{
					GuildID: "guild-2",
					Reason:  "guild config not found",
				},
			},
			wantErr: errors.New("guild config not found"),
		},
		{
			name: "db error on get",
			mockDBSetup: func(m *guilddb.MockGuildDB) {
				m.EXPECT().GetConfig(gomock.Any(), sharedtypes.GuildID("guild-3")).Return(nil, errors.New("db error"))
			},
			config: &guildtypes.GuildConfig{
				GuildID: "guild-3",
			},
			wantResult: GuildOperationResult{
				Failure: &guildevents.GuildConfigUpdateFailedPayload{
					GuildID: "guild-3",
					Reason:  "could not fetch existing config: db error",
				},
			},
			wantErr: errors.New("db error"),
		},
		{
			name: "db error on update",
			mockDBSetup: func(m *guilddb.MockGuildDB) {
				m.EXPECT().GetConfig(gomock.Any(), sharedtypes.GuildID("guild-4")).Return(validConfig, nil)
				m.EXPECT().UpdateConfig(gomock.Any(), sharedtypes.GuildID("guild-4"), gomock.Any()).Return(errors.New("update error"))
			},
			config: &guildtypes.GuildConfig{
				GuildID:              "guild-4",
				SignupChannelID:      "signup-chan",
				EventChannelID:       "event-chan",
				LeaderboardChannelID: "leaderboard-chan",
				UserRoleID:           "role-1",
				SignupEmoji:          ":frolf:",
			},
			wantResult: GuildOperationResult{
				Failure: &guildevents.GuildConfigUpdateFailedPayload{
					GuildID: "guild-4",
					Reason:  "update error",
				},
			},
			wantErr: errors.New("update error"),
		},
		{
			name:        "invalid guildID",
			mockDBSetup: func(m *guilddb.MockGuildDB) {},
			config:      &guildtypes.GuildConfig{},
			wantResult: GuildOperationResult{
				Error: errors.New("invalid guild ID"),
			},
			wantErr: errors.New("invalid guild ID"),
		},
		{
			name:        "nil context",
			mockDBSetup: func(m *guilddb.MockGuildDB) {},
			config:      &guildtypes.GuildConfig{GuildID: "guild-5"},
			wantResult: GuildOperationResult{
				Error: errors.New("context cannot be nil"),
			},
			wantErr: errors.New("context cannot be nil"),
		},
		{
			name:        "nil config",
			mockDBSetup: func(m *guilddb.MockGuildDB) {},
			config:      nil,
			wantResult: GuildOperationResult{
				Error: errors.New("config cannot be nil"),
			},
			wantErr: errors.New("config cannot be nil"),
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
			callCtx := ctx
			if tt.name == "nil context" {
				callCtx = nil
			}
			got, err := s.UpdateGuildConfig(callCtx, tt.config)
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
				exp := tt.wantResult.Success.(*guildevents.GuildConfigUpdatedPayload)
				actual := got.Success.(*guildevents.GuildConfigUpdatedPayload)
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
				exp := tt.wantResult.Failure.(*guildevents.GuildConfigUpdateFailedPayload)
				actual := got.Failure.(*guildevents.GuildConfigUpdateFailedPayload)
				if exp.GuildID != actual.GuildID {
					t.Errorf("expected failure GuildID %q, got %q", exp.GuildID, actual.GuildID)
				}
				if exp.Reason != actual.Reason {
					t.Errorf("expected failure Reason %q, got %q", exp.Reason, actual.Reason)
				}
			}
			if tt.wantResult.Error != nil {
				if got.Error == nil || got.Error.Error() != tt.wantResult.Error.Error() {
					t.Errorf("expected error payload %v, got %v", tt.wantResult.Error, got.Error)
				}
			}
		})
	}
}
