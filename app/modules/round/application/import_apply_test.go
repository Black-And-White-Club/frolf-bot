package roundservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"
	"testing"

	importermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/importer"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestRoundService_ApplyImportedScores_HoleScores(t *testing.T) {
	ptrScore := func(i int) *sharedtypes.Score {
		s := sharedtypes.Score(i)
		return &s
	}

	gID := sharedtypes.GuildID("guild-hole")
	rID := sharedtypes.RoundID(uuid.New())
	importID := "import-holes"
	u1 := sharedtypes.DiscordID("user-1")
	u2 := sharedtypes.DiscordID("user-2")

	// 9-hole scorecard par values and a player's hole scores
	parScores9 := []int{3, 4, 3, 3, 4, 5, 3, 4, 3}
	holeScores9 := []int{3, 3, 4, 3, 5, 5, 2, 4, 4}

	// Only 3 holes provided (partial upload)
	partialHoles := []int{3, 4, 3}

	type testCase struct {
		name      string
		input     roundtypes.ImportApplyScoresInput
		setupFake func(*FakeRepo)
		verify    func(*testing.T, *FakeRepo, ApplyImportedScoresResult, error)
	}

	tests := []testCase{
		{
			name: "Singles - HoleScores set on updated participant",
			input: roundtypes.ImportApplyScoresInput{
				GuildID:  gID,
				RoundID:  rID,
				ImportID: importID,
				Scores: []roundtypes.ImportScoreData{
					{UserID: u1, Score: 33, HoleScores: holeScores9},
				},
			},
			setupFake: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: rID, EventMessageID: "msg-1"}, nil
				}
				r.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{
						{UserID: u1, Score: ptrScore(40), Response: roundtypes.ResponseAccept},
					}, nil
				}
				r.UpdateRoundsAndParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error {
					parts := updates[0].Participants
					if len(parts[0].HoleScores) != 9 {
						return errors.New("expected 9 hole scores on updated participant")
					}
					if parts[0].HoleScores[1] != holeScores9[1] {
						return errors.New("hole score value mismatch on updated participant")
					}
					return nil
				}
			},
			verify: func(t *testing.T, repo *FakeRepo, res ApplyImportedScoresResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.IsFailure() {
					t.Fatalf("unexpected failure: %v", (*res.Failure).Error())
				}
				parts := (*res.Success).Participants
				if len(parts[0].HoleScores) != 9 {
					t.Errorf("expected 9 hole scores on result participant, got %d", len(parts[0].HoleScores))
				}
			},
		},
		{
			name: "Singles - HoleScores set on newly added participant",
			input: roundtypes.ImportApplyScoresInput{
				GuildID:  gID,
				RoundID:  rID,
				ImportID: importID,
				Scores: []roundtypes.ImportScoreData{
					{UserID: u2, Score: 33, HoleScores: holeScores9},
				},
			},
			setupFake: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: rID, EventMessageID: "msg-1"}, nil
				}
				r.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{}, nil // No existing participants
				}
				r.UpdateRoundsAndParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error {
					parts := updates[0].Participants
					if len(parts[0].HoleScores) != 9 {
						return errors.New("expected 9 hole scores on new participant")
					}
					return nil
				}
			},
			verify: func(t *testing.T, repo *FakeRepo, res ApplyImportedScoresResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.IsFailure() {
					t.Fatalf("unexpected failure: %v", (*res.Failure).Error())
				}
				parts := (*res.Success).Participants
				if len(parts) != 1 {
					t.Fatalf("expected 1 participant, got %d", len(parts))
				}
				if len(parts[0].HoleScores) != 9 {
					t.Errorf("expected 9 hole scores on new participant result, got %d", len(parts[0].HoleScores))
				}
			},
		},
		{
			name: "Singles - ParScores trigger UpdateRound with correct values",
			input: roundtypes.ImportApplyScoresInput{
				GuildID:   gID,
				RoundID:   rID,
				ImportID:  importID,
				ParScores: parScores9,
				Scores:    []roundtypes.ImportScoreData{{UserID: u1, Score: 33}},
			},
			setupFake: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: rID, EventMessageID: "msg-1"}, nil
				}
				r.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{}, nil
				}
				r.UpdateRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID, round *roundtypes.Round) (*roundtypes.Round, error) {
					if len(round.ParScores) != 9 {
						return nil, errors.New("expected 9 par scores in UpdateRound call")
					}
					for i, v := range parScores9 {
						if round.ParScores[i] != v {
							return nil, errors.New("par score value mismatch in UpdateRound")
						}
					}
					return &roundtypes.Round{}, nil
				}
			},
			verify: func(t *testing.T, repo *FakeRepo, res ApplyImportedScoresResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.IsFailure() {
					t.Fatalf("UpdateRound validation failed: %v", (*res.Failure).Error())
				}
				if !slices.Contains(repo.Trace(), "UpdateRound") {
					t.Errorf("expected UpdateRound called when ParScores non-empty, trace=%v", repo.Trace())
				}
			},
		},
		{
			name: "Singles - No UpdateRound called when ParScores is empty",
			input: roundtypes.ImportApplyScoresInput{
				GuildID:  gID,
				RoundID:  rID,
				ImportID: importID,
				Scores:   []roundtypes.ImportScoreData{{UserID: u1, Score: 33}},
				// ParScores intentionally absent
			},
			setupFake: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: rID}, nil
				}
				r.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{}, nil
				}
			},
			verify: func(t *testing.T, repo *FakeRepo, res ApplyImportedScoresResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.IsFailure() {
					t.Fatalf("unexpected failure: %v", (*res.Failure).Error())
				}
				if slices.Contains(repo.Trace(), "UpdateRound") {
					t.Errorf("UpdateRound should not be called when ParScores is empty, trace=%v", repo.Trace())
				}
			},
		},
		{
			// UDisc may export a subset of holes if the scorecard was cut short or
			// a hole was skipped. We store whatever was provided and render "-" for gaps.
			name: "Singles - Partial HoleScores (3 of 9 holes) stored as-is",
			input: roundtypes.ImportApplyScoresInput{
				GuildID:  gID,
				RoundID:  rID,
				ImportID: importID,
				Scores: []roundtypes.ImportScoreData{
					{UserID: u1, Score: 33, HoleScores: partialHoles},
				},
			},
			setupFake: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: rID}, nil
				}
				r.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{}, nil
				}
				r.UpdateRoundsAndParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error {
					parts := updates[0].Participants
					if len(parts[0].HoleScores) != 3 {
						return errors.New("expected exactly 3 hole scores, not padded or truncated")
					}
					return nil
				}
			},
			verify: func(t *testing.T, repo *FakeRepo, res ApplyImportedScoresResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.IsFailure() {
					t.Fatalf("unexpected failure: %v", (*res.Failure).Error())
				}
				if len((*res.Success).Participants[0].HoleScores) != 3 {
					t.Errorf("expected 3 partial hole scores preserved, got %d", len((*res.Success).Participants[0].HoleScores))
				}
			},
		},
		{
			name: "Singles Overwrite - HoleScores preserved",
			input: roundtypes.ImportApplyScoresInput{
				GuildID:                 gID,
				RoundID:                 rID,
				ImportID:                importID,
				OverwriteExistingScores: true,
				Scores: []roundtypes.ImportScoreData{
					{UserID: u1, Score: 33, HoleScores: holeScores9},
					{UserID: u2, Score: 36, HoleScores: partialHoles},
				},
			},
			setupFake: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: rID, EventMessageID: "msg-1"}, nil
				}
				r.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{}, nil
				}
				r.UpdateRoundsAndParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error {
					parts := updates[0].Participants
					if len(parts[0].HoleScores) != 9 {
						return errors.New("expected 9 hole scores for u1 in overwrite")
					}
					if len(parts[1].HoleScores) != 3 {
						return errors.New("expected 3 partial hole scores for u2 in overwrite")
					}
					return nil
				}
			},
			verify: func(t *testing.T, repo *FakeRepo, res ApplyImportedScoresResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.IsFailure() {
					t.Fatalf("unexpected failure: %v", (*res.Failure).Error())
				}
				parts := (*res.Success).Participants
				if len(parts) != 2 {
					t.Fatalf("expected 2 participants, got %d", len(parts))
				}
				if len(parts[0].HoleScores) != 9 {
					t.Errorf("u1: expected 9 hole scores, got %d", len(parts[0].HoleScores))
				}
				if len(parts[1].HoleScores) != 3 {
					t.Errorf("u2: expected 3 partial hole scores, got %d", len(parts[1].HoleScores))
				}
			},
		},
		{
			name: "Teams - HoleScores set on updated participant",
			input: roundtypes.ImportApplyScoresInput{
				GuildID:  gID,
				RoundID:  rID,
				ImportID: importID,
				Scores: []roundtypes.ImportScoreData{
					{UserID: u1, Score: 33, HoleScores: holeScores9, TeamID: uuid.New()},
				},
			},
			setupFake: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: rID, EventMessageID: "msg-1", Teams: []roundtypes.NormalizedTeam{{}}}, nil
				}
				r.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{
						{UserID: u1, Score: ptrScore(40), Response: roundtypes.ResponseAccept},
					}, nil
				}
				r.RoundHasGroupsFunc = func(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) (bool, error) {
					return true, nil
				}
				r.UpdateRoundsAndParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error {
					parts := updates[0].Participants
					if len(parts[0].HoleScores) != 9 {
						return errors.New("expected 9 hole scores on updated team participant")
					}
					if parts[0].HoleScores[1] != holeScores9[1] {
						return errors.New("hole score value mismatch on updated team participant")
					}
					return nil
				}
			},
			verify: func(t *testing.T, repo *FakeRepo, res ApplyImportedScoresResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.IsFailure() {
					t.Fatalf("unexpected failure: %v", (*res.Failure).Error())
				}
				parts := (*res.Success).Participants
				if len(parts[0].HoleScores) != 9 {
					t.Errorf("expected 9 hole scores on team result participant, got %d", len(parts[0].HoleScores))
				}
			},
		},
		{
			name: "Teams - HoleScores set on newly added participant",
			input: roundtypes.ImportApplyScoresInput{
				GuildID:  gID,
				RoundID:  rID,
				ImportID: importID,
				Scores: []roundtypes.ImportScoreData{
					{UserID: u2, Score: 33, HoleScores: holeScores9, TeamID: uuid.New()},
				},
			},
			setupFake: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: rID, EventMessageID: "msg-1", Teams: []roundtypes.NormalizedTeam{{}}}, nil
				}
				r.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{}, nil
				}
				r.RoundHasGroupsFunc = func(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) (bool, error) {
					return true, nil
				}
				r.UpdateRoundsAndParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error {
					parts := updates[0].Participants
					if len(parts[0].HoleScores) != 9 {
						return errors.New("expected 9 hole scores on new team participant")
					}
					return nil
				}
			},
			verify: func(t *testing.T, repo *FakeRepo, res ApplyImportedScoresResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.IsFailure() {
					t.Fatalf("unexpected failure: %v", (*res.Failure).Error())
				}
				parts := (*res.Success).Participants
				if len(parts) != 1 {
					t.Fatalf("expected 1 participant, got %d", len(parts))
				}
				if len(parts[0].HoleScores) != 9 {
					t.Errorf("expected 9 hole scores on new team participant result, got %d", len(parts[0].HoleScores))
				}
			},
		},
		{
			name: "Teams - Guest participant gets HoleScores",
			input: roundtypes.ImportApplyScoresInput{
				GuildID:  gID,
				RoundID:  rID,
				ImportID: importID,
				Scores: []roundtypes.ImportScoreData{
					{UserID: "", RawName: "Guest McGuest", Score: 38, HoleScores: holeScores9, TeamID: uuid.New()},
				},
			},
			setupFake: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: rID, Teams: []roundtypes.NormalizedTeam{{}}}, nil
				}
				r.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{}, nil
				}
				r.RoundHasGroupsFunc = func(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) (bool, error) {
					return true, nil
				}
				r.UpdateRoundsAndParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error {
					parts := updates[0].Participants
					if parts[0].UserID != "" {
						return errors.New("expected guest (empty UserID)")
					}
					if len(parts[0].HoleScores) != 9 {
						return errors.New("expected 9 hole scores on guest team participant")
					}
					return nil
				}
			},
			verify: func(t *testing.T, repo *FakeRepo, res ApplyImportedScoresResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.IsFailure() {
					t.Fatalf("unexpected failure: %v", (*res.Failure).Error())
				}
				if len((*res.Success).Participants[0].HoleScores) != 9 {
					t.Errorf("expected 9 hole scores on guest team result participant")
				}
			},
		},
		{
			name: "Teams - ParScores saved to round",
			input: roundtypes.ImportApplyScoresInput{
				GuildID:   gID,
				RoundID:   rID,
				ImportID:  importID,
				ParScores: parScores9,
				Scores: []roundtypes.ImportScoreData{
					{UserID: u1, Score: 33, TeamID: uuid.New()},
				},
			},
			setupFake: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: rID, EventMessageID: "msg-1", Teams: []roundtypes.NormalizedTeam{{}}}, nil
				}
				r.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{}, nil
				}
				r.RoundHasGroupsFunc = func(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) (bool, error) {
					return true, nil
				}
				r.UpdateRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID, round *roundtypes.Round) (*roundtypes.Round, error) {
					if len(round.ParScores) != 9 {
						return nil, errors.New("expected 9 par scores in UpdateRound call")
					}
					for i, v := range parScores9 {
						if round.ParScores[i] != v {
							return nil, errors.New("par score value mismatch in UpdateRound")
						}
					}
					return &roundtypes.Round{}, nil
				}
			},
			verify: func(t *testing.T, repo *FakeRepo, res ApplyImportedScoresResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.IsFailure() {
					t.Fatalf("UpdateRound validation failed: %v", (*res.Failure).Error())
				}
				if !slices.Contains(repo.Trace(), "UpdateRound") {
					t.Errorf("expected UpdateRound called when ParScores non-empty, trace=%v", repo.Trace())
				}
			},
		},
		{
			// Guest players enabled: guests should also carry hole scores through.
			name: "Singles - Guest participant gets HoleScores",
			input: roundtypes.ImportApplyScoresInput{
				GuildID:           gID,
				RoundID:           rID,
				ImportID:          importID,
				AllowGuestPlayers: true,
				Scores: []roundtypes.ImportScoreData{
					{UserID: "", RawName: "Guest McGuest", Score: 38, HoleScores: holeScores9},
				},
			},
			setupFake: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: rID}, nil
				}
				r.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{}, nil
				}
				r.UpdateRoundsAndParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error {
					parts := updates[0].Participants
					if parts[0].UserID != "" {
						return errors.New("expected guest (empty UserID)")
					}
					if len(parts[0].HoleScores) != 9 {
						return errors.New("expected 9 hole scores on guest participant")
					}
					return nil
				}
			},
			verify: func(t *testing.T, repo *FakeRepo, res ApplyImportedScoresResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.IsFailure() {
					t.Fatalf("unexpected failure: %v", (*res.Failure).Error())
				}
				if len((*res.Success).Participants[0].HoleScores) != 9 {
					t.Errorf("expected 9 hole scores on guest result participant")
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

			svc := &RoundService{
				repo:            repo,
				logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
				metrics:         &roundmetrics.NoOpMetrics{},
				importerMetrics: importermetrics.NewNoOpMetrics(),
				tracer:          noop.NewTracerProvider().Tracer("test"),
			}

			res, err := svc.ApplyImportedScores(context.Background(), tc.input)
			tc.verify(t, repo, res, err)
		})
	}
}

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
	tag1 := sharedtypes.TagNumber(17)

	type testCase struct {
		name      string
		input     roundtypes.ImportApplyScoresInput
		setupFake func(*FakeRepo)
		verify    func(*testing.T, *FakeRepo, ApplyImportedScoresResult, error)
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
			verify: func(t *testing.T, repo *FakeRepo, res ApplyImportedScoresResult, err error) {
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
				if !slices.Contains(repo.Trace(), "UpdateImportStatus") {
					t.Fatalf("expected UpdateImportStatus to be called, trace=%v", repo.Trace())
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
			verify: func(t *testing.T, repo *FakeRepo, res ApplyImportedScoresResult, err error) {
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
				if slices.Contains(repo.Trace(), "UpdateImportStatus") {
					t.Fatalf("did not expect UpdateImportStatus on failure, trace=%v", repo.Trace())
				}
			},
		},
		{
			name: "Singles - Success (Overwrite with guests)",
			input: roundtypes.ImportApplyScoresInput{
				GuildID:                 gID,
				RoundID:                 rID,
				ImportID:                importID,
				AllowGuestPlayers:       true,
				OverwriteExistingScores: true,
				Scores: []roundtypes.ImportScoreData{
					{UserID: u1, Score: 49, RawName: "User 1"},
					{UserID: "", Score: 53, RawName: "Guest Player"},
				},
			},
			setupFake: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: rID, EventMessageID: "msg-123"}, nil
				}
				r.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{
						{UserID: u1, Score: ptrScore(60), Response: roundtypes.ResponseAccept, TagNumber: &tag1},
						{UserID: u2, Score: nil, Response: roundtypes.ResponseAccept},
					}, nil
				}
				r.UpdateRoundsAndParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error {
					if len(updates) != 1 {
						return errors.New("expected 1 update")
					}
					parts := updates[0].Participants
					if len(parts) != 2 {
						return errors.New("expected overwrite to keep exactly 2 participants")
					}
					if parts[0].UserID != u1 {
						return errors.New("expected first participant to be matched user")
					}
					if parts[0].TagNumber == nil || *parts[0].TagNumber != tag1 {
						return errors.New("expected matched user tag to be preserved")
					}
					if parts[1].UserID != "" || parts[1].RawName != "Guest Player" {
						return errors.New("expected second participant to be guest row")
					}
					return nil
				}
			},
			verify: func(t *testing.T, repo *FakeRepo, res ApplyImportedScoresResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.IsFailure() {
					t.Fatalf("unexpected failure: %v", (*res.Failure).Error())
				}
				if len((*res.Success).Participants) != 2 {
					t.Fatalf("expected 2 participants after overwrite, got %d", len((*res.Success).Participants))
				}
				if !slices.Contains(repo.Trace(), "UpdateImportStatus") {
					t.Fatalf("expected UpdateImportStatus to be called, trace=%v", repo.Trace())
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
			verify: func(t *testing.T, repo *FakeRepo, res ApplyImportedScoresResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.IsFailure() {
					t.Fatalf("unexpected failure: %v", (*res.Failure).Error())
				}
				if !slices.Contains(repo.Trace(), "UpdateImportStatus") {
					t.Fatalf("expected UpdateImportStatus to be called, trace=%v", repo.Trace())
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
				repo:            repo,
				logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
				metrics:         &roundmetrics.NoOpMetrics{},
				importerMetrics: importermetrics.NewNoOpMetrics(),
				tracer:          noop.NewTracerProvider().Tracer("test"),
			}

			res, err := svc.ApplyImportedScores(context.Background(), tc.input)
			tc.verify(t, repo, res, err)
		})
	}
}
