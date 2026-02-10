package leaderboardhandlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestLeaderboardHandlers_HandleBatchTagAssignmentRequested(t *testing.T) {
	testGuildID := sharedtypes.GuildID("test-guild-123")
	testBatchID := uuid.New().String()

	tests := []struct {
		name          string
		setupFake     func(f *FakeService)
		wantErr       bool
		wantResultLen int
	}{
		{
			name: "Successfully assign batch tags",
			setupFake: func(f *FakeService) {
				f.ExecuteBatchTagAssignmentFunc = func(ctx context.Context, guildID sharedtypes.GuildID, requests []sharedtypes.TagAssignmentRequest, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
					// Service returns domain data (the slice of new assignments)
					return results.SuccessResult[leaderboardtypes.LeaderboardData, error](leaderboardtypes.LeaderboardData{
						{UserID: "user-456", TagNumber: 1},
					}), nil
				}
			},
			wantErr: false,
			// mapSuccessResults likely produces 1 'batch_assigned' event + 1 'tag_updated' event per assignment
			wantResultLen: 2,
		},
		{
			name: "Service Infrastructure Error",
			setupFake: func(f *FakeService) {
				f.ExecuteBatchTagAssignmentFunc = func(ctx context.Context, guildID sharedtypes.GuildID, requests []sharedtypes.TagAssignmentRequest, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
					return results.OperationResult[leaderboardtypes.LeaderboardData, error]{}, fmt.Errorf("connection refused")
				}
			},
			wantErr:       true,
			wantResultLen: 0,
		},
		{
			name: "Tag Swap Required Error",
			setupFake: func(f *FakeService) {
				f.ExecuteBatchTagAssignmentFunc = func(ctx context.Context, guildID sharedtypes.GuildID, requests []sharedtypes.TagAssignmentRequest, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
					err := error(&leaderboardservice.TagSwapNeededError{
						RequestorID: "user-456",
						CurrentTag:  5,
						TargetTag:   1,
					})
					return results.FailureResult[leaderboardtypes.LeaderboardData, error](err), nil
				}
			},
			wantErr:       false, // Should return empty results and hand off to saga
			wantResultLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeService()
			fakeSaga := NewFakeSagaCoordinator()
			tt.setupFake(fakeSvc)

			h := &LeaderboardHandlers{
				service:         fakeSvc,
				userService:     NewFakeUserService(),
				sagaCoordinator: fakeSaga,
				logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
			}

			payload := &sharedevents.BatchTagAssignmentRequestedPayloadV1{
				ScopedGuildID: sharedevents.ScopedGuildID{GuildID: testGuildID},
				BatchID:       testBatchID,
				Assignments:   []sharedevents.TagAssignmentInfoV1{{UserID: "user-456", TagNumber: 1}},
			}

			res, err := h.HandleBatchTagAssignmentRequested(context.Background(), payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr %v, got %v", tt.wantErr, err)
			}

			if len(res) != tt.wantResultLen {
				t.Errorf("got %d results, want %d", len(res), tt.wantResultLen)
			}

			// Verification for the Saga flow
			if tt.name == "Tag Swap Required Error" {
				if len(fakeSaga.Trace()) == 0 {
					t.Error("Expected saga coordinator to be called but trace is empty")
				}
			}
		})
	}
}

func TestLeaderboardHandlers_HandleRoundBasedAssignment(t *testing.T) {
	testGuildID := sharedtypes.GuildID("test-guild-123")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testBatchID := uuid.New().String()

	tests := []struct {
		name             string
		setupFake        func(f *FakeService)
		setupRoundLookup func(rl *FakeRoundLookup)
		payload          *sharedevents.BatchTagAssignmentRequestedPayloadV1
		wantErr          bool
		validate         func(t *testing.T, res []handlerwrapper.Result)
	}{
		{
			name: "Enrich PointsAwarded event with full round data",
			payload: &sharedevents.BatchTagAssignmentRequestedPayloadV1{
				ScopedGuildID: sharedevents.ScopedGuildID{GuildID: testGuildID},
				BatchID:       testBatchID,
				RoundID:       &testRoundID,
				Source:        sharedtypes.ServiceUpdateSourceProcessScores,
				Assignments:   []sharedevents.TagAssignmentInfoV1{{UserID: "user-1", TagNumber: 1}},
			},
			setupFake: func(f *FakeService) {
				f.ProcessRoundFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, playerResults []leaderboardservice.PlayerResult, source sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardservice.ProcessRoundResult, error], error) {
					return results.SuccessResult[leaderboardservice.ProcessRoundResult, error](leaderboardservice.ProcessRoundResult{
						LeaderboardData: leaderboardtypes.LeaderboardData{{UserID: "user-1", TagNumber: 1}},
						PointsAwarded:   map[sharedtypes.DiscordID]int{"user-1": 10},
					}), nil
				}
			},
			setupRoundLookup: func(rl *FakeRoundLookup) {
				rl.GetRoundFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						EventMessageID: "msg-123",
						Title:          "Enriched Round",
						Participants:   []roundtypes.Participant{{UserID: "user-1", Points: nil}}, // Points nil initially
						Teams:          []roundtypes.NormalizedTeam{{TeamID: uuid.New(), Total: 50}},
					}, nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, res []handlerwrapper.Result) {
				// Expect 3 events: 2 from mapSuccessResults (batch assigned + sync request) + 1 PointsAwarded
				assert.Len(t, res, 3)

				// Find PointsAwarded event
				var pointsEvent *sharedevents.PointsAwardedPayloadV1
				for _, r := range res {
					if r.Topic == sharedevents.PointsAwardedV1 {
						pointsEvent = r.Payload.(*sharedevents.PointsAwardedPayloadV1)
						// Check metadata
						assert.Equal(t, "msg-123", r.Metadata["discord_message_id"])
					}
				}
				assert.NotNil(t, pointsEvent)
				assert.Equal(t, "msg-123", pointsEvent.EventMessageID)
				assert.Equal(t, roundtypes.Title("Enriched Round"), pointsEvent.Title)
				assert.Len(t, pointsEvent.Teams, 1)
				assert.Len(t, pointsEvent.Participants, 1)
				// Verify point merging
				assert.NotNil(t, pointsEvent.Participants[0].Points)
				assert.Equal(t, 10, *pointsEvent.Participants[0].Points)
			},
		},
		{
			name: "Round lookup failure - warns but continues without enrichment",
			payload: &sharedevents.BatchTagAssignmentRequestedPayloadV1{
				ScopedGuildID: sharedevents.ScopedGuildID{GuildID: testGuildID},
				BatchID:       testBatchID,
				RoundID:       &testRoundID,
				Source:        sharedtypes.ServiceUpdateSourceProcessScores,
				Assignments:   []sharedevents.TagAssignmentInfoV1{{UserID: "user-1", TagNumber: 1}},
			},
			setupFake: func(f *FakeService) {
				f.ProcessRoundFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, playerResults []leaderboardservice.PlayerResult, source sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardservice.ProcessRoundResult, error], error) {
					return results.SuccessResult[leaderboardservice.ProcessRoundResult, error](leaderboardservice.ProcessRoundResult{
						LeaderboardData: leaderboardtypes.LeaderboardData{{UserID: "user-1", TagNumber: 1}},
						PointsAwarded:   map[sharedtypes.DiscordID]int{"user-1": 10},
					}), nil
				}
			},
			setupRoundLookup: func(rl *FakeRoundLookup) {
				rl.GetRoundFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return nil, fmt.Errorf("db error")
				}
			},
			wantErr: false,
			validate: func(t *testing.T, res []handlerwrapper.Result) {
				var pointsEvent *sharedevents.PointsAwardedPayloadV1
				for _, r := range res {
					if r.Topic == sharedevents.PointsAwardedV1 {
						pointsEvent = r.Payload.(*sharedevents.PointsAwardedPayloadV1)
						assert.Empty(t, r.Metadata["discord_message_id"])
					}
				}
				assert.NotNil(t, pointsEvent)
				assert.Empty(t, pointsEvent.EventMessageID)
				assert.Empty(t, pointsEvent.Title)
			},
		},
		{
			name: "Nil RoundLookup - skips enrichment",
			payload: &sharedevents.BatchTagAssignmentRequestedPayloadV1{
				ScopedGuildID: sharedevents.ScopedGuildID{GuildID: testGuildID},
				BatchID:       testBatchID,
				RoundID:       &testRoundID,
				Source:        sharedtypes.ServiceUpdateSourceProcessScores,
				Assignments:   []sharedevents.TagAssignmentInfoV1{{UserID: "user-1", TagNumber: 1}},
			},
			setupFake: func(f *FakeService) {
				f.ProcessRoundFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, playerResults []leaderboardservice.PlayerResult, source sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardservice.ProcessRoundResult, error], error) {
					return results.SuccessResult[leaderboardservice.ProcessRoundResult, error](leaderboardservice.ProcessRoundResult{
						LeaderboardData: leaderboardtypes.LeaderboardData{{UserID: "user-1", TagNumber: 1}},
						PointsAwarded:   map[sharedtypes.DiscordID]int{"user-1": 10},
					}), nil
				}
			},
			setupRoundLookup: nil, // This will result in h.roundLookup being nil
			wantErr:          false,
			validate: func(t *testing.T, res []handlerwrapper.Result) {
				var pointsEvent *sharedevents.PointsAwardedPayloadV1
				for _, r := range res {
					if r.Topic == sharedevents.PointsAwardedV1 {
						pointsEvent = r.Payload.(*sharedevents.PointsAwardedPayloadV1)
					}
				}
				assert.NotNil(t, pointsEvent)
				assert.Empty(t, pointsEvent.EventMessageID)
				assert.Empty(t, pointsEvent.Title)
			},
		},
		{
			name: "Round returned as nil - skips enrichment",
			payload: &sharedevents.BatchTagAssignmentRequestedPayloadV1{
				ScopedGuildID: sharedevents.ScopedGuildID{GuildID: testGuildID},
				BatchID:       testBatchID,
				RoundID:       &testRoundID,
				Source:        sharedtypes.ServiceUpdateSourceProcessScores,
				Assignments:   []sharedevents.TagAssignmentInfoV1{{UserID: "user-1", TagNumber: 1}},
			},
			setupFake: func(f *FakeService) {
				f.ProcessRoundFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, playerResults []leaderboardservice.PlayerResult, source sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardservice.ProcessRoundResult, error], error) {
					return results.SuccessResult[leaderboardservice.ProcessRoundResult, error](leaderboardservice.ProcessRoundResult{
						LeaderboardData: leaderboardtypes.LeaderboardData{{UserID: "user-1", TagNumber: 1}},
						PointsAwarded:   map[sharedtypes.DiscordID]int{"user-1": 10},
					}), nil
				}
			},
			setupRoundLookup: func(rl *FakeRoundLookup) {
				rl.GetRoundFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return nil, nil // Nil round, no error
				}
			},
			wantErr: false,
			validate: func(t *testing.T, res []handlerwrapper.Result) {
				var pointsEvent *sharedevents.PointsAwardedPayloadV1
				for _, r := range res {
					if r.Topic == sharedevents.PointsAwardedV1 {
						pointsEvent = r.Payload.(*sharedevents.PointsAwardedPayloadV1)
					}
				}
				assert.NotNil(t, pointsEvent)
				assert.Empty(t, pointsEvent.EventMessageID)
				assert.Empty(t, pointsEvent.Title)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeService()
			fakeSaga := NewFakeSagaCoordinator()
			var roundLookup RoundLookup
			if tt.setupRoundLookup != nil {
				fakeRoundLookup := &FakeRoundLookup{}
				tt.setupRoundLookup(fakeRoundLookup)
				roundLookup = fakeRoundLookup
			}
			if tt.setupFake != nil {
				tt.setupFake(fakeSvc)
			}

			h := &LeaderboardHandlers{
				service:         fakeSvc,
				userService:     NewFakeUserService(),
				sagaCoordinator: fakeSaga,
				roundLookup:     roundLookup, // True nil interface if not set
				logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
			}

			res, err := h.HandleBatchTagAssignmentRequested(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr %v, got %v", tt.wantErr, err)
			}

			if tt.validate != nil {
				tt.validate(t, res)
			}
		})
	}
}
