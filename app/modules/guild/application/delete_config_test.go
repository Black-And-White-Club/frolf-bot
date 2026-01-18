package guildservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guilddbmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestGuildService_DeleteGuildConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := guilddbmocks.NewMockRepository(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	metrics := &guildmetrics.NoOpMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name        string
		mockSetup   func(*guilddbmocks.MockRepository)
		guildID     sharedtypes.GuildID
		wantSuccess bool
		wantFailure bool
		wantErr     error
		failReason  string
	}{
		{
			name: "success",
			mockSetup: func(m *guilddbmocks.MockRepository) {
				m.EXPECT().DeleteConfig(gomock.Any(), sharedtypes.GuildID("guild-1")).Return(nil)
			},
			guildID:     "guild-1",
			wantSuccess: true,
		},
		{
			name: "idempotent - already deleted returns success",
			mockSetup: func(m *guilddbmocks.MockRepository) {
				// Repo is idempotent - returns nil for already deleted
				m.EXPECT().DeleteConfig(gomock.Any(), sharedtypes.GuildID("guild-2")).Return(nil)
			},
			guildID:     "guild-2",
			wantSuccess: true,
		},
		{
			name: "db error on delete",
			mockSetup: func(m *guilddbmocks.MockRepository) {
				m.EXPECT().DeleteConfig(gomock.Any(), sharedtypes.GuildID("guild-4")).Return(errors.New("delete error"))
			},
			guildID:     "guild-4",
			wantFailure: true,
			wantErr:     errors.New("delete error"),
			failReason:  "delete error",
		},
		{
			name:        "invalid guildID",
			mockSetup:   func(m *guilddbmocks.MockRepository) {},
			guildID:     "",
			wantFailure: true,
			failReason:  ErrInvalidGuildID.Error(),
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

			got, err := s.DeleteGuildConfig(ctx, tt.guildID)

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
					t.Errorf("expected success, got nil")
					return
				}
				actual := got.Success.(*guildevents.GuildConfigDeletedPayloadV1)
				if actual.GuildID != tt.guildID {
					t.Errorf("expected GuildID %q, got %q", tt.guildID, actual.GuildID)
				}
			}

			// Failure check
			if tt.wantFailure {
				if got.Failure == nil {
					t.Errorf("expected failure payload, got nil")
					return
				}
				actual := got.Failure.(*guildevents.GuildConfigDeletionFailedPayloadV1)
				if actual.Reason != tt.failReason {
					t.Errorf("expected failure Reason %q, got %q", tt.failReason, actual.Reason)
				}
			}
		})
	}
}
