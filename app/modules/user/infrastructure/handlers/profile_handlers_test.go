package userhandlers

import (
	"context"
	"errors"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestUserHandlers_HandleUserProfileUpdated(t *testing.T) {
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testDisplayName := "Test User"
	testAvatarHash := "hash123"

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	tests := []struct {
		name      string
		payload   *userevents.UserProfileUpdatedPayloadV1
		setupFake func(*FakeUserService)
		wantErr   bool
	}{
		{
			name: "Successful profile update",
			payload: &userevents.UserProfileUpdatedPayloadV1{
				UserID:      testUserID,
				DisplayName: testDisplayName,
				AvatarHash:  testAvatarHash,
			},
			setupFake: func(f *FakeUserService) {
				f.UpdateUserProfileFunc = func(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, displayName, avatarHash string) error {
					if userID != testUserID || displayName != testDisplayName || avatarHash != testAvatarHash {
						return errors.New("unexpected arguments")
					}
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "Service error (should be handled gracefully)",
			payload: &userevents.UserProfileUpdatedPayloadV1{
				UserID: testUserID,
			},
			setupFake: func(f *FakeUserService) {
				f.UpdateUserProfileFunc = func(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, displayName, avatarHash string) error {
					return errors.New("db error")
				}
			},
			wantErr: false, // Handler swallows errors for profile updates
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := NewFakeUserService()
			if tt.setupFake != nil {
				tt.setupFake(fake)
			}

			h := NewUserHandlers(fake, logger, tracer, nil, metrics)
			res, err := h.HandleUserProfileUpdated(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUserProfileUpdated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if res != nil {
				t.Errorf("expected nil results, got %v", res)
			}
		})
	}
}
