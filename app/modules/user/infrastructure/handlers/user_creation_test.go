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
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestUserHandlers_HandleUserSignupRequest(t *testing.T) {
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("33333333333333333")
	testTagNumber := sharedtypes.TagNumber(1)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		payload        *userevents.UserSignupRequestedPayloadV1
		setupFake      func(*FakeUserService)
		wantLen        int
		wantTopic      string
		wantClubSync   bool
		wantErr        bool
	}{
		{
			name: "Successful signup with tag (Availability Check Flow)",
			payload: &userevents.UserSignupRequestedPayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: &testTagNumber,
			},
			setupFake: func(f *FakeUserService) {
				// No service call expected for this branch
			},
			wantLen:      1,
			wantTopic:    sharedevents.TagAvailabilityCheckRequestedV1,
			wantClubSync: false,
			wantErr:      false,
		},
		{
			name: "Successful signup with tag and guild metadata publishes club sync",
			payload: &userevents.UserSignupRequestedPayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: &testTagNumber,
				GuildName: "Test Guild",
			},
			setupFake: func(f *FakeUserService) {},
			wantLen:      2,
			wantTopic:    sharedevents.TagAvailabilityCheckRequestedV1,
			wantClubSync: true,
			wantErr:      false,
		},
		{
			name: "Successful signup without tag (Direct Creation Flow)",
			payload: &userevents.UserSignupRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
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
			wantLen:      1,
			wantTopic:    userevents.UserCreatedV1,
			wantClubSync: false,
			wantErr:      false,
		},
		{
			name: "Successful signup without tag with guild metadata publishes club sync",
			payload: &userevents.UserSignupRequestedPayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				GuildName: "Test Guild",
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
			wantLen:      2,
			wantTopic:    userevents.UserCreatedV1,
			wantClubSync: true,
			wantErr:      false,
		},
		{
			name: "Failed signup (Domain Failure)",
			payload: &userevents.UserSignupRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
			},
			setupFake: func(f *FakeUserService) {
				f.CreateUserFunc = func(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tag *sharedtypes.TagNumber, udiscUsername *string, udiscName *string) (userservice.UserResult, error) {
					return results.FailureResult[*userservice.CreateUserResponse, error](errors.New("already exists")), nil
				}
			},
			wantLen:      1,
			wantTopic:    userevents.UserCreationFailedV1,
			wantClubSync: false,
			wantErr:      false,
		},
		{
			name: "Error from CreateUser (Infrastructure Failure)",
			payload: &userevents.UserSignupRequestedPayloadV1{
				GuildID: testGuildID,
				UserID:  testUserID,
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
			res, err := h.HandleUserSignupRequest(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUserSignupRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				assert.Len(t, res, tt.wantLen)
				if len(res) > 0 {
					assert.Equal(t, tt.wantTopic, res[0].Topic)
				}
				if tt.wantClubSync {
					lastResult := res[len(res)-1]
					assert.Equal(t, sharedevents.ClubSyncFromDiscordRequestedV1, lastResult.Topic)
					syncPayload, ok := lastResult.Payload.(*sharedevents.ClubSyncFromDiscordRequestedPayloadV1)
					assert.True(t, ok)
					assert.Equal(t, tt.payload.GuildID, syncPayload.GuildID)
					assert.Equal(t, tt.payload.GuildName, syncPayload.GuildName)
				}
			}
		})
	}
}
