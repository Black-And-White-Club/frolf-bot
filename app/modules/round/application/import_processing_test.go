package roundservice

import (
	"context"
	"errors"
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
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
				repo:       repo,
				userLookup: lookup,
				logger:     logger,
				tracer:     tracer,
				metrics:    metrics,
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
