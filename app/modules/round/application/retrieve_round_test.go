package roundservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestRoundService_GetRound(t *testing.T) {
	ctx := context.Background()
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testStartTime := sharedtypes.StartTime(time.Now())
	testEventType := roundtypes.EventType("Test Event Type")

	testRound := &roundtypes.Round{
		ID:          testRoundID,
		GuildID:     testGuildID,
		Title:       "Test Round",
		Description: "Test Description",
		Location:    "Test Location",
		EventType:   &testEventType,
		StartTime:   &testStartTime,
		Finalized:   false,
		CreatedBy:   "Test User",
		State:       "Test State",
	}

	tests := []struct {
		name      string
		setupFake func(*FakeRepo)
		verify    func(t *testing.T, res results.OperationResult[*roundtypes.Round, error], err error, fake *FakeRepo)
	}{
		{
			name: "successful retrieval",
			setupFake: func(f *FakeRepo) {
				f.GetRoundFunc = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return testRound, nil
				}
			},
			verify: func(t *testing.T, res results.OperationResult[*roundtypes.Round, error], err error, fake *FakeRepo) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !res.IsSuccess() {
					t.Fatal("expected success result")
				}
				if (*res.Success).ID != testRoundID {
					t.Errorf("expected round ID %s, got %s", testRoundID, (*res.Success).ID)
				}
			},
		},
		{
			name: "error retrieving round",
			setupFake: func(f *FakeRepo) {
				f.GetRoundFunc = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error) {
					return nil, errors.New("database error")
				}
			},
			verify: func(t *testing.T, res results.OperationResult[*roundtypes.Round, error], err error, fake *FakeRepo) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.IsSuccess() {
					t.Fatal("expected failure result")
				}
				if res.Failure == nil || (*res.Failure).Error() != "database error" {
					t.Errorf("expected error 'database error', got %v", res.Failure)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fakeRepo := NewFakeRepo()
			if tt.setupFake != nil {
				tt.setupFake(fakeRepo)
			}

			s := &RoundService{
				repo:           fakeRepo,
				logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
				metrics:        &roundmetrics.NoOpMetrics{},
				tracer:         noop.NewTracerProvider().Tracer("test"),
				roundValidator: &FakeRoundValidator{},
				eventBus:       &FakeEventBus{},
				parserFactory:  &StubFactory{},
			}

			result, err := s.GetRound(ctx, testGuildID, testRoundID)
			if tt.verify != nil {
				tt.verify(t, result, err, fakeRepo)
			}
		})
	}
}

func TestRoundService_GetRoundsForGuild(t *testing.T) {
	ctx := context.Background()
	testGuildID := sharedtypes.GuildID("guild-123")

	tests := []struct {
		name      string
		setupFake func(*FakeRepo)
		verify    func(t *testing.T, rounds []*roundtypes.Round, err error, fake *FakeRepo)
	}{
		{
			name: "successful retrieval",
			setupFake: func(f *FakeRepo) {
				f.GetRoundsByGuildIDFunc = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, states ...roundtypes.RoundState) ([]*roundtypes.Round, error) {
					return []*roundtypes.Round{{ID: sharedtypes.RoundID(uuid.New())}}, nil
				}
			},
			verify: func(t *testing.T, rounds []*roundtypes.Round, err error, fake *FakeRepo) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(rounds) != 1 {
					t.Errorf("expected 1 round, got %d", len(rounds))
				}
			},
		},
		{
			name: "error retrieving rounds",
			setupFake: func(f *FakeRepo) {
				f.GetRoundsByGuildIDFunc = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, states ...roundtypes.RoundState) ([]*roundtypes.Round, error) {
					return nil, errors.New("database error")
				}
			},
			verify: func(t *testing.T, rounds []*roundtypes.Round, err error, fake *FakeRepo) {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if rounds != nil {
					t.Fatal("expected nil rounds on error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fakeRepo := NewFakeRepo()
			if tt.setupFake != nil {
				tt.setupFake(fakeRepo)
			}

			s := &RoundService{
				repo:           fakeRepo,
				logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
				metrics:        &roundmetrics.NoOpMetrics{},
				tracer:         noop.NewTracerProvider().Tracer("test"),
				roundValidator: &FakeRoundValidator{},
				eventBus:       &FakeEventBus{},
				parserFactory:  &StubFactory{},
			}

			rounds, err := s.GetRoundsForGuild(ctx, testGuildID)
			if tt.verify != nil {
				tt.verify(t, rounds, err, fakeRepo)
			}
		})
	}
}
