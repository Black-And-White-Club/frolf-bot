package leaderboardhandlers

import (
	"context"
	"fmt"
	"testing"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application/mocks"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleTagAvailabilityCheckRequested(t *testing.T) {
	testGuildID := sharedtypes.GuildID("test-guild-123")
	testUserID := sharedtypes.DiscordID("user-456")
	testTagNumber := sharedtypes.TagNumber(7)

	tests := []struct {
		name       string
		payload    *sharedevents.TagAvailabilityCheckRequestedPayloadV1
		mockSetup  func(*leaderboardmocks.MockService)
		wantErr    bool
		wantTopic  string
		validateFn func(t *testing.T, payload any)
	}{
		{
			name: "tag number missing",
			payload: &sharedevents.TagAvailabilityCheckRequestedPayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: nil,
			},
			mockSetup: func(mockLeaderboardService *leaderboardmocks.MockService) {
				mockLeaderboardService.EXPECT().CheckTagAvailability(
					gomock.Any(),
					testGuildID,
					testUserID,
					(*sharedtypes.TagNumber)(nil),
				).Return(
					sharedevents.TagAvailabilityCheckResultPayloadV1{},
					&sharedevents.TagAvailabilityCheckFailedPayloadV1{
						GuildID:   testGuildID,
						UserID:    testUserID,
						TagNumber: nil,
						Reason:    "tag number is required",
					},
					nil,
				)
			},
			wantErr:   false,
			wantTopic: sharedevents.TagAvailabilityCheckFailedV1,
			validateFn: func(t *testing.T, payload any) {
				failed, ok := payload.(*sharedevents.TagAvailabilityCheckFailedPayloadV1)
				if !ok {
					t.Fatalf("expected failure payload type, got %T", payload)
				}
				if failed.Reason != "tag number is required" {
					t.Errorf("expected reason %q, got %q", "tag number is required", failed.Reason)
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
			mockSetup: func(mockLeaderboardService *leaderboardmocks.MockService) {
				mockLeaderboardService.EXPECT().CheckTagAvailability(
					gomock.Any(),
					testGuildID,
					testUserID,
					&testTagNumber,
				).Return(
					sharedevents.TagAvailabilityCheckResultPayloadV1{},
					nil,
					fmt.Errorf("db error"),
				)
			},
			wantErr: true,
		},
		{
			name: "tag available",
			payload: &sharedevents.TagAvailabilityCheckRequestedPayloadV1{
				GuildID:   testGuildID,
				UserID:    testUserID,
				TagNumber: &testTagNumber,
			},
			mockSetup: func(mockLeaderboardService *leaderboardmocks.MockService) {
				mockLeaderboardService.EXPECT().CheckTagAvailability(
					gomock.Any(),
					testGuildID,
					testUserID,
					&testTagNumber,
				).Return(
					sharedevents.TagAvailabilityCheckResultPayloadV1{
						GuildID:   testGuildID,
						UserID:    testUserID,
						TagNumber: &testTagNumber,
						Available: true,
					},
					nil,
					nil,
				)
			},
			wantErr:   false,
			wantTopic: sharedevents.TagAvailableV1,
			validateFn: func(t *testing.T, payload any) {
				available, ok := payload.(*sharedevents.TagAvailablePayloadV1)
				if !ok {
					t.Fatalf("expected available payload type, got %T", payload)
				}
				if available.TagNumber != testTagNumber {
					t.Errorf("expected tag %d, got %d", testTagNumber, available.TagNumber)
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
			mockSetup: func(mockLeaderboardService *leaderboardmocks.MockService) {
				mockLeaderboardService.EXPECT().CheckTagAvailability(
					gomock.Any(),
					testGuildID,
					testUserID,
					&testTagNumber,
				).Return(
					sharedevents.TagAvailabilityCheckResultPayloadV1{
						GuildID:   testGuildID,
						UserID:    testUserID,
						TagNumber: &testTagNumber,
						Available: false,
						Reason:    "tag already taken",
					},
					nil,
					nil,
				)
			},
			wantErr:   false,
			wantTopic: sharedevents.TagUnavailableV1,
			validateFn: func(t *testing.T, payload any) {
				unavailable, ok := payload.(*sharedevents.TagUnavailablePayloadV1)
				if !ok {
					t.Fatalf("expected unavailable payload type, got %T", payload)
				}
				if unavailable.Reason != "tag already taken" {
					t.Errorf("expected reason %q, got %q", "tag already taken", unavailable.Reason)
				}
				if unavailable.TagNumber != testTagNumber {
					t.Errorf("expected tag %d, got %d", testTagNumber, unavailable.TagNumber)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockLeaderboardService := leaderboardmocks.NewMockService(ctrl)

			if tt.mockSetup != nil {
				tt.mockSetup(mockLeaderboardService)
			}

			h := &LeaderboardHandlers{
				service: mockLeaderboardService,
			}

			ctx := context.Background()
			results, err := h.HandleTagAvailabilityCheckRequested(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagAvailabilityCheckRequested() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}

			if results[0].Topic != tt.wantTopic {
				t.Errorf("expected topic %s, got %s", tt.wantTopic, results[0].Topic)
			}

			if tt.validateFn != nil {
				tt.validateFn(t, results[0].Payload)
			}
		})
	}
}
