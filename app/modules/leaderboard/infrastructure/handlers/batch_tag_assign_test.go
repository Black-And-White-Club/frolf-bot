package leaderboardhandlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboarddomain "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/domain"
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
				f.ProcessRoundCommandFunc = func(ctx context.Context, cmd leaderboardservice.ProcessRoundCommand) (*leaderboardservice.ProcessRoundOutput, error) {
					return &leaderboardservice.ProcessRoundOutput{
						FinalParticipantTags: map[string]int{"user-1": 1},
						PointAwards: []leaderboarddomain.PointAward{
							{MemberID: "user-1", Points: 10},
						},
					}, nil
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
				f.ProcessRoundCommandFunc = func(ctx context.Context, cmd leaderboardservice.ProcessRoundCommand) (*leaderboardservice.ProcessRoundOutput, error) {
					return &leaderboardservice.ProcessRoundOutput{
						FinalParticipantTags: map[string]int{"user-1": 1},
						PointAwards: []leaderboarddomain.PointAward{
							{MemberID: "user-1", Points: 10},
						},
					}, nil
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
				f.ProcessRoundCommandFunc = func(ctx context.Context, cmd leaderboardservice.ProcessRoundCommand) (*leaderboardservice.ProcessRoundOutput, error) {
					return &leaderboardservice.ProcessRoundOutput{
						FinalParticipantTags: map[string]int{"user-1": 1},
						PointAwards: []leaderboarddomain.PointAward{
							{MemberID: "user-1", Points: 10},
						},
					}, nil
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
				f.ProcessRoundCommandFunc = func(ctx context.Context, cmd leaderboardservice.ProcessRoundCommand) (*leaderboardservice.ProcessRoundOutput, error) {
					return &leaderboardservice.ProcessRoundOutput{
						FinalParticipantTags: map[string]int{"user-1": 1},
						PointAwards: []leaderboarddomain.PointAward{
							{MemberID: "user-1", Points: 10},
						},
					}, nil
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

// TestHandleRoundBasedAssignment_LegacyFallbackUsesZeroNotSequential verifies that
// when assignments carry FinishRank == 0, the handler forwards FinishRank == 0 to
// the service (opting into the domain's "no rank" path) rather than assigning
// sequential i+1 positions that would silently break tied finishers.
func TestHandleRoundBasedAssignment_LegacyFallbackUsesZeroNotSequential(t *testing.T) {
	testGuildID := sharedtypes.GuildID("test-guild-fallback")
	testRoundID := sharedtypes.RoundID(uuid.New())

	var capturedParticipants []leaderboardservice.RoundParticipantInput

	fakeSvc := NewFakeService()
	fakeSvc.ProcessRoundCommandFunc = func(ctx context.Context, cmd leaderboardservice.ProcessRoundCommand) (*leaderboardservice.ProcessRoundOutput, error) {
		capturedParticipants = cmd.Participants
		return &leaderboardservice.ProcessRoundOutput{
			FinalParticipantTags: map[string]int{"user-a": 1, "user-b": 2},
			PointAwards:          []leaderboarddomain.PointAward{},
			PointsSkipped:        true,
		}, nil
	}

	h := &LeaderboardHandlers{
		service:     fakeSvc,
		userService: NewFakeUserService(),
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	// Both assignments have FinishRank == 0 (legacy / unset)
	payload := &sharedevents.BatchTagAssignmentRequestedPayloadV1{
		ScopedGuildID: sharedevents.ScopedGuildID{GuildID: testGuildID},
		BatchID:       uuid.New().String(),
		RoundID:       &testRoundID,
		Source:        sharedtypes.ServiceUpdateSourceProcessScores,
		Assignments: []sharedevents.TagAssignmentInfoV1{
			{UserID: "user-a", TagNumber: 1, FinishRank: 0},
			{UserID: "user-b", TagNumber: 2, FinishRank: 0},
		},
	}

	_, err := h.HandleBatchTagAssignmentRequested(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, p := range capturedParticipants {
		if p.FinishRank != 0 {
			t.Errorf("participant %s: expected FinishRank 0 (no-rank sentinel), got %d (sequential i+1 fallback is broken for ties)", p.MemberID, p.FinishRank)
		}
	}
}

func TestMapSuccessResults_TagRemovalBehavior(t *testing.T) {
	testGuildID := sharedtypes.GuildID("test-guild-123")
	testBatchID := uuid.New().String()

	tests := []struct {
		name            string
		requests        []sharedtypes.TagAssignmentRequest
		wantAssignCount int
		wantAssignLen   int
		wantChangedTags map[sharedtypes.DiscordID]sharedtypes.TagNumber
	}{
		{
			name: "Pure removal: tag=0 excluded from Assignments, present in ChangedTags",
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user-1", TagNumber: 0},
			},
			wantAssignCount: 1,
			wantAssignLen:   0,
			wantChangedTags: map[sharedtypes.DiscordID]sharedtypes.TagNumber{
				"user-1": 0,
			},
		},
		{
			name: "Mixed batch: only real assignments in Assignments, removal tracked in ChangedTags",
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user-1", TagNumber: 5},
				{UserID: "user-2", TagNumber: 0},
			},
			wantAssignCount: 2,
			wantAssignLen:   1,
			wantChangedTags: map[sharedtypes.DiscordID]sharedtypes.TagNumber{
				"user-1": 5,
				"user-2": 0,
			},
		},
		{
			name: "Pure assignment: all entries visible in Assignments",
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user-1", TagNumber: 3},
			},
			wantAssignCount: 1,
			wantAssignLen:   1,
			wantChangedTags: map[sharedtypes.DiscordID]sharedtypes.TagNumber{
				"user-1": 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &LeaderboardHandlers{}
			res := h.mapSuccessResults(
				testGuildID,
				"",
				testBatchID,
				tt.requests,
				sharedtypes.ServiceUpdateSourceTagSwap,
				"",
			)

			assert.Len(t, res, 2)

			batchPayload, ok := res[0].Payload.(*leaderboardevents.LeaderboardBatchTagAssignedPayloadV1)
			assert.True(t, ok, "first result should be LeaderboardBatchTagAssignedPayloadV1")
			assert.Equal(t, leaderboardevents.LeaderboardBatchTagAssignedV1, res[0].Topic)
			assert.Equal(t, tt.wantAssignCount, batchPayload.AssignmentCount)
			assert.Len(t, batchPayload.Assignments, tt.wantAssignLen)
			for _, a := range batchPayload.Assignments {
				assert.NotEqual(t, sharedtypes.TagNumber(0), a.TagNumber, "tag=0 must never appear in Assignments")
			}

			syncPayload, ok := res[1].Payload.(*sharedevents.SyncRoundsTagRequestPayloadV1)
			assert.True(t, ok, "second result should be SyncRoundsTagRequestPayloadV1")
			assert.Equal(t, sharedevents.SyncRoundsTagRequestV1, res[1].Topic)
			assert.Equal(t, tt.wantChangedTags, syncPayload.ChangedTags)
		})
	}
}
