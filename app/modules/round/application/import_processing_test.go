package roundservice

import (
	"context"
	"errors"
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	importermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/importer"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestRoundService_ImportProcessing(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-123")
	roundID := sharedtypes.RoundID(uuid.New())
	importID := "import-456"

	tests := []struct {
		name          string
		isNormalize   bool
		mode          sharedtypes.RoundMode
		inputData     *roundtypes.ParsedScorecard
		setupFakes    func(r *FakeRepo, l *FakeUserLookup)
		allowGuests   bool
		expectSuccess bool
		expectedCode  string
		check         func(t *testing.T, res any)
	}{
		// --- NormalizeParsedScorecard Tests ---
		{
			name:        "Normalize: Singles Success",
			isNormalize: true,
			mode:        sharedtypes.RoundModeSingles,
			inputData: &roundtypes.ParsedScorecard{
				Mode:         sharedtypes.RoundModeSingles,
				PlayerScores: []roundtypes.PlayerScoreRow{{PlayerName: "Alice", Total: 54}},
			},
			expectSuccess: true,
		},
		{
			name:        "Normalize: Doubles Success (Team Mapping)",
			isNormalize: true,
			mode:        sharedtypes.RoundModeDoubles,
			inputData: &roundtypes.ParsedScorecard{
				Mode: sharedtypes.RoundModeDoubles,
				PlayerScores: []roundtypes.PlayerScoreRow{
					{TeamNames: []string{"Alice", "Bob"}, Total: 48},
				},
			},
			expectSuccess: true,
			check: func(t *testing.T, res any) {
				payload := res.(*roundtypes.NormalizedScorecard)
				if len(payload.Teams) != 1 || len(payload.Teams[0].Members) != 2 {
					t.Errorf("failed to map doubles team members correctly")
				}
			},
		},
		{
			name:          "Normalize: Error Nil Data",
			isNormalize:   true,
			inputData:     nil,
			expectSuccess: false,
		},

		// --- IngestNormalizedScorecard Tests ---
		{
			name: "Ingest: Singles Partial Match",
			mode: sharedtypes.RoundModeSingles,
			setupFakes: func(r *FakeRepo, l *FakeUserLookup) {
				l.FindByDisplayFn = func(name string) sharedtypes.DiscordID {
					if name == "alice" {
						return "alice-id"
					}
					return ""
				}
			},
			expectSuccess: true,
			check: func(t *testing.T, res any) {
				p := res.(*roundtypes.IngestScorecardResult)
				if p.MatchedPlayers != 1 || p.UnmatchedPlayers != 1 {
					t.Errorf("expected 1 match/1 skip, got %d/%d", p.MatchedPlayers, p.UnmatchedPlayers)
				}
			},
		},
		{
			name: "Ingest: resolveUserID Fallback Logic",
			mode: sharedtypes.RoundModeSingles,
			setupFakes: func(r *FakeRepo, l *FakeUserLookup) {
				l.FindByUsernameFn = func(name string) sharedtypes.DiscordID {
					return "" // Fail all username lookups
				}
				l.FindByDisplayFn = func(name string) sharedtypes.DiscordID {
					if name == "alice" {
						return "alice-discord-id" // Succeed on display name fallback
					}
					return ""
				}
			},
			expectSuccess: true,
			check: func(t *testing.T, res any) {
				p := res.(*roundtypes.IngestScorecardResult)
				if p.MatchedPlayers != 1 {
					t.Errorf("expected 1 match via fallback, got %d", p.MatchedPlayers)
				}
			},
		},
		{
			name: "Ingest: Doubles Create Groups (First Ingest)",
			mode: sharedtypes.RoundModeDoubles,
			setupFakes: func(r *FakeRepo, l *FakeUserLookup) {
				l.FindByDisplayFn = func(name string) sharedtypes.DiscordID { return "user-id" }
				r.RoundHasGroupsFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.RoundID) (bool, error) { return false, nil }
				r.CreateRoundGroupsFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.RoundID, p []roundtypes.Participant) error {
					return nil
				}
			},
			expectSuccess: true,
		},
		{
			name: "Ingest: Doubles Skip Group Creation (Already Exists)",
			mode: sharedtypes.RoundModeDoubles,
			setupFakes: func(r *FakeRepo, l *FakeUserLookup) {
				l.FindByDisplayFn = func(name string) sharedtypes.DiscordID { return "user-id" }
				r.RoundHasGroupsFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.RoundID) (bool, error) { return true, nil }
				r.CreateRoundGroupsFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.RoundID, p []roundtypes.Participant) error {
					t.Error("CreateRoundGroups should NOT be called when groups already exist")
					return nil
				}
			},
			expectSuccess: true,
		},
		{
			name: "Ingest: DB Error Outcome",
			mode: sharedtypes.RoundModeDoubles,
			setupFakes: func(r *FakeRepo, l *FakeUserLookup) {
				l.FindByUsernameFn = func(name string) sharedtypes.DiscordID { return "user-id" }
				r.RoundHasGroupsFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.RoundID) (bool, error) {
					return false, errors.New("db connection lost")
				}
			},
			expectSuccess: false,
		},
		{
			name: "Ingest: Failure - No Matches Found",
			mode: sharedtypes.RoundModeSingles,
			setupFakes: func(r *FakeRepo, l *FakeUserLookup) {
				l.FindByDisplayFn = func(name string) sharedtypes.DiscordID { return "" }
			},
			expectSuccess: false,
		},
		{
			name: "Ingest: Singles Guest Fallback",
			mode: sharedtypes.RoundModeSingles,
			setupFakes: func(r *FakeRepo, l *FakeUserLookup) {
				l.FindByDisplayFn = func(name string) sharedtypes.DiscordID { return "" }
			},
			allowGuests:   true,
			expectSuccess: true,
			check: func(t *testing.T, res any) {
				p := res.(*roundtypes.IngestScorecardResult)
				if p.MatchedPlayers != 0 {
					t.Errorf("expected 0 matches, got %d", p.MatchedPlayers)
				}
				if p.UnmatchedPlayers != 2 {
					t.Errorf("expected 2 unmatched, got %d", p.UnmatchedPlayers)
				}
				if len(p.Scores) != 2 {
					t.Fatalf("expected 2 score entries, got %d", len(p.Scores))
				}
				if p.Scores[0].UserID != "" || p.Scores[0].RawName == "" {
					t.Errorf("expected guest score row with raw name, got %+v", p.Scores[0])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewFakeRepo()
			lookup := &FakeUserLookup{}
			logger := loggerfrolfbot.NoOpLogger
			tracerProvider := noop.NewTracerProvider()
			tracer := tracerProvider.Tracer("test")
			metrics := &roundmetrics.NoOpMetrics{}
			if tt.setupFakes != nil {
				tt.setupFakes(repo, lookup)
			}

			s := &RoundService{
				repo:            repo,
				userLookup:      lookup,
				logger:          logger,
				tracer:          tracer,
				metrics:         metrics,
				importerMetrics: importermetrics.NewNoOpMetrics(),
			}

			if tt.isNormalize {
				meta := roundtypes.Metadata{GuildID: guildID, RoundID: roundID, ImportID: importID}
				res, _ := s.NormalizeParsedScorecard(ctx, tt.inputData, meta)
				if tt.expectSuccess {
					if res.Success == nil {
						t.Errorf("expected success but got failure: %+v", res.Failure)
						return
					}
					if tt.check != nil {
						tt.check(t, *res.Success)
					}
				} else {
					if res.Failure == nil {
						t.Error("expected failure but got success")
					}
				}
			} else {
				payload := roundtypes.ImportIngestScorecardInput{
					GuildID: guildID, RoundID: roundID, ImportID: importID,
					NormalizedData: roundtypes.NormalizedScorecard{
						Mode: tt.mode,
					},
					AllowGuestPlayers: tt.allowGuests,
				}
				if tt.mode == sharedtypes.RoundModeSingles {
					payload.NormalizedData.Players = []roundtypes.NormalizedPlayer{
						{DisplayName: "Alice", Total: 54}, {DisplayName: "Bob", Total: 60},
					}
				} else {
					payload.NormalizedData.Teams = []roundtypes.NormalizedTeam{
						{Members: []roundtypes.TeamMember{{RawName: "Alice"}}, Total: 48},
					}
				}

				res, _ := s.IngestNormalizedScorecard(ctx, payload)
				if tt.expectSuccess {
					if res.Success == nil {
						t.Errorf("expected success but got failure: %+v", res.Failure)
						return
					}
					if tt.check != nil {
						tt.check(t, *res.Success)
					}
				} else {
					if res.Failure == nil {
						t.Error("expected failure but got success")
					}
				}
			}
		})
	}
}

func TestRoundService_IngestNormalizedScorecard_UsesBatchLookup(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-123")
	roundID := sharedtypes.RoundID(uuid.New())

	lookup := &batchLookupStub{
		resolved: map[string]sharedtypes.DiscordID{
			"alice": "alice-id",
		},
	}

	service := &RoundService{
		repo:            NewFakeRepo(),
		userLookup:      lookup,
		logger:          loggerfrolfbot.NoOpLogger,
		tracer:          noop.NewTracerProvider().Tracer("test"),
		metrics:         &roundmetrics.NoOpMetrics{},
		importerMetrics: importermetrics.NewNoOpMetrics(),
	}

	req := roundtypes.ImportIngestScorecardInput{
		GuildID:  guildID,
		RoundID:  roundID,
		ImportID: "import-123",
		NormalizedData: roundtypes.NormalizedScorecard{
			Mode: sharedtypes.RoundModeSingles,
			Players: []roundtypes.NormalizedPlayer{
				{DisplayName: "Alice", Total: 54},
				{DisplayName: "Bob", Total: 60},
			},
		},
	}

	res, err := service.IngestNormalizedScorecard(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Failure != nil {
		t.Fatalf("unexpected failure: %v", *res.Failure)
	}
	if lookup.batchCalls != 1 {
		t.Fatalf("expected one batch lookup call, got %d", lookup.batchCalls)
	}
	if lookup.singleLookupCalls != 0 {
		t.Fatalf("expected no single-name lookup calls, got %d", lookup.singleLookupCalls)
	}
}

type batchLookupStub struct {
	resolved          map[string]sharedtypes.DiscordID
	batchCalls        int
	singleLookupCalls int
}

func (b *batchLookupStub) ResolveByNormalizedNames(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, normalizedNames []string) (map[string]sharedtypes.DiscordID, error) {
	b.batchCalls++
	out := make(map[string]sharedtypes.DiscordID, len(normalizedNames))
	for _, name := range normalizedNames {
		out[name] = b.resolved[name]
	}
	return out, nil
}

func (b *batchLookupStub) FindByNormalizedUDiscUsername(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, normalizedUsername string) (*UserIdentity, error) {
	b.singleLookupCalls++
	return nil, nil
}

func (b *batchLookupStub) FindGlobalByNormalizedUDiscUsername(ctx context.Context, db bun.IDB, normalizedUsername string) (*UserIdentity, error) {
	b.singleLookupCalls++
	return nil, nil
}

func (b *batchLookupStub) FindByNormalizedUDiscDisplayName(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, normalizedDisplayName string) (*UserIdentity, error) {
	b.singleLookupCalls++
	return nil, nil
}

func (b *batchLookupStub) FindByPartialUDiscName(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, partialName string) ([]*UserIdentity, error) {
	return nil, nil
}

func TestRoundService_HoleScorePassthrough(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-holes")
	roundID := sharedtypes.RoundID(uuid.New())
	importID := "import-holes"

	holes9 := []int{3, 3, 4, 3, 5, 5, 2, 4, 4}
	par9 := []int{3, 4, 3, 3, 4, 5, 3, 4, 3}
	partial := []int{3, 4, 3}

	t.Run("Normalize/Singles - HoleScores cloned into NormalizedPlayer", func(t *testing.T) {
		svc := &RoundService{
			logger:          loggerfrolfbot.NoOpLogger,
			tracer:          noop.NewTracerProvider().Tracer("test"),
			metrics:         &roundmetrics.NoOpMetrics{},
			importerMetrics: importermetrics.NewNoOpMetrics(),
		}
		meta := roundtypes.Metadata{GuildID: guildID, RoundID: roundID, ImportID: importID}
		data := &roundtypes.ParsedScorecard{
			Mode: sharedtypes.RoundModeSingles,
			PlayerScores: []roundtypes.PlayerScoreRow{
				{PlayerName: "Alice", Total: 33, HoleScores: holes9},
			},
		}

		res, _ := svc.NormalizeParsedScorecard(ctx, data, meta)
		if res.Success == nil {
			t.Fatalf("expected success, got failure: %v", res.Failure)
		}
		players := (*res.Success).Players
		if len(players) != 1 {
			t.Fatalf("expected 1 player, got %d", len(players))
		}
		if len(players[0].HoleScores) != 9 {
			t.Errorf("expected 9 hole scores, got %d", len(players[0].HoleScores))
		}
		for i, v := range holes9 {
			if players[0].HoleScores[i] != v {
				t.Errorf("hole %d: want %d, got %d", i+1, v, players[0].HoleScores[i])
			}
		}
	})

	t.Run("Normalize/Singles - Partial HoleScores cloned as-is", func(t *testing.T) {
		svc := &RoundService{
			logger:          loggerfrolfbot.NoOpLogger,
			tracer:          noop.NewTracerProvider().Tracer("test"),
			metrics:         &roundmetrics.NoOpMetrics{},
			importerMetrics: importermetrics.NewNoOpMetrics(),
		}
		meta := roundtypes.Metadata{GuildID: guildID, RoundID: roundID, ImportID: importID}
		data := &roundtypes.ParsedScorecard{
			Mode: sharedtypes.RoundModeSingles,
			PlayerScores: []roundtypes.PlayerScoreRow{
				{PlayerName: "Alice", Total: 10, HoleScores: partial},
			},
		}

		res, _ := svc.NormalizeParsedScorecard(ctx, data, meta)
		if res.Success == nil {
			t.Fatalf("expected success")
		}
		if len((*res.Success).Players[0].HoleScores) != 3 {
			t.Errorf("expected 3 partial hole scores, got %d", len((*res.Success).Players[0].HoleScores))
		}
	})

	t.Run("Normalize/Doubles - HoleScores cloned into NormalizedTeam", func(t *testing.T) {
		svc := &RoundService{
			logger:          loggerfrolfbot.NoOpLogger,
			tracer:          noop.NewTracerProvider().Tracer("test"),
			metrics:         &roundmetrics.NoOpMetrics{},
			importerMetrics: importermetrics.NewNoOpMetrics(),
		}
		meta := roundtypes.Metadata{GuildID: guildID, RoundID: roundID, ImportID: importID}
		data := &roundtypes.ParsedScorecard{
			Mode: sharedtypes.RoundModeDoubles,
			PlayerScores: []roundtypes.PlayerScoreRow{
				{TeamNames: []string{"Alice", "Bob"}, Total: 33, HoleScores: holes9},
			},
		}

		res, _ := svc.NormalizeParsedScorecard(ctx, data, meta)
		if res.Success == nil {
			t.Fatalf("expected success")
		}
		teams := (*res.Success).Teams
		if len(teams) != 1 {
			t.Fatalf("expected 1 team")
		}
		if len(teams[0].HoleScores) != 9 {
			t.Errorf("expected 9 team hole scores, got %d", len(teams[0].HoleScores))
		}
	})

	t.Run("Normalize - ParScores cloned into NormalizedScorecard", func(t *testing.T) {
		svc := &RoundService{
			logger:          loggerfrolfbot.NoOpLogger,
			tracer:          noop.NewTracerProvider().Tracer("test"),
			metrics:         &roundmetrics.NoOpMetrics{},
			importerMetrics: importermetrics.NewNoOpMetrics(),
		}
		meta := roundtypes.Metadata{GuildID: guildID, RoundID: roundID, ImportID: importID}
		data := &roundtypes.ParsedScorecard{
			Mode:         sharedtypes.RoundModeSingles,
			ParScores:    par9,
			PlayerScores: []roundtypes.PlayerScoreRow{{PlayerName: "Alice", Total: 33}},
		}

		res, _ := svc.NormalizeParsedScorecard(ctx, data, meta)
		if res.Success == nil {
			t.Fatalf("expected success")
		}
		if len((*res.Success).ParScores) != 9 {
			t.Errorf("expected 9 par scores, got %d", len((*res.Success).ParScores))
		}
		for i, v := range par9 {
			if (*res.Success).ParScores[i] != v {
				t.Errorf("par hole %d: want %d, got %d", i+1, v, (*res.Success).ParScores[i])
			}
		}
	})

	t.Run("Ingest/Singles - HoleScores passed to ScoreInfo for matched player", func(t *testing.T) {
		lookup := &batchLookupStub{
			resolved: map[string]sharedtypes.DiscordID{"alice": "alice-id"},
		}
		svc := &RoundService{
			repo:            NewFakeRepo(),
			userLookup:      lookup,
			logger:          loggerfrolfbot.NoOpLogger,
			tracer:          noop.NewTracerProvider().Tracer("test"),
			metrics:         &roundmetrics.NoOpMetrics{},
			importerMetrics: importermetrics.NewNoOpMetrics(),
		}
		req := roundtypes.ImportIngestScorecardInput{
			GuildID:  guildID,
			RoundID:  roundID,
			ImportID: importID,
			NormalizedData: roundtypes.NormalizedScorecard{
				Mode: sharedtypes.RoundModeSingles,
				Players: []roundtypes.NormalizedPlayer{
					{DisplayName: "Alice", Total: 33, HoleScores: holes9},
				},
			},
		}

		res, err := svc.IngestNormalizedScorecard(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Success == nil {
			t.Fatalf("expected success, got failure")
		}
		scores := (*res.Success).Scores
		if len(scores) != 1 {
			t.Fatalf("expected 1 score, got %d", len(scores))
		}
		if len(scores[0].HoleScores) != 9 {
			t.Errorf("expected 9 hole scores in ScoreInfo, got %d", len(scores[0].HoleScores))
		}
	})

	t.Run("Ingest/Singles - Partial HoleScores pass through unchanged", func(t *testing.T) {
		lookup := &batchLookupStub{
			resolved: map[string]sharedtypes.DiscordID{"alice": "alice-id"},
		}
		svc := &RoundService{
			repo:            NewFakeRepo(),
			userLookup:      lookup,
			logger:          loggerfrolfbot.NoOpLogger,
			tracer:          noop.NewTracerProvider().Tracer("test"),
			metrics:         &roundmetrics.NoOpMetrics{},
			importerMetrics: importermetrics.NewNoOpMetrics(),
		}
		req := roundtypes.ImportIngestScorecardInput{
			GuildID:  guildID,
			RoundID:  roundID,
			ImportID: importID,
			NormalizedData: roundtypes.NormalizedScorecard{
				Mode: sharedtypes.RoundModeSingles,
				Players: []roundtypes.NormalizedPlayer{
					{DisplayName: "Alice", Total: 10, HoleScores: partial},
				},
			},
		}

		res, _ := svc.IngestNormalizedScorecard(ctx, req)
		if res.Success == nil {
			t.Fatalf("expected success")
		}
		if len((*res.Success).Scores[0].HoleScores) != 3 {
			t.Errorf("expected 3 partial hole scores, got %d", len((*res.Success).Scores[0].HoleScores))
		}
	})

	t.Run("Ingest/Singles - Guest player gets HoleScores", func(t *testing.T) {
		lookup := &batchLookupStub{
			resolved: map[string]sharedtypes.DiscordID{}, // no matches → guest
		}
		svc := &RoundService{
			repo:            NewFakeRepo(),
			userLookup:      lookup,
			logger:          loggerfrolfbot.NoOpLogger,
			tracer:          noop.NewTracerProvider().Tracer("test"),
			metrics:         &roundmetrics.NoOpMetrics{},
			importerMetrics: importermetrics.NewNoOpMetrics(),
		}
		req := roundtypes.ImportIngestScorecardInput{
			GuildID:           guildID,
			RoundID:           roundID,
			ImportID:          importID,
			AllowGuestPlayers: true,
			NormalizedData: roundtypes.NormalizedScorecard{
				Mode: sharedtypes.RoundModeSingles,
				Players: []roundtypes.NormalizedPlayer{
					{DisplayName: "Guest McGee", Total: 38, HoleScores: holes9},
				},
			},
		}

		res, _ := svc.IngestNormalizedScorecard(ctx, req)
		if res.Success == nil {
			t.Fatalf("expected success")
		}
		scores := (*res.Success).Scores
		if len(scores) != 1 {
			t.Fatalf("expected 1 score entry")
		}
		if scores[0].UserID != "" {
			t.Errorf("expected guest (empty UserID), got %q", scores[0].UserID)
		}
		if len(scores[0].HoleScores) != 9 {
			t.Errorf("expected 9 hole scores on guest ScoreInfo, got %d", len(scores[0].HoleScores))
		}
	})

	t.Run("Ingest - ParScores returned in IngestScorecardResult", func(t *testing.T) {
		lookup := &batchLookupStub{
			resolved: map[string]sharedtypes.DiscordID{"alice": "alice-id"},
		}
		svc := &RoundService{
			repo:            NewFakeRepo(),
			userLookup:      lookup,
			logger:          loggerfrolfbot.NoOpLogger,
			tracer:          noop.NewTracerProvider().Tracer("test"),
			metrics:         &roundmetrics.NoOpMetrics{},
			importerMetrics: importermetrics.NewNoOpMetrics(),
		}
		req := roundtypes.ImportIngestScorecardInput{
			GuildID:  guildID,
			RoundID:  roundID,
			ImportID: importID,
			NormalizedData: roundtypes.NormalizedScorecard{
				Mode:      sharedtypes.RoundModeSingles,
				ParScores: par9,
				Players: []roundtypes.NormalizedPlayer{
					{DisplayName: "Alice", Total: 33},
				},
			},
		}

		res, err := svc.IngestNormalizedScorecard(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Success == nil {
			t.Fatalf("expected success")
		}
		result := *res.Success
		if len(result.ParScores) != 9 {
			t.Errorf("expected 9 par scores in result, got %d", len(result.ParScores))
		}
		for i, v := range par9 {
			if result.ParScores[i] != v {
				t.Errorf("par hole %d: want %d, got %d", i+1, v, result.ParScores[i])
			}
		}
	})
}
