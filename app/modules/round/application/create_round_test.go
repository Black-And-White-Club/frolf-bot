package roundservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestRoundService_ValidateRoundCreationWithClock(t *testing.T) {
	ctx := context.Background()

	// Valid input for testing
	validInput := &roundtypes.CreateRoundInput{
		Title:       roundtypes.Title("Test Round"),
		Description: descPtr("Test Description"),
		Location:    roundtypes.Location("Test Location"),
		StartTime:   "2029-09-16T12:00:00Z",
		UserID:      sharedtypes.DiscordID("user-1"),
		ChannelID:   "channel-1",
		Timezone:    "America/Chicago",
		GuildID:     sharedtypes.GuildID("guild-1"),
	}

	tests := []struct {
		name      string
		setupFake func(*FakeRoundValidator, *FakeTimeParser)
		input     *roundtypes.CreateRoundInput
		verify    func(t *testing.T, res CreateRoundResult, infraErr error, validator *FakeRoundValidator, parser *FakeTimeParser)
	}{
		{
			name:  "success - valid round",
			input: validInput,
			setupFake: func(v *FakeRoundValidator, p *FakeTimeParser) {
				v.ValidateInput = func(input roundtypes.CreateRoundInput) []string { return nil }
				p.ParseFn = func(startTimeStr string, tz roundtypes.Timezone, clock roundutil.Clock) (int64, error) {
					return time.Date(2029, 9, 16, 12, 0, 0, 0, time.UTC).Unix(), nil
				}
			},
			verify: func(t *testing.T, res CreateRoundResult, infraErr error, validator *FakeRoundValidator, parser *FakeTimeParser) {
				if infraErr != nil {
					t.Fatalf("unexpected infrastructure error: %v", infraErr)
				}
				if res.Failure != nil {
					t.Fatalf("expected domain success, got failure: %v", *res.Failure)
				}
				if res.Success == nil {
					t.Fatal("expected success payload")
				}
				payload := *res.Success
				if payload.Round.Title != "Test Round" {
					t.Errorf("expected success payload with title 'Test Round', got %v", payload.Round.Title)
				}
				if payload.Round.ID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("expected generated round ID")
				}
			},
		},
		{
			name: "validation failure - all required fields missing",
			input: &roundtypes.CreateRoundInput{
				Title:       "",
				Description: nil,
				Location:    "",
				StartTime:   "",
				UserID:      "user-x",
			},
			setupFake: func(v *FakeRoundValidator, p *FakeTimeParser) {
				v.ValidateInput = func(input roundtypes.CreateRoundInput) []string {
					return []string{"Title is required"}
				}
			},
			verify: func(t *testing.T, res CreateRoundResult, infraErr error, validator *FakeRoundValidator, parser *FakeTimeParser) {
				if res.IsSuccess() {
					t.Fatal("expected domain validation failure, but got success")
				}
				if res.Failure == nil {
					t.Fatal("expected failure")
				}
			},
		},
		{
			name: "start time in the past",
			input: &roundtypes.CreateRoundInput{
				Title:     roundtypes.Title("Past Round"),
				StartTime: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
				UserID:    sharedtypes.DiscordID("user-2"),
			},
			setupFake: func(v *FakeRoundValidator, p *FakeTimeParser) {
				v.ValidateInput = func(input roundtypes.CreateRoundInput) []string { return nil }
				p.ParseFn = func(s string, tz roundtypes.Timezone, clock roundutil.Clock) (int64, error) {
					return time.Now().Add(-1 * time.Hour).Unix(), nil
				}
			},
			verify: func(t *testing.T, res CreateRoundResult, infraErr error, validator *FakeRoundValidator, parser *FakeTimeParser) {
				if res.IsSuccess() {
					t.Fatal("expected failure for past start time")
				}
				if res.Failure == nil {
					t.Fatal("expected failure")
				}
			},
		},
		{
			name: "time parsing failure",
			input: &roundtypes.CreateRoundInput{
				Title:     roundtypes.Title("Some Round"),
				StartTime: "invalid-time",
				UserID:    sharedtypes.DiscordID("user-1"),
			},
			setupFake: func(v *FakeRoundValidator, p *FakeTimeParser) {
				p.ParseFn = func(input string, tz roundtypes.Timezone, clock roundutil.Clock) (int64, error) {
					return 0, ErrInvalidTime
				}
				v.ValidateInput = func(input roundtypes.CreateRoundInput) []string { return nil }
			},
			verify: func(t *testing.T, res CreateRoundResult, infraErr error, validator *FakeRoundValidator, parser *FakeTimeParser) {
				if res.IsSuccess() {
					t.Fatal("expected failure")
				}
				if res.Failure == nil {
					t.Fatal("expected failure")
				}
			},
		},
		{
			name: "guild config enrichment applied",
			input: &roundtypes.CreateRoundInput{
				Title:     roundtypes.Title("Enriched Round"),
				UserID:    sharedtypes.DiscordID("user-4"),
				StartTime: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
				GuildID:   sharedtypes.GuildID("guild-4"),
			},
			setupFake: func(v *FakeRoundValidator, p *FakeTimeParser) {
				v.ValidateInput = func(input roundtypes.CreateRoundInput) []string { return nil }
				p.ParseFn = func(startTimeStr string, tz roundtypes.Timezone, clock roundutil.Clock) (int64, error) {
					return time.Now().Add(1 * time.Hour).Unix(), nil
				}
			},
			verify: func(t *testing.T, res CreateRoundResult, infraErr error, validator *FakeRoundValidator, parser *FakeTimeParser) {
				if infraErr != nil {
					t.Fatalf("unexpected infrastructure error: %v", infraErr)
				}
				if res.Failure != nil {
					t.Fatalf("expected domain success, got failure: %v", *res.Failure)
				}
				if res.Success == nil {
					t.Fatal("expected success")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			validator := &FakeRoundValidator{}
			parser := &FakeTimeParser{}
			if tt.setupFake != nil {
				tt.setupFake(validator, parser)
			}

			s := &RoundService{
				repo:           NewFakeRepo(),
				roundValidator: validator,
				logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
				metrics:        &roundmetrics.NoOpMetrics{},
				tracer:         noop.NewTracerProvider().Tracer("test"),
				parserFactory:  &StubFactory{},
			}

			res, err := s.ValidateRoundCreationWithClock(ctx, tt.input, parser, roundutil.RealClock{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.verify != nil {
				tt.verify(t, res, err, validator, parser)
			}
		})
	}
}

func startTimePtr(t time.Time) *sharedtypes.StartTime {
	st := sharedtypes.StartTime(t)
	return &st
}

func descPtr(s string) *roundtypes.Description {
	d := roundtypes.Description(s)
	return &d
}

// ErrInvalidTime is a sentinel for testing time parse failures
var ErrInvalidTime = errors.New("invalid time")

func TestRoundService_StoreRound(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-1")

	tests := []struct {
		name           string
		setupFake      func(*FakeRepo)
		round          *roundtypes.Round
		expectInfraErr bool // For network/db connection errors
		verify         func(t *testing.T, res CreateRoundResult, infraErr error, fake *FakeRepo)
	}{
		{
			name: "success - creates round and groups",
			setupFake: func(r *FakeRepo) {
				r.CreateRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rObj *roundtypes.Round) error { return nil }
				r.RoundHasGroupsFunc = func(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) (bool, error) { return false, nil }
				r.CreateRoundGroupsFunc = func(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID, participants []roundtypes.Participant) error {
					return nil
				}
			},
			round: &roundtypes.Round{
				ID:           sharedtypes.RoundID(uuid.New()),
				Title:        roundtypes.Title("Test Round"),
				Description:  roundtypes.Description("Test Description"),
				Location:     roundtypes.Location("Test Location"),
				StartTime:    startTimePtr(time.Now().Add(1 * time.Hour)),
				CreatedBy:    sharedtypes.DiscordID("user-1"),
				State:        roundtypes.RoundStateUpcoming,
				Participants: []roundtypes.Participant{{UserID: "user-123"}},
			},
			verify: func(t *testing.T, res CreateRoundResult, infraErr error, fake *FakeRepo) {
				if infraErr != nil {
					t.Fatalf("unexpected infrastructure error: %v", infraErr)
				}
				if res.Failure != nil {
					t.Fatalf("expected domain success, got failure: %v", *res.Failure)
				}
				if res.Success == nil {
					t.Fatal("expected success payload")
				}
				trace := fake.Trace()
				expectedTrace := []string{"CreateRound", "RoundHasGroups", "CreateRoundGroups"}
				if len(trace) != len(expectedTrace) {
					t.Fatalf("expected trace length %d, got %d: %v", len(expectedTrace), len(trace), trace)
				}
				for i, call := range expectedTrace {
					if trace[i] != call {
						t.Fatalf("expected trace[%d] = %s, got %s", i, call, trace[i])
					}
				}
			},
		},
		{
			name: "success - skips group creation when groups exist",
			setupFake: func(r *FakeRepo) {
				r.CreateRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rObj *roundtypes.Round) error { return nil }
				r.RoundHasGroupsFunc = func(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) (bool, error) { return true, nil }
			},
			round: &roundtypes.Round{
				ID:           sharedtypes.RoundID(uuid.New()),
				Title:        roundtypes.Title("Test Round"),
				Description:  roundtypes.Description("Test Description"),
				Location:     roundtypes.Location("Test Location"),
				StartTime:    startTimePtr(time.Now().Add(1 * time.Hour)),
				CreatedBy:    sharedtypes.DiscordID("user-1"),
				State:        roundtypes.RoundStateUpcoming,
				Participants: []roundtypes.Participant{{UserID: "user-123"}},
			},
			verify: func(t *testing.T, res CreateRoundResult, infraErr error, fake *FakeRepo) {
				if infraErr != nil {
					t.Fatalf("unexpected infrastructure error: %v", infraErr)
				}
				if res.Failure != nil {
					t.Fatalf("expected domain success, got failure: %v", *res.Failure)
				}
				if res.Success == nil {
					t.Fatal("expected success payload")
				}
				trace := fake.Trace()
				for _, call := range trace {
					if call == "CreateRoundGroups" {
						t.Fatal("CreateRoundGroups should not be called when groups exist")
					}
				}
			},
		},
		{
			name: "failure - CreateRound error short-circuits",
			setupFake: func(r *FakeRepo) {
				r.CreateRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rObj *roundtypes.Round) error {
					return errors.New("db down")
				}
			},
			round: &roundtypes.Round{
				ID:           sharedtypes.RoundID(uuid.New()),
				Title:        roundtypes.Title("Test Round"),
				Description:  roundtypes.Description("Test Description"),
				Location:     roundtypes.Location("Test Location"),
				StartTime:    startTimePtr(time.Now().Add(1 * time.Hour)),
				CreatedBy:    sharedtypes.DiscordID("user-1"),
				State:        roundtypes.RoundStateUpcoming,
				Participants: []roundtypes.Participant{{UserID: "user-123"}},
			},
			expectInfraErr: true,
			verify: func(t *testing.T, res CreateRoundResult, infraErr error, fake *FakeRepo) {
				if !strings.Contains(infraErr.Error(), "db down") {
					t.Fatalf("expected 'db down' error, got: %v", infraErr)
				}
				trace := fake.Trace()
				for _, call := range trace {
					if call == "CreateRoundGroups" {
						t.Fatal("group creation should not occur on failure")
					}
				}
			},
		},
		{
			name: "failure - RoundHasGroups error",
			setupFake: func(r *FakeRepo) {
				r.CreateRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rObj *roundtypes.Round) error { return nil }
				r.RoundHasGroupsFunc = func(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) (bool, error) {
					return false, errors.New("db check failed")
				}
			},
			round: &roundtypes.Round{
				ID:           sharedtypes.RoundID(uuid.New()),
				Title:        roundtypes.Title("Test Round"),
				Description:  roundtypes.Description("Test Description"),
				Location:     roundtypes.Location("Test Location"),
				StartTime:    startTimePtr(time.Now().Add(1 * time.Hour)),
				CreatedBy:    sharedtypes.DiscordID("user-1"),
				State:        roundtypes.RoundStateUpcoming,
				Participants: []roundtypes.Participant{{UserID: "user-123"}},
			},
			expectInfraErr: true,
		},
		{
			name: "failure - CreateRoundGroups error",
			setupFake: func(r *FakeRepo) {
				r.CreateRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rObj *roundtypes.Round) error { return nil }
				r.RoundHasGroupsFunc = func(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) (bool, error) { return false, nil }
				r.CreateRoundGroupsFunc = func(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID, participants []roundtypes.Participant) error {
					return errors.New("groups fail")
				}
			},
			round: &roundtypes.Round{
				ID:           sharedtypes.RoundID(uuid.New()),
				Title:        roundtypes.Title("Test Round"),
				Description:  roundtypes.Description("Test Description"),
				Location:     roundtypes.Location("Test Location"),
				StartTime:    startTimePtr(time.Now().Add(1 * time.Hour)),
				CreatedBy:    sharedtypes.DiscordID("user-1"),
				State:        roundtypes.RoundStateUpcoming,
				Participants: []roundtypes.Participant{{UserID: "user-123"}},
			},
			expectInfraErr: true,
		},
		{
			name: "failure - invalid round data",
			setupFake: func(r *FakeRepo) {
				// No repo setup needed - validation should fail before DB calls
			},
			round: &roundtypes.Round{
				ID:        sharedtypes.RoundID(uuid.New()),
				Title:     "", // Empty title
				StartTime: startTimePtr(time.Now().Add(1 * time.Hour)),
				CreatedBy: sharedtypes.DiscordID("user-1"),
			},
			verify: func(t *testing.T, res CreateRoundResult, infraErr error, fake *FakeRepo) {
				if res.IsSuccess() {
					t.Fatal("expected domain validation failure, but got success")
				}
				if res.Failure == nil {
					t.Fatal("expected failure")
				}
			},
		},
		{
			name: "failure - nil start time",
			setupFake: func(r *FakeRepo) {
				// No repo setup needed - validation should fail before DB calls
			},
			round: &roundtypes.Round{
				ID:        sharedtypes.RoundID(uuid.New()),
				Title:     roundtypes.Title("Test Round"),
				StartTime: nil, // Nil start time
				CreatedBy: sharedtypes.DiscordID("user-1"),
			},
			verify: func(t *testing.T, res CreateRoundResult, infraErr error, fake *FakeRepo) {
				if res.IsSuccess() {
					t.Fatal("expected domain validation failure, but got success")
				}
				if res.Failure == nil {
					t.Fatal("expected failure")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := NewFakeRepo()
			if tt.setupFake != nil {
				tt.setupFake(repo)
			}

			s := &RoundService{
				repo:          repo,
				logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
				metrics:       &roundmetrics.NoOpMetrics{},
				tracer:        noop.NewTracerProvider().Tracer("test"),
				parserFactory: &StubFactory{},
			}

			res, err := s.StoreRound(ctx, tt.round, guildID)

			if tt.expectInfraErr && err == nil {
				t.Fatal("expected infra error but got nil")
			}
			if !tt.expectInfraErr && err != nil {
				t.Fatalf("unexpected infra error: %v", err)
			}

			if tt.verify != nil {
				tt.verify(t, res, err, repo)
			}
		})
	}
}

func TestRoundService_UpdateRoundMessageID(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-1")
	roundID := sharedtypes.RoundID(uuid.New())
	messageID := "discord-123"

	tests := []struct {
		name           string
		setupFake      func(*FakeRepo)
		expectInfraErr bool // For network/db connection errors
		verify         func(t *testing.T, result *roundtypes.Round, infraErr error, fake *FakeRepo)
	}{
		{
			name: "success path",
			setupFake: func(r *FakeRepo) {
				r.UpdateEventMessageIDFunc = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: roundID}, nil
				}
			},
			verify: func(t *testing.T, result *roundtypes.Round, infraErr error, fake *FakeRepo) {
				if infraErr != nil {
					t.Fatalf("unexpected infrastructure error: %v", infraErr)
				}
				if result == nil {
					t.Fatal("expected round result")
				}
				if result.ID != roundID {
					t.Errorf("expected round ID %s, got %s", roundID, result.ID)
				}
			},
		},
		{
			name: "DB error path",
			setupFake: func(r *FakeRepo) {
				r.UpdateEventMessageIDFunc = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error) {
					return nil, errors.New("db failure")
				}
			},
			expectInfraErr: true,
			verify: func(t *testing.T, result *roundtypes.Round, infraErr error, fake *FakeRepo) {
				if infraErr == nil || !strings.Contains(infraErr.Error(), "db failure") {
					t.Fatalf("expected 'db failure' error, got: %v", infraErr)
				}
			},
		},
		{
			name: "empty guildID",
			setupFake: func(r *FakeRepo) {
				r.UpdateEventMessageIDFunc = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: roundID}, nil
				}
			},
			verify: func(t *testing.T, result *roundtypes.Round, infraErr error, fake *FakeRepo) {
				if infraErr != nil {
					t.Fatalf("unexpected infrastructure error: %v", infraErr)
				}
				if result == nil {
					t.Fatal("expected round result")
				}
				if result.ID != roundID {
					t.Errorf("expected round ID %s, got %s", roundID, result.ID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := NewFakeRepo()
			if tt.setupFake != nil {
				tt.setupFake(repo)
			}
			s := &RoundService{
				repo:          repo,
				logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
				metrics:       &roundmetrics.NoOpMetrics{},
				tracer:        noop.NewTracerProvider().Tracer("test"),
				parserFactory: &StubFactory{},
			}
			result, err := s.UpdateRoundMessageID(ctx, guildID, roundID, messageID)
			if tt.expectInfraErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.expectInfraErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.verify != nil {
				tt.verify(t, result, err, repo)
			}
		})
	}
}
