package leaderboardhandlers

import (
	"context"
	"fmt"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleTagAvailabilityCheckRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardService := leaderboardmocks.NewMockService(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &leaderboardmetrics.NoOpMetrics{}

	testGuildID := sharedtypes.GuildID("test-guild-123")
	testUserID := sharedtypes.DiscordID("user-456")
	testTagNum := sharedtypes.TagNumber(1)
	testTagNumber := &testTagNum
	testPayload := &leaderboardevents.TagAvailabilityCheckRequestedPayloadV1{
		GuildID:   testGuildID,
		UserID:    testUserID,
		TagNumber: testTagNumber,
	}

	tests := []struct {
		name          string
		mockSetup     func()
		payload       *leaderboardevents.TagAvailabilityCheckRequestedPayloadV1
		wantErr       bool
		wantResultLen int
	}{
		{
			name: "Tag is available",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().CheckTagAvailability(
					gomock.Any(),
					testGuildID,
					*testPayload,
				).Return(
					&leaderboardevents.TagAvailabilityCheckResultPayloadV1{
						GuildID:   testGuildID,
						UserID:    testUserID,
						TagNumber: testTagNumber,
						Available: true,
					},
					nil,
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 2, // TagAvailable + BatchTagAssignmentRequested
		},
		{
			name: "Tag is unavailable",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().CheckTagAvailability(
					gomock.Any(),
					testGuildID,
					*testPayload,
				).Return(
					&leaderboardevents.TagAvailabilityCheckResultPayloadV1{
						GuildID:   testGuildID,
						UserID:    testUserID,
						TagNumber: testTagNumber,
						Available: false,
						Reason:    "tag already taken",
					},
					nil,
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1, // TagUnavailable only
		},
		{
			name: "Availability check failed",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().CheckTagAvailability(
					gomock.Any(),
					testGuildID,
					*testPayload,
				).Return(
					nil,
					&leaderboardevents.TagAvailabilityCheckFailedPayloadV1{
						GuildID: testGuildID,
						Reason:  "database error",
					},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1, // TagAvailabilityCheckFailed only
		},
		{
			name: "Service error",
			mockSetup: func() {
				mockLeaderboardService.EXPECT().CheckTagAvailability(
					gomock.Any(),
					testGuildID,
					*testPayload,
				).Return(
					nil,
					nil,
					fmt.Errorf("service error"),
				)
			},
			payload:       testPayload,
			wantErr:       true,
			wantResultLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := &LeaderboardHandlers{
				leaderboardService: mockLeaderboardService,
				logger:             logger,
				tracer:             tracer,
				metrics:            metrics,
			}

			ctx := context.Background()
			results, err := h.HandleTagAvailabilityCheckRequested(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagAvailabilityCheckRequested() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(results) != tt.wantResultLen {
				t.Errorf("HandleTagAvailabilityCheckRequested() result length = %d, want %d", len(results), tt.wantResultLen)
			}

			// Verify result topics
			if !tt.wantErr && tt.wantResultLen > 0 {
				expectedTopics := []string{}
				if tt.name == "Tag is available" {
					expectedTopics = []string{userevents.TagAvailableV1, sharedevents.LeaderboardBatchTagAssignmentRequestedV1}
				} else if tt.name == "Tag is unavailable" {
					expectedTopics = []string{userevents.TagUnavailableV1}
				} else if tt.name == "Availability check failed" {
					expectedTopics = []string{leaderboardevents.TagAvailabilityCheckFailedV1}
				}

				for i, expectedTopic := range expectedTopics {
					if i >= len(results) {
						t.Errorf("HandleTagAvailabilityCheckRequested() missing result at index %d", i)
						continue
					}
					if results[i].Topic != expectedTopic {
						t.Errorf("HandleTagAvailabilityCheckRequested() result[%d].Topic = %s, want %s", i, results[i].Topic, expectedTopic)
					}
				}
			}
		})
	}
}
