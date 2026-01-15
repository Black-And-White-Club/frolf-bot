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
				Success: &guildevents.GuildConfigCreatedPayloadV1{
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
				Failure: &guildevents.GuildConfigCreationFailedPayloadV1{
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
				Success: &guildevents.GuildConfigCreatedPayloadV1{
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
				Failure: &guildevents.GuildConfigCreationFailedPayloadV1{
					GuildID: "guild-3",
					Reason:  ErrGuildConfigConflict.Error(),
				},
			},
			wantErr: ErrGuildConfigConflict,
		},
		{
			name:        "missing required field - signup channel",
			mockDBSetup: func(m *guilddb.MockGuildDB) {},
			config: &guildtypes.GuildConfig{
				GuildID: "guild-4",
				// Missing SignupChannelID, etc.
			},
			wantResult: GuildOperationResult{
				Failure: &guildevents.GuildConfigCreationFailedPayloadV1{
					GuildID: "guild-4",
					Reason:  "signup channel ID required",
				},
			},
			wantErr: errors.New("signup channel ID required"),
		},
		{
			name:        "missing required field - event channel",
			mockDBSetup: func(m *guilddb.MockGuildDB) {},
			config: &guildtypes.GuildConfig{
				GuildID:         "guild-5",
				SignupChannelID: "signup-chan",
				// Missing EventChannelID
			},
			wantResult: GuildOperationResult{
				Failure: &guildevents.GuildConfigCreationFailedPayloadV1{
					GuildID: "guild-5",
					Reason:  "event channel ID required",
				},
			},
			wantErr: errors.New("event channel ID required"),
		},
		{
			name:        "missing required field - leaderboard channel",
			mockDBSetup: func(m *guilddb.MockGuildDB) {},
			config: &guildtypes.GuildConfig{
				GuildID:         "guild-6",
				SignupChannelID: "signup-chan",
				EventChannelID:  "event-chan",
				// Missing LeaderboardChannelID
			},
			wantResult: GuildOperationResult{
				Failure: &guildevents.GuildConfigCreationFailedPayloadV1{
					GuildID: "guild-6",
					Reason:  "leaderboard channel ID required",
				},
			},
			wantErr: errors.New("leaderboard channel ID required"),
		},
		{
			name:        "missing required field - user role",
			mockDBSetup: func(m *guilddb.MockGuildDB) {},
			config: &guildtypes.GuildConfig{
				GuildID:              "guild-7",
				SignupChannelID:      "signup-chan",
				EventChannelID:       "event-chan",
				LeaderboardChannelID: "leaderboard-chan",
				// Missing UserRoleID
			},
			wantResult: GuildOperationResult{
				Failure: &guildevents.GuildConfigCreationFailedPayloadV1{
					GuildID: "guild-7",
					Reason:  "user role ID required",
				},
			},
			wantErr: errors.New("user role ID required"),
		},
		{
			name:        "missing required field - signup emoji",
			mockDBSetup: func(m *guilddb.MockGuildDB) {},
			config: &guildtypes.GuildConfig{
				GuildID:              "guild-8",
				SignupChannelID:      "signup-chan",
				EventChannelID:       "event-chan",
				LeaderboardChannelID: "leaderboard-chan",
				UserRoleID:           "role-1",
				// Missing SignupEmoji
			},
			wantResult: GuildOperationResult{
				Failure: &guildevents.GuildConfigCreationFailedPayloadV1{
					GuildID: "guild-8",
					Reason:  "signup emoji required",
				},
			},
			wantErr: errors.New("signup emoji required"),
		},
		{
			name:        "nil context",
			mockDBSetup: func(m *guilddb.MockGuildDB) {},
			config:      validConfig,
			wantResult: GuildOperationResult{
				Error: ErrNilContext,
			},
			wantErr: ErrNilContext,
		},
		{
			name:        "nil config",
			mockDBSetup: func(m *guilddb.MockGuildDB) {},
			config:      nil,
			wantResult: GuildOperationResult{
				Failure: &guildevents.GuildConfigCreationFailedPayloadV1{
					GuildID: "",
					Reason:  "config payload is nil",
				},
			},
			wantErr: errors.New("config payload is nil"),
		},
		{
			name:        "empty guild ID",
			mockDBSetup: func(m *guilddb.MockGuildDB) {},
			config: &guildtypes.GuildConfig{
				GuildID:              "",
				SignupChannelID:      "signup-chan",
				EventChannelID:       "event-chan",
				LeaderboardChannelID: "leaderboard-chan",
				UserRoleID:           "role-1",
				SignupEmoji:          ":frolf:",
			},
			wantResult: GuildOperationResult{
				Failure: &guildevents.GuildConfigCreationFailedPayloadV1{
					GuildID: "",
					Reason:  ErrInvalidGuildID.Error(),
				},
			},
			wantErr: ErrInvalidGuildID,
		},
		{
			name: "GetConfig returns database error",
			mockDBSetup: func(m *guilddb.MockGuildDB) {
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
			wantResult: GuildOperationResult{
				Failure: &guildevents.GuildConfigCreationFailedPayloadV1{
					GuildID: "guild-9",
					Reason:  "db lookup error",
				},
			},
			wantErr: errors.New("db lookup error"),
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

			// Use nil context if the test name indicates it
			testCtx := ctx
			if tt.name == "nil context" {
				testCtx = nil
			}

			got, err := s.CreateGuildConfig(testCtx, tt.config)
			if tt.wantErr != nil {
				// Accept the error either returned directly or embedded in the result.Error
				if err != nil {
					if err.Error() != tt.wantErr.Error() {
						t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
					}
				} else if got.Error != nil {
					if got.Error.Error() != tt.wantErr.Error() {
						t.Errorf("expected error: %v, got result.Error: %v", tt.wantErr, got.Error)
					}
				} else {
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
				exp := tt.wantResult.Success.(*guildevents.GuildConfigCreatedPayloadV1)
				actual := got.Success.(*guildevents.GuildConfigCreatedPayloadV1)
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
				exp := tt.wantResult.Failure.(*guildevents.GuildConfigCreationFailedPayloadV1)
				actual := got.Failure.(*guildevents.GuildConfigCreationFailedPayloadV1)
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
