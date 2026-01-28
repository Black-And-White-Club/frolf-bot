package roundservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

func ptrScore(s int) *sharedtypes.Score {
	v := sharedtypes.Score(s)
	return &v
}

func ptrTag(t int) *sharedtypes.TagNumber {
	v := sharedtypes.TagNumber(t)
	return &v
}

func TestRoundService_FinalizeRound(t *testing.T) {
	ctx := context.Background()
	roundID := sharedtypes.RoundID(uuid.New())
	guildID := sharedtypes.GuildID("guild-1")

	tests := []struct {
		name       string
		setupRepo  func(f *FakeRepo)
		payload    *roundtypes.FinalizeRoundInput
		wantTrace  []string
		assertFunc func(t *testing.T, res FinalizeRoundResult)
	}{
		{
			name: "success",
			setupRepo: func(f *FakeRepo) {
				f.UpdateRoundStateFunc = func(ctx context.Context, db bun.IDB, gid sharedtypes.GuildID, rid sharedtypes.RoundID, state roundtypes.RoundState) error {
					return nil
				}
				f.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:      r,
						GuildID: g,
						State:   roundtypes.RoundStateFinalized,
					}, nil
				}
				f.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return []roundtypes.Participant{
						{UserID: "user1", Score: ptrScore(3)},
					}, nil
				}
			},
			payload: &roundtypes.FinalizeRoundInput{
				GuildID: guildID,
				RoundID: roundID,
			},
			wantTrace: []string{"UpdateRoundState", "GetRound", "GetParticipants"},
			assertFunc: func(t *testing.T, res FinalizeRoundResult) {
				if res.IsFailure() {
					t.Fatalf("expected success, got failure: %v", res.Failure)
				}
				payload := res.Success
				if (*payload).Round.ID != roundID {
					t.Errorf("expected roundID %v, got %v", roundID, (*payload).Round.ID)
				}
				if len((*payload).Participants) != 1 {
					t.Errorf("expected 1 participant, got %d", len((*payload).Participants))
				}
			},
		},
		{
			name: "fail update round state",
			setupRepo: func(f *FakeRepo) {
				f.UpdateRoundStateFunc = func(ctx context.Context, db bun.IDB, gid sharedtypes.GuildID, rid sharedtypes.RoundID, state roundtypes.RoundState) error {
					return errors.New("db error")
				}
			},
			payload: &roundtypes.FinalizeRoundInput{
				GuildID: guildID,
				RoundID: roundID,
			},
			wantTrace: []string{"UpdateRoundState"},
			assertFunc: func(t *testing.T, res FinalizeRoundResult) {
				if res.IsSuccess() {
					t.Fatal("expected failure")
				}
				if res.Failure == nil {
					t.Error("expected failure error to be non-nil")
				}
			},
		},
		{
			name: "fail get round after state update",
			setupRepo: func(f *FakeRepo) {
				f.UpdateRoundStateFunc = func(ctx context.Context, db bun.IDB, gid sharedtypes.GuildID, rid sharedtypes.RoundID, state roundtypes.RoundState) error {
					return nil
				}
				f.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return nil, errors.New("db get error")
				}
			},
			payload: &roundtypes.FinalizeRoundInput{
				GuildID: guildID,
				RoundID: roundID,
			},
			wantTrace: []string{"UpdateRoundState", "GetRound"},
			assertFunc: func(t *testing.T, res FinalizeRoundResult) {
				if res.IsSuccess() {
					t.Fatal("expected failure")
				}
			},
		},
		{
			name: "fail get participants",
			setupRepo: func(f *FakeRepo) {
				f.UpdateRoundStateFunc = func(ctx context.Context, db bun.IDB, gid sharedtypes.GuildID, rid sharedtypes.RoundID, state roundtypes.RoundState) error {
					return nil
				}
				f.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:      r,
						GuildID: g,
						State:   roundtypes.RoundStateFinalized,
					}, nil
				}
				f.GetParticipantsFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, r sharedtypes.RoundID) ([]roundtypes.Participant, error) {
					return nil, errors.New("participants error")
				}
			},
			payload: &roundtypes.FinalizeRoundInput{
				GuildID: guildID,
				RoundID: roundID,
			},
			wantTrace: []string{"UpdateRoundState", "GetRound", "GetParticipants"},
			assertFunc: func(t *testing.T, res FinalizeRoundResult) {
				if res.IsSuccess() {
					t.Fatal("expected failure")
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

			s := &RoundService{
				repo:           repo,
				logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
				metrics:        &roundmetrics.NoOpMetrics{},
				tracer:         noop.NewTracerProvider().Tracer("test"),
				parserFactory:  &StubFactory{},
			}

			res, err := s.FinalizeRound(ctx, tt.payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tt.assertFunc(t, res)

			if tt.wantTrace != nil {
				if !reflect.DeepEqual(repo.Trace(), tt.wantTrace) {
					t.Errorf("expected trace %v, got %v", tt.wantTrace, repo.Trace())
				}
			}
		})
	}
}

func TestRoundService_NotifyScoreModule(t *testing.T) {
	ctx := context.Background()
	roundID := sharedtypes.RoundID(uuid.New())
	guildID := sharedtypes.GuildID("guild-1")

	tests := []struct {
		name       string
		payload    *roundtypes.FinalizeRoundResult
		assertFunc func(t *testing.T, res results.OperationResult[*roundtypes.Round, error])
	}{
		{
			name: "success singles",
			payload: &roundtypes.FinalizeRoundResult{
				Round: &roundtypes.Round{
					ID:      roundID,
					GuildID: guildID,
				},
				Participants: []roundtypes.Participant{
					{UserID: "user1", Score: ptrScore(3), TagNumber: ptrTag(1)},
					{UserID: "user2", Score: ptrScore(5), TagNumber: ptrTag(2)},
				},
			},
			assertFunc: func(t *testing.T, res results.OperationResult[*roundtypes.Round, error]) {
				if res.IsFailure() {
					t.Fatalf("expected success, got failure: %+v", res.Failure)
				}
				round := res.Success
				if (*round).ID != roundID {
					t.Errorf("expected roundID %v, got %v", roundID, (*round).ID)
				}
			},
		},
		{
			name: "failure no scores",
			payload: &roundtypes.FinalizeRoundResult{
				Round: &roundtypes.Round{
					ID:      roundID,
					GuildID: guildID,
				},
				Participants: []roundtypes.Participant{
					{UserID: "user1", Score: nil}, // No score
				},
			},
			assertFunc: func(t *testing.T, res results.OperationResult[*roundtypes.Round, error]) {
				if res.IsSuccess() {
					t.Fatal("expected failure")
				}
				if (*res.Failure).Error() != "no participants with submitted scores found" {
					t.Errorf("unexpected error: %v", res.Failure)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &RoundService{
				logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
				metrics:        &roundmetrics.NoOpMetrics{},
				tracer:         noop.NewTracerProvider().Tracer("test"),
				parserFactory:  &StubFactory{},
			}

			// Ensure Round.Participants matches Participants for the logic to work
			tt.payload.Round.Participants = tt.payload.Participants

			res, err := s.NotifyScoreModule(ctx, tt.payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tt.assertFunc(t, res)
		})
	}
}
