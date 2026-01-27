package userhandlers

import (
	"context"
	"errors"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestUserHandlers_HandleUpdateUDiscIdentityRequest(t *testing.T) {
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("33333333333333333")
	testUsername := "testuser"
	testName := "Test User"

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	tests := []struct {
		name      string
		payload   *userevents.UpdateUDiscIdentityRequestedPayloadV1
		setupFake func(*FakeUserService)
		wantLen   int
		wantTopic string
		wantErr   bool
	}{
		{
			name: "Successful update",
			payload: &userevents.UpdateUDiscIdentityRequestedPayloadV1{
				GuildID:  testGuildID,
				UserID:   testUserID,
				Username: &testUsername,
				Name:     &testName,
			},
			setupFake: func(f *FakeUserService) {
				f.UpdateUDiscIdentityFunc = func(ctx context.Context, userID sharedtypes.DiscordID, username *string, name *string) (results.OperationResult[bool, error], error) {
					// Service returns bool success
					return results.SuccessResult[bool, error](true), nil
				}
			},
			wantLen:   1,
			wantTopic: userevents.UDiscIdentityUpdatedV1,
			wantErr:   false,
		},
		{
			name: "Update failed (business logic failure)",
			payload: &userevents.UpdateUDiscIdentityRequestedPayloadV1{
				GuildID:  testGuildID,
				UserID:   testUserID,
				Username: &testUsername,
				Name:     &testName,
			},
			setupFake: func(f *FakeUserService) {
				f.UpdateUDiscIdentityFunc = func(ctx context.Context, userID sharedtypes.DiscordID, username *string, name *string) (results.OperationResult[bool, error], error) {
					// Service returns a domain failure
					return results.FailureResult[bool, error](errors.New("identity conflict")), nil
				}
			},
			wantLen:   1,
			wantTopic: userevents.UDiscIdentityUpdateFailedV1,
			wantErr:   false,
		},
		{
			name: "Infrastructure error",
			payload: &userevents.UpdateUDiscIdentityRequestedPayloadV1{
				UserID: testUserID,
			},
			setupFake: func(f *FakeUserService) {
				f.UpdateUDiscIdentityFunc = func(ctx context.Context, userID sharedtypes.DiscordID, username *string, name *string) (results.OperationResult[bool, error], error) {
					return results.OperationResult[bool, error]{}, context.DeadlineExceeded
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
			res, err := h.HandleUpdateUDiscIdentityRequest(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUpdateUDiscIdentityRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(res) != tt.wantLen {
					t.Errorf("got %d results, want %d", len(res), tt.wantLen)
					return
				}
				if tt.wantLen > 0 && res[0].Topic != tt.wantTopic {
					t.Errorf("got topic %s, want %s", res[0].Topic, tt.wantTopic)
				}
			}
		})
	}
}
