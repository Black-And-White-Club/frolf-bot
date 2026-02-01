package leaderboardhandlers

import (
	"context"
	"fmt"
	"testing"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewLeaderboardHandlers(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Creates handlers with all dependencies",
			test: func(t *testing.T) {
				fakeService := NewFakeService()
				fakeUserService := NewFakeUserService()
				fakeSaga := NewFakeSagaCoordinator()
				tracer := noop.NewTracerProvider().Tracer("test")
				fakeHelpers := &FakeHelpers{}
				noOpMetrics := &leaderboardmetrics.NoOpMetrics{}

				// Call the constructor
				handlers := NewLeaderboardHandlers(fakeService, fakeUserService, fakeSaga, nil, tracer, fakeHelpers, noOpMetrics)

				if handlers == nil {
					t.Fatalf("NewLeaderboardHandlers returned nil")
				}

				// Type assert to access internal fields for verification
				lbHandlers, ok := handlers.(*LeaderboardHandlers)
				if !ok {
					t.Fatalf("returned object is not *LeaderboardHandlers")
				}

				if lbHandlers.service != fakeService {
					t.Errorf("service not correctly assigned")
				}
				if lbHandlers.userService != fakeUserService {
					t.Errorf("userService not correctly assigned")
				}
				if lbHandlers.sagaCoordinator != fakeSaga {
					t.Errorf("saga coordinator not correctly assigned")
				}
				if lbHandlers.helpers != fakeHelpers {
					t.Errorf("helpers not correctly assigned")
				}
			},
		},
		{
			name: "Handles nil dependencies",
			test: func(t *testing.T) {
				handlers := NewLeaderboardHandlers(nil, nil, nil, nil, nil, nil, nil)

				if handlers == nil {
					t.Fatalf("NewLeaderboardHandlers returned nil")
				}

				lbHandlers := handlers.(*LeaderboardHandlers)
				if lbHandlers.service != nil {
					t.Errorf("service should be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestLeaderboardHandlers_HandleGuildConfigCreated(t *testing.T) {
	testGuildID := sharedtypes.GuildID("guild-123")
	testPayload := &guildevents.GuildConfigCreatedPayloadV1{
		GuildID: testGuildID,
	}

	tests := []struct {
		name      string
		setupFake func(f *FakeService)
		wantErr   bool
	}{
		{
			name: "Successfully ensures leaderboard",
			setupFake: func(f *FakeService) {
				f.EnsureGuildLeaderboardFunc = func(ctx context.Context, g sharedtypes.GuildID) (results.OperationResult[bool, error], error) {
					if g != testGuildID {
						return results.OperationResult[bool, error]{}, fmt.Errorf("wrong guild ID")
					}
					return results.SuccessResult[bool, error](true), nil
				}
			},
			wantErr: false,
		},
		{
			name: "Service error bubbles up",
			setupFake: func(f *FakeService) {
				f.EnsureGuildLeaderboardFunc = func(ctx context.Context, g sharedtypes.GuildID) (results.OperationResult[bool, error], error) {
					return results.OperationResult[bool, error]{}, fmt.Errorf("infrastructure failure")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeService()
			tt.setupFake(fakeSvc)

			h := &LeaderboardHandlers{
				service: fakeSvc,
			}

			res, err := h.HandleGuildConfigCreated(context.Background(), testPayload)

			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr %v, got %v", tt.wantErr, err)
			}

			// This handler returns an empty slice of results on success
			if !tt.wantErr && len(res) != 0 {
				t.Errorf("expected 0 results, got %d", len(res))
			}
		})
	}
}
