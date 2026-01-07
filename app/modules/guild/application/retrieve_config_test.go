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

func TestGuildService_GetGuildConfig(t *testing.T) {
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
		guildID     sharedtypes.GuildID
		wantResult  GuildOperationResult
		wantErr     error
	}{
		{
			name: "success",
			mockDBSetup: func(m *guilddb.MockGuildDB) {
				m.EXPECT().GetConfig(gomock.Any(), sharedtypes.GuildID("guild-1")).Return(validConfig, nil)
			},
			guildID: "guild-1",
			wantResult: GuildOperationResult{
				Success: &guildevents.GuildConfigRetrievedPayload{
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
			guildID: "guild-2",
			wantResult: GuildOperationResult{
				Failure: &guildevents.GuildConfigRetrievalFailedPayload{
					GuildID: "guild-2",
					Reason:  "guild config not found",
				},
			},
			wantErr: errors.New("guild config not found"),
		},
		{
			name: "db error",
			mockDBSetup: func(m *guilddb.MockGuildDB) {
				m.EXPECT().GetConfig(gomock.Any(), sharedtypes.GuildID("guild-3")).Return(nil, errors.New("db error"))
			},
			guildID: "guild-3",
			wantResult: GuildOperationResult{
				Failure: &guildevents.GuildConfigRetrievalFailedPayload{
					GuildID: "guild-3",
					Reason:  "db error",
				},
			},
			wantErr: errors.New("db error"),
		},
		{
			name:        "invalid guildID",
			mockDBSetup: func(m *guilddb.MockGuildDB) {},
			guildID:     "",
			wantResult: GuildOperationResult{
				Error: errors.New("invalid guild ID"),
			},
			wantErr: errors.New("invalid guild ID"),
		},
		{
			name:        "nil context",
			mockDBSetup: func(m *guilddb.MockGuildDB) {},
			guildID:     "guild-4",
			wantResult: GuildOperationResult{
				Error: errors.New("context cannot be nil"),
			},
			wantErr: errors.New("context cannot be nil"),
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
			got, err := s.GetGuildConfig(callCtx, tt.guildID)
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
				exp := tt.wantResult.Success.(*guildevents.GuildConfigRetrievedPayload)
				actual := got.Success.(*guildevents.GuildConfigRetrievedPayload)
				if exp.GuildID != actual.GuildID {
					t.Errorf("expected GuildID %q, got %q", exp.GuildID, actual.GuildID)
				}
				expCfg := exp.Config
				actCfg := actual.Config
				if expCfg.SignupChannelID != actCfg.SignupChannelID {
					t.Errorf("expected SignupChannelID %v, got %v", expCfg.SignupChannelID, actCfg.SignupChannelID)
				}
				if expCfg.EventChannelID != actCfg.EventChannelID {
					t.Errorf("expected EventChannelID %v, got %v", expCfg.EventChannelID, actCfg.EventChannelID)
				}
				if expCfg.LeaderboardChannelID != actCfg.LeaderboardChannelID {
					t.Errorf("expected LeaderboardChannelID %v, got %v", expCfg.LeaderboardChannelID, actCfg.LeaderboardChannelID)
				}
				if expCfg.UserRoleID != actCfg.UserRoleID {
					t.Errorf("expected UserRoleID %v, got %v", expCfg.UserRoleID, actCfg.UserRoleID)
				}
				if expCfg.SignupEmoji != actCfg.SignupEmoji {
					t.Errorf("expected SignupEmoji %v, got %v", expCfg.SignupEmoji, actCfg.SignupEmoji)
				}
				if expCfg.AutoSetupCompleted != actCfg.AutoSetupCompleted {
					t.Errorf("expected AutoSetupCompleted %v, got %v", expCfg.AutoSetupCompleted, actCfg.AutoSetupCompleted)
				}
				if (expCfg.SetupCompletedAt == nil) != (actCfg.SetupCompletedAt == nil) {
					t.Errorf("expected SetupCompletedAt nil mismatch: %v vs %v", expCfg.SetupCompletedAt, actCfg.SetupCompletedAt)
				} else if expCfg.SetupCompletedAt != nil && actCfg.SetupCompletedAt != nil && !expCfg.SetupCompletedAt.Equal(*actCfg.SetupCompletedAt) {
					t.Errorf("expected SetupCompletedAt %v, got %v", expCfg.SetupCompletedAt, actCfg.SetupCompletedAt)
				}
			}
			if tt.wantResult.Failure != nil {
				if got.Failure == nil {
					t.Errorf("expected failure, got nil")
					return
				}
				exp := tt.wantResult.Failure.(*guildevents.GuildConfigRetrievalFailedPayload)
				actual := got.Failure.(*guildevents.GuildConfigRetrievalFailedPayload)
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
