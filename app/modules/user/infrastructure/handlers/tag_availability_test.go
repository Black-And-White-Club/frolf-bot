package userhandlers

import (
	"context"
	"errors"
	"testing"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestUserHandlers_HandleTagAvailable(t *testing.T) {
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("98765432109876543")
	testTagNumber := sharedtypes.TagNumber(1)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	tests := []struct {
		name      string
		payload   *sharedevents.TagAvailablePayloadV1
		setupFake func(*FakeUserService)
		wantLen   int
		wantTopic string
		wantErr   bool
	}{
		{
			name: "Successfully handle TagAvailable event",
			payload: &sharedevents.TagAvailablePayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: testTagNumber,
			},
			setupFake: func(f *FakeUserService) {
				f.CreateUserFunc = func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tag *sharedtypes.TagNumber, udiscUsername *string, udiscName *string) (userservice.UserResult, error) {
					return results.SuccessResult[*userservice.CreateUserResponse, error](&userservice.CreateUserResponse{
						UserData: usertypes.UserData{
							UserID: userID,
						},
						TagNumber: tag,
					}), nil
				}
			},
			wantLen:   1,
			wantTopic: userevents.UserCreatedV1,
			wantErr:   false,
		},
		{
			name: "Fail to create user (Domain Logic)",
			payload: &sharedevents.TagAvailablePayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: testTagNumber,
			},
			setupFake: func(f *FakeUserService) {
				f.CreateUserFunc = func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tag *sharedtypes.TagNumber, udiscUsername *string, udiscName *string) (userservice.UserResult, error) {
					return results.FailureResult[*userservice.CreateUserResponse, error](errors.New("conflict")), nil
				}
			},
			wantLen:   1,
			wantTopic: userevents.UserCreationFailedV1,
			wantErr:   false,
		},
		{
			name: "Service failure (Infrastructure)",
			payload: &sharedevents.TagAvailablePayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: testTagNumber,
			},
			setupFake: func(f *FakeUserService) {
				f.CreateUserFunc = func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tag *sharedtypes.TagNumber, udiscUsername *string, udiscName *string) (userservice.UserResult, error) {
					return userservice.UserResult{}, context.DeadlineExceeded
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := NewFakeUserService()
			if tt.setupFake != nil {
				tt.setupFake(fake)
			}

			h := NewUserHandlers(fake, logger, tracer, nil, metrics)
			res, err := h.HandleTagAvailable(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(res) != tt.wantLen {
					t.Errorf("got %d results, want %d", len(res), tt.wantLen)
					return
				}
				if res[0].Topic != tt.wantTopic {
					t.Errorf("got topic %s, want %s", res[0].Topic, tt.wantTopic)
				}
			}
		})
	}
}

func TestUserHandlers_HandleTagUnavailable(t *testing.T) {
	testUserID := sharedtypes.DiscordID("98765432109876543")
	testGuildID := sharedtypes.GuildID("98765432109876543")
	testTagNumber := sharedtypes.TagNumber(2)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	tests := []struct {
		name      string
		payload   *sharedevents.TagUnavailablePayloadV1
		wantLen   int
		wantTopic string
	}{
		{
			name: "Successfully handle TagUnavailable event",
			payload: &sharedevents.TagUnavailablePayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: testTagNumber,
				Reason:    "tag not available",
			},
			wantLen:   1,
			wantTopic: userevents.UserCreationFailedV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := NewFakeUserService() // No mock setup needed for this one!
			h := NewUserHandlers(fake, logger, tracer, nil, metrics)

			res, err := h.HandleTagUnavailable(context.Background(), tt.payload)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(res) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(res), tt.wantLen)
			}
			if res[0].Topic != tt.wantTopic {
				t.Errorf("got topic %s, want %s", res[0].Topic, tt.wantTopic)
			}
		})
	}
}
