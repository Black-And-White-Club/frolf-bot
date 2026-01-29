package roundservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestRoundService_ApplyImportedScores(t *testing.T) {
	// Helper for creating pointers
	ptrScore := func(i int) *sharedtypes.Score {
		s := sharedtypes.Score(i)
		return &s
	}

	// Test data
	gID := sharedtypes.GuildID("guild-123")
	rID := sharedtypes.RoundID(uuid.New())
	importID := "import-abc"
	u1 := sharedtypes.DiscordID("user-1")
	u2 := sharedtypes.DiscordID("user-2")
	teamID := uuid.New()

	type testCase struct {
		name      string
		input     roundtypes.ImportApplyScoresInput
		setupFake func(*FakeRepo)
		verify    func(*testing.T, ApplyImportedScoresResult, error)
	}

	tests := []testCase{
		{
			name: "Singles - Success (Update existing + Add new)",
			input: roundtypes.ImportApplyScoresInput{
				GuildID:  gID,
				RoundID:  rID,
				ImportID: importID,
				Scores: []roundtypes.ImportScoreData{
					{UserID: u1, Score: 50, RawName: "User 1"}, // Existing
					{UserID: u2, Score: 52, RawName: "User 2"}, // New
				},
			},
			setupFake: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:             rID,
						EventMessageID: "msg-123",
						Teams:          nil, // Singles
					}, nil
				}
				r.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					// User 1 exists with old score
					return []roundtypes.Participant{
						{UserID: u1, Score: ptrScore(60), Response: roundtypes.ResponseAccept},
					}, nil
				}
				r.UpdateRoundsAndParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error {
					if len(updates) != 1 {
						return errors.New("expected 1 update")
					}
					parts := updates[0].Participants
					if len(parts) != 2 {
						return errors.New("expected 2 participants")
					}
					// Verify User 1 updated
					if parts[0].UserID == u1 && *parts[0].Score != 50 {
						return errors.New("user 1 score not updated")
					}
					// Verify User 2 added
					if parts[1].UserID == u2 && *parts[1].Score != 52 {
						return errors.New("user 2 score not added")
					}
					return nil
				}
			},
			verify: func(t *testing.T, res ApplyImportedScoresResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.IsFailure() {
					t.Fatalf("unexpected failure: %v", (*res.Failure).Error())
				}
				success := res.Success
				if (*success).EventMessageID != "msg-123" {
					t.Errorf("expected EventMessageID msg-123, got %s", (*success).EventMessageID)
				}
				if len((*success).Participants) != 2 {
					t.Errorf("expected 2 participants in result, got %d", len((*success).Participants))
				}
			},
		},
		{
			name: "Singles - Failure (No scores applied - Guest only)",
			input: roundtypes.ImportApplyScoresInput{
				GuildID:  gID,
				RoundID:  rID,
				ImportID: importID,
				Scores: []roundtypes.ImportScoreData{
					{UserID: "", Score: 50, RawName: "Guest"}, // Guests skipped in singles
				},
			},
			setupFake: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: rID}, nil
				}
				r.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{}, nil
				}
			},
			verify: func(t *testing.T, res ApplyImportedScoresResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !res.IsFailure() {
					t.Fatal("expected failure result")
				}
				errMsg := (*res.Failure).Error()
				if errMsg != "no scores were successfully applied" {
					t.Errorf("unexpected error message: %s", errMsg)
				}
			},
		},
		{
			name: "Teams - Success (Create Groups)",
			input: roundtypes.ImportApplyScoresInput{
				GuildID:  gID,
				RoundID:  rID,
				ImportID: importID,
				Scores: []roundtypes.ImportScoreData{
					{UserID: u1, Score: 45, TeamID: teamID},
				},
			},
			setupFake: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:    rID,
						Teams: []roundtypes.NormalizedTeam{{TeamID: teamID}}, // Teams mode
					}, nil
				}
				r.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{}, nil
				}
				r.RoundHasGroupsFunc = func(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) (bool, error) {
					return false, nil // Groups need creating
				}
				r.CreateRoundGroupsFunc = func(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID, participants []roundtypes.Participant) error {
					if len(participants) != 1 {
						return errors.New("expected 1 participant for group creation")
					}
					return nil
				}
				r.UpdateRoundsAndParticipantsFunc = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error {
					return nil
				}
			},
			verify: func(t *testing.T, res ApplyImportedScoresResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.IsFailure() {
					t.Fatalf("unexpected failure: %v", (*res.Failure).Error())
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := NewFakeRepo()
			if tc.setupFake != nil {
				tc.setupFake(repo)
			}

			// We don't need UserLookup for this test, so pass nil or empty
			svc := &RoundService{
				repo:    repo,
				logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
				metrics: &roundmetrics.NoOpMetrics{},
				tracer:  noop.NewTracerProvider().Tracer("test"),
			}

			res, err := svc.ApplyImportedScores(context.Background(), tc.input)
			tc.verify(t, res, err)
		})
	}
}
