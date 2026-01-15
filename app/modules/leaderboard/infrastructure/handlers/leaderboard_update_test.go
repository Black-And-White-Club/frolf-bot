package leaderboardhandlers

import (
	"context"
	"fmt"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboardmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleLeaderboardUpdateRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRoundID := sharedtypes.RoundID(uuid.New())
	testSortedParticipantTags := []string{
		"1:12345678901234567", // 1st place
		"2:12345678901234568", // 2nd place
	}

	testPayload := &leaderboardevents.LeaderboardUpdateRequestedPayloadV1{
		RoundID:               testRoundID,
		SortedParticipantTags: testSortedParticipantTags,
		Source:                "round",
		UpdateID:              testRoundID.String(),
	}

	mockLeaderboardService := leaderboardmocks.NewMockService(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &leaderboardmetrics.NoOpMetrics{}

	tests := []struct {
		name          string
		mockSetup     func()
		payload       *leaderboardevents.LeaderboardUpdateRequestedPayloadV1
		wantErr       bool
		wantResultLen int
		wantTopics    []string
	}{
		{
			name: "Successfully handle LeaderboardUpdateRequested",
			mockSetup: func() {
				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    sharedtypes.DiscordID("12345678901234567"),
						TagNumber: sharedtypes.TagNumber(1),
					},
					{
						UserID:    sharedtypes.DiscordID("12345678901234568"),
						TagNumber: sharedtypes.TagNumber(2),
					},
				}

				mockLeaderboardService.EXPECT().ExecuteBatchTagAssignment(
					gomock.Any(),
					gomock.Any(), // GuildID
					expectedRequests,
					testRoundID,
					sharedtypes.ServiceUpdateSourceProcessScores,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Leaderboard: []leaderboardtypes.LeaderboardEntry{
							{UserID: sharedtypes.DiscordID("12345678901234567"), TagNumber: sharedtypes.TagNumber(1)},
							{UserID: sharedtypes.DiscordID("12345678901234568"), TagNumber: sharedtypes.TagNumber(2)},
						},
					},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 2,
			wantTopics:    []string{leaderboardevents.LeaderboardUpdatedV1, leaderboardevents.TagUpdateForScheduledRoundsV1},
		},
		{
			name: "Invalid tag format - missing colon",
			mockSetup: func() {
				// Handler will call ExecuteBatchTagAssignment with empty requests; return an error to simulate validation
				mockLeaderboardService.EXPECT().ExecuteBatchTagAssignment(
					gomock.Any(),
					gomock.Any(),
					[]sharedtypes.TagAssignmentRequest{},
					testRoundID,
					sharedtypes.ServiceUpdateSourceProcessScores,
				).Return(leaderboardservice.LeaderboardOperationResult{}, fmt.Errorf("invalid tags"))
			},
			payload: &leaderboardevents.LeaderboardUpdateRequestedPayloadV1{
				RoundID:               testRoundID,
				SortedParticipantTags: []string{"12345678901234567"}, // Missing colon
				Source:                "round",
				UpdateID:              testRoundID.String(),
			},
			wantErr:       true,
			wantResultLen: 0,
		},
		{
			name: "Invalid tag number format",
			mockSetup: func() {
				// Handler will parse tag number into 0 on invalid format and call service with that request
				mockLeaderboardService.EXPECT().ExecuteBatchTagAssignment(
					gomock.Any(),
					gomock.Any(),
					[]sharedtypes.TagAssignmentRequest{{UserID: sharedtypes.DiscordID("12345678901234567"), TagNumber: sharedtypes.TagNumber(0)}},
					testRoundID,
					sharedtypes.ServiceUpdateSourceProcessScores,
				).Return(leaderboardservice.LeaderboardOperationResult{}, fmt.Errorf("invalid tag number"))
			},
			payload: &leaderboardevents.LeaderboardUpdateRequestedPayloadV1{
				RoundID:               testRoundID,
				SortedParticipantTags: []string{"invalid:12345678901234567"},
				Source:                "round",
				UpdateID:              testRoundID.String(),
			},
			wantErr:       true,
			wantResultLen: 0,
		},
		{
			name: "Service error in ProcessTagAssignments",
			mockSetup: func() {
				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    sharedtypes.DiscordID("12345678901234567"),
						TagNumber: sharedtypes.TagNumber(1),
					},
					{
						UserID:    sharedtypes.DiscordID("12345678901234568"),
						TagNumber: sharedtypes.TagNumber(2),
					},
				}

				mockLeaderboardService.EXPECT().ExecuteBatchTagAssignment(
					gomock.Any(),
					gomock.Any(), // GuildID
					expectedRequests,
					testRoundID,
					sharedtypes.ServiceUpdateSourceProcessScores,
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			payload:       testPayload,
			wantErr:       true,
			wantResultLen: 0,
		},
		{
			name: "Service failure with domain error payload",
			mockSetup: func() {
				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    sharedtypes.DiscordID("12345678901234567"),
						TagNumber: sharedtypes.TagNumber(1),
					},
					{
						UserID:    sharedtypes.DiscordID("12345678901234568"),
						TagNumber: sharedtypes.TagNumber(2),
					},
				}

				mockLeaderboardService.EXPECT().ExecuteBatchTagAssignment(
					gomock.Any(),
					gomock.Any(), // GuildID
					expectedRequests,
					testRoundID,
					sharedtypes.ServiceUpdateSourceProcessScores,
				).Return(
					leaderboardservice.LeaderboardOperationResult{
						Err: fmt.Errorf("tag assignment validation failed"),
					},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 2,
			wantTopics:    []string{leaderboardevents.LeaderboardUpdatedV1, sharedevents.TagUpdateForScheduledRoundsV1},
		},
		{
			name: "Unexpected service result - neither success nor failure",
			mockSetup: func() {
				expectedRequests := []sharedtypes.TagAssignmentRequest{
					{
						UserID:    sharedtypes.DiscordID("12345678901234567"),
						TagNumber: sharedtypes.TagNumber(1),
					},
					{
						UserID:    sharedtypes.DiscordID("12345678901234568"),
						TagNumber: sharedtypes.TagNumber(2),
					},
				}

				mockLeaderboardService.EXPECT().ExecuteBatchTagAssignment(
					gomock.Any(),
					gomock.Any(),
					expectedRequests,
					testRoundID,
					sharedtypes.ServiceUpdateSourceProcessScores,
				).Return(
					leaderboardservice.LeaderboardOperationResult{},
					nil,
				)
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 2,
			wantTopics:    []string{leaderboardevents.LeaderboardUpdatedV1, sharedevents.TagUpdateForScheduledRoundsV1},
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
			results, err := h.HandleLeaderboardUpdateRequested(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleLeaderboardUpdateRequested() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(results) != tt.wantResultLen {
				t.Errorf("HandleLeaderboardUpdateRequested() result length = %d, want %d", len(results), tt.wantResultLen)
			}

			if !tt.wantErr && len(tt.wantTopics) > 0 {
				for i, wantTopic := range tt.wantTopics {
					if i >= len(results) {
						t.Errorf("HandleLeaderboardUpdateRequested() missing result at index %d", i)
						continue
					}
					if results[i].Topic != wantTopic {
						t.Errorf("HandleLeaderboardUpdateRequested() result[%d].Topic = %s, want %s", i, results[i].Topic, wantTopic)
					}
				}
			}
		})
	}
}
