package leaderboardhandlers

import (
	"context"
	"fmt"
	"testing"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
)

func TestLeaderboardHandlers_HandleTagAvailabilityCheckRequested(t *testing.T) {
	testGuildID := sharedtypes.GuildID("test-guild-123")
	testUserID := sharedtypes.DiscordID("user-456")
	testTagNumber := sharedtypes.TagNumber(7)

	tests := []struct {
		name       string
		payload    *sharedevents.TagAvailabilityCheckRequestedPayloadV1
		setupFake  func(*FakeService)
		wantErr    bool
		wantTopic  string
		validateFn func(t *testing.T, payload any)
	}{
		{
			name: "tag available",
			payload: &sharedevents.TagAvailabilityCheckRequestedPayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: &testTagNumber,
			},
			setupFake: func(f *FakeService) {
				f.CheckTagAvailabilityFunc = func(ctx context.Context, g sharedtypes.GuildID, u sharedtypes.DiscordID, tn sharedtypes.TagNumber) (results.OperationResult[leaderboardservice.TagAvailabilityResult, error], error) {
					return results.SuccessResult[leaderboardservice.TagAvailabilityResult, error](leaderboardservice.TagAvailabilityResult{
						Available: true,
					}), nil
				}
			},
			wantErr:   false,
			wantTopic: sharedevents.TagAvailableV1,
			validateFn: func(t *testing.T, payload any) {
				p, ok := payload.(*sharedevents.TagAvailablePayloadV1)
				if !ok {
					t.Fatalf("expected TagAvailablePayloadV1, got %T", payload)
				}
				if p.TagNumber != testTagNumber {
					t.Errorf("expected tag %d, got %d", testTagNumber, p.TagNumber)
				}
			},
		},
		{
			name: "tag unavailable",
			payload: &sharedevents.TagAvailabilityCheckRequestedPayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: &testTagNumber,
			},
			setupFake: func(f *FakeService) {
				f.CheckTagAvailabilityFunc = func(ctx context.Context, g sharedtypes.GuildID, u sharedtypes.DiscordID, tn sharedtypes.TagNumber) (results.OperationResult[leaderboardservice.TagAvailabilityResult, error], error) {
					return results.SuccessResult[leaderboardservice.TagAvailabilityResult, error](leaderboardservice.TagAvailabilityResult{
						Available: false,
						Reason:    "tag already taken",
					}), nil
				}
			},
			wantErr:   false,
			wantTopic: sharedevents.TagUnavailableV1,
			validateFn: func(t *testing.T, payload any) {
				p, ok := payload.(*sharedevents.TagUnavailablePayloadV1)
				if !ok {
					t.Fatalf("expected TagUnavailablePayloadV1, got %T", payload)
				}
				if p.Reason != "tag already taken" {
					t.Errorf("expected reason 'tag already taken', got %q", p.Reason)
				}
			},
		},
		{
			name: "service error",
			payload: &sharedevents.TagAvailabilityCheckRequestedPayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: &testTagNumber,
			},
			setupFake: func(f *FakeService) {
				f.CheckTagAvailabilityFunc = func(ctx context.Context, g sharedtypes.GuildID, u sharedtypes.DiscordID, tn sharedtypes.TagNumber) (results.OperationResult[leaderboardservice.TagAvailabilityResult, error], error) {
					return results.OperationResult[leaderboardservice.TagAvailabilityResult, error]{}, fmt.Errorf("internal database error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeService()
			if tt.setupFake != nil {
				tt.setupFake(fakeSvc)
			}

			h := &LeaderboardHandlers{
				service:     fakeSvc,
				userService: NewFakeUserService(),
			}

			res, err := h.HandleTagAvailabilityCheckRequested(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr %v, got %v", tt.wantErr, err)
				return
			}

			if tt.wantErr {
				return
			}

			if len(res) != 1 {
				t.Fatalf("expected 1 result, got %d", len(res))
			}

			if res[0].Topic != tt.wantTopic {
				t.Errorf("expected topic %s, got %s", tt.wantTopic, res[0].Topic)
			}

			if tt.validateFn != nil {
				tt.validateFn(t, res[0].Payload)
			}
		})
	}
}
