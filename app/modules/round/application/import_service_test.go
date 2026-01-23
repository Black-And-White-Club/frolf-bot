package roundservice

import (
	"context"
	"errors"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
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
		payload    roundevents.ScorecardUploadedPayloadV1
		assertFunc func(t *testing.T, res results.OperationResult, repo *FakeRepo)
	}{
		{
			name: "Success - CSV Upload",
			payload: roundevents.ScorecardUploadedPayloadV1{
				GuildID: guildID, RoundID: roundID, ImportID: "import-1", FileName: "test.csv", FileData: []byte("content"),
			},
			setupRepo: func(f *FakeRepo) {
				f.GetRoundFunc = func(ctx context.Context, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: r, GuildID: g}, nil
				}
				f.UpdateRoundFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, r *roundtypes.Round) (*roundtypes.Round, error) {
					if r.ImportStatus != "pending" {
						return nil, errors.New("expected pending status")
					}
					return r, nil
				}
			},
			assertFunc: func(t *testing.T, res results.OperationResult, repo *FakeRepo) {
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
			payload: roundevents.ScorecardUploadedPayloadV1{
				GuildID: guildID, RoundID: roundID,
			},
			setupRepo: func(f *FakeRepo) {
				f.GetRoundFunc = func(ctx context.Context, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return nil, nil // Round not found
				}
			},
			assertFunc: func(t *testing.T, res results.OperationResult, repo *FakeRepo) {
				if res.Failure == nil {
					t.Fatal("expected failure")
				}
				fail := res.Failure.(*roundevents.ImportFailedPayloadV1)
				if fail.ErrorCode != errCodeRoundNotFound {
					t.Errorf("expected error code %s, got %s", errCodeRoundNotFound, fail.ErrorCode)
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
		payload    roundevents.ScorecardURLRequestedPayloadV1
		assertFunc func(t *testing.T, res results.OperationResult)
	}{
		{
			name: "Success - Valid UDisc URL",
			payload: roundevents.ScorecardURLRequestedPayloadV1{
				GuildID: guildID, RoundID: roundID, UDiscURL: "https://udisc.com/scorecards/12345",
			},
			setupRepo: func(f *FakeRepo) {
				f.GetRoundFunc = func(ctx context.Context, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: r, GuildID: g}, nil
				}
				f.UpdateRoundFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, r *roundtypes.Round) (*roundtypes.Round, error) {
					if r.ImportType != "url" {
						return nil, errors.New("expected import type url")
					}
					return r, nil
				}
			},
			assertFunc: func(t *testing.T, res results.OperationResult) {
				if res.Success == nil {
					t.Fatalf("expected success, got failure: %+v", res.Failure)
				}
				success := res.Success.(*roundevents.ScorecardUploadedPayloadV1)
				if success.FileURL == "" {
					t.Error("expected FileURL to be set")
				}
			},
		},
		{
			name: "Failure - Invalid URL Domain",
			payload: roundevents.ScorecardURLRequestedPayloadV1{
				GuildID: guildID, RoundID: roundID, UDiscURL: "https://google.com/scorecards",
			},
			setupRepo: func(f *FakeRepo) {
				f.GetRoundFunc = func(ctx context.Context, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: r, GuildID: g}, nil
				}
			},
			assertFunc: func(t *testing.T, res results.OperationResult) {
				if res.Failure == nil {
					t.Fatal("expected failure")
				}
				fail := res.Failure.(*roundevents.ImportFailedPayloadV1)
				if fail.ErrorCode != errCodeInvalidUDiscURL {
					t.Errorf("expected error code %s, got %s", errCodeInvalidUDiscURL, fail.ErrorCode)
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
	invalidCSV := []byte("Not,A,Valid,CSV")

	tests := []struct {
		name       string
		fileData   []byte
		setupRepo  func(f *FakeRepo)
		assertFunc func(t *testing.T, res results.OperationResult, repo *FakeRepo)
	}{
		{
			name:     "Success - Valid CSV Data",
			fileData: validCSV,
			setupRepo: func(f *FakeRepo) {
				f.UpdateImportStatusFunc = func(guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, importID string, status string) error {
					return nil // Accept any status update
				}
			},
			assertFunc: func(t *testing.T, res results.OperationResult, repo *FakeRepo) {
				if res.Success == nil {
					t.Fatalf("expected success, got failure: %+v", res.Failure)
				}
				payload := res.Success.(*roundevents.ParsedScorecardPayloadV1)
				if len(payload.ParsedData.PlayerScores) != 2 {
					t.Errorf("expected 2 players, got %d", len(payload.ParsedData.PlayerScores))
				}

				// Verify we updated status to "parsing" then "parsed"
				trace := repo.Trace()
				if len(trace) < 2 {
					t.Error("expected at least 2 repo calls (UpdateImportStatus x2)")
				}
			},
		},
		{
			name:     "Failure - Parser Error (Missing Columns)",
			fileData: invalidCSV,
			setupRepo: func(f *FakeRepo) {
				f.UpdateImportStatusFunc = func(guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, importID string, status string) error {
					return nil
				}
			},
			assertFunc: func(t *testing.T, res results.OperationResult, repo *FakeRepo) {
				if res.Failure == nil {
					t.Fatal("expected failure")
				}
				fail := res.Failure.(*roundevents.ScorecardParseFailedPayloadV1)
				if fail.Error == "" {
					t.Error("expected error message in failure payload")
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

			payload := roundevents.ScorecardUploadedPayloadV1{
				GuildID:  guildID,
				RoundID:  roundID,
				ImportID: "import-1",
				FileName: "test.csv",
			}

			// We pass fileData directly to skip the downloadFile logic
			res, err := s.ParseScorecard(ctx, payload, tt.fileData)
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
		repo:    repo,
		logger:  loggerfrolfbot.NoOpLogger,
		metrics: &roundmetrics.NoOpMetrics{},
		tracer:  noop.NewTracerProvider().Tracer("test"),
	}
}
