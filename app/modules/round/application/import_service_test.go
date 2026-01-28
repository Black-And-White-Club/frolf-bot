package roundservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/parsers"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

// =============================================================================
// TEST: CreateImportJob
// =============================================================================

func TestRoundService_CreateImportJob(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-1")
	roundID := sharedtypes.RoundID(uuid.New())

	tests := []struct {
		name       string
		setupRepo  func(f *FakeRepo)
		payload    *roundtypes.ImportCreateJobInput
		assertFunc func(t *testing.T, res results.OperationResult[roundtypes.CreateImportJobResult, error], repo *FakeRepo)
	}{
		{
			name: "Success - CSV Upload",
			payload: &roundtypes.ImportCreateJobInput{
				GuildID: guildID, RoundID: roundID, ImportID: "import-1", FileName: "test.csv", FileData: []byte("content"),
			},
			setupRepo: func(f *FakeRepo) {
				f.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: r, GuildID: g}, nil
				}
				f.UpdateRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID, r *roundtypes.Round) (*roundtypes.Round, error) {
					if r.ImportStatus != "pending" {
						return nil, errors.New("expected pending status")
					}
					return r, nil
				}
			},
			assertFunc: func(t *testing.T, res results.OperationResult[roundtypes.CreateImportJobResult, error], repo *FakeRepo) {
				if res.Success == nil {
					t.Fatalf("expected success, got failure: %+v", res.Failure)
				}

				// Verify DB calls
				trace := repo.Trace()
				if len(trace) != 2 || trace[0] != "GetRound" || trace[1] != "UpdateRound" {
					t.Errorf("unexpected trace: %v", trace)
				}
			},
		},
		{
			name: "Failure - Round Not Found",
			payload: &roundtypes.ImportCreateJobInput{
				GuildID: guildID, RoundID: roundID,
			},
			setupRepo: func(f *FakeRepo) {
				f.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return nil, nil // Round not found
				}
			},
			assertFunc: func(t *testing.T, res results.OperationResult[roundtypes.CreateImportJobResult, error], repo *FakeRepo) {
				if res.Failure == nil {
					t.Fatal("expected failure")
				}
				if (*res.Failure).Error() != "round not found" {
					t.Errorf("expected error 'round not found', got %v", res.Failure)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewFakeRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			s := createTestService(repo)

			res, err := s.CreateImportJob(ctx, tt.payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.assertFunc(t, res, repo)
		})
	}
}

// =============================================================================
// TEST: ScorecardURLRequested
// =============================================================================

func TestRoundService_ScorecardURLRequested(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-1")
	roundID := sharedtypes.RoundID(uuid.New())

	tests := []struct {
		name       string
		setupRepo  func(f *FakeRepo)
		payload    *roundtypes.ImportCreateJobInput
		assertFunc func(t *testing.T, res results.OperationResult[roundtypes.CreateImportJobResult, error])
	}{
		{
			name: "Success - Valid UDisc URL",
			payload: &roundtypes.ImportCreateJobInput{
				GuildID: guildID, RoundID: roundID, UDiscURL: "https://udisc.com/scorecards/12345",
			},
			setupRepo: func(f *FakeRepo) {
				f.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: r, GuildID: g}, nil
				}
				f.UpdateRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID, r *roundtypes.Round) (*roundtypes.Round, error) {
					if r.ImportType != "url" {
						return nil, errors.New("expected import type url")
					}
					return r, nil
				}
			},
			assertFunc: func(t *testing.T, res results.OperationResult[roundtypes.CreateImportJobResult, error]) {
				if res.Success == nil {
					t.Fatalf("expected success, got failure: %+v", res.Failure)
				}
				success := res.Success
				if (*success).Job.UDiscURL == "" {
					t.Error("expected UDiscURL to be set")
				}
			},
		},
		{
			name: "Failure - Invalid URL Domain",
			payload: &roundtypes.ImportCreateJobInput{
				GuildID: guildID, RoundID: roundID, UDiscURL: "https://google.com/scorecards",
			},
			setupRepo: func(f *FakeRepo) {
				f.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: r, GuildID: g}, nil
				}
			},
			assertFunc: func(t *testing.T, res results.OperationResult[roundtypes.CreateImportJobResult, error]) {
				if res.Failure == nil {
					t.Fatal("expected failure")
				}
				if !strings.Contains((*res.Failure).Error(), "invalid UDisc URL") {
					t.Errorf("expected error containing 'invalid UDisc URL', got %v", res.Failure)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewFakeRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			s := createTestService(repo)

			res, err := s.ScorecardURLRequested(ctx, tt.payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.assertFunc(t, res)
		})
	}
}

// =============================================================================
// TEST: ParseScorecard
// =============================================================================

func TestRoundService_ParseScorecard(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-1")
	roundID := sharedtypes.RoundID(uuid.New())

	// Simple CSV content for testing
	validCSV := []byte("PlayerName,+/-,Hole1\nAlice,-2,3\nBob,0,4")

	tests := []struct {
		name       string
		fileData   []byte
		setupRepo  func(f *FakeRepo)
		assertFunc func(t *testing.T, res results.OperationResult[roundtypes.ParsedScorecard, error], repo *FakeRepo)
	}{
		{
			name:     "Success - Valid CSV Data",
			fileData: validCSV,
			setupRepo: func(f *FakeRepo) {
				f.UpdateImportStatusFunc = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, importID string, status string, errorMessage string, errorCode string) error {
					return nil // Accept any status update
				}
			},
			assertFunc: func(t *testing.T, res results.OperationResult[roundtypes.ParsedScorecard, error], repo *FakeRepo) {
				if res.Success == nil {
					t.Fatalf("expected success, got failure: %+v", res.Failure)
				}
				payload := res.Success
				// The stub parser always returns 1 player score, regardless of input
				// unless we change the StubParser in fake_test.go.
				if len((*payload).PlayerScores) != 1 {
					t.Errorf("expected 1 player, got %d", len((*payload).PlayerScores))
				}

				// Verify we updated status to "parsing" then "parsed"
				trace := repo.Trace()
				if len(trace) < 2 {
					t.Error("expected at least 2 repo calls (UpdateImportStatus x2)")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewFakeRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			s := createTestService(repo)
			// Inject StubFactory for testing
			s.parserFactory = &StubFactory{}

			payload := &roundtypes.ImportParseScorecardInput{
				GuildID:  guildID,
				RoundID:  roundID,
				ImportID: "import-1",
				FileName: "test.csv",
				FileData: tt.fileData,
			}

			// We pass fileData directly in the input payload
			res, err := s.ParseScorecard(ctx, payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.assertFunc(t, res, repo)
		})
	}
}

// Helper to init service with dependencies
func createTestService(repo *FakeRepo) *RoundService {
	return &RoundService{
		repo:          repo,
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		metrics:       &roundmetrics.NoOpMetrics{},
		tracer:        noop.NewTracerProvider().Tracer("test"),
		parserFactory: parsers.NewFactory(),
	}
}
