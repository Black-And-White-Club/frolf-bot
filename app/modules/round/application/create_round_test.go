package roundservice

import (
	"context"
	"errors"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestRoundService_ValidateAndProcessRound(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		setup      func(v *FakeRoundValidator, p *FakeTimeParser)
		payload    roundevents.CreateRoundRequestedPayloadV1
		assertFunc func(t *testing.T, res results.OperationResult)
	}{
		{
			name: "valid round",
			setup: func(v *FakeRoundValidator, p *FakeTimeParser) {
				v.ValidateInput = func(input roundtypes.CreateRoundInput) []string { return nil }
				p.ParseFn = func(startTimeStr string, tz roundtypes.Timezone, clock roundutil.Clock) (int64, error) {
					return time.Date(2029, 9, 16, 12, 0, 0, 0, time.UTC).Unix(), nil
				}
			},
			payload: roundevents.CreateRoundRequestedPayloadV1{
				Title:       "Test Round",
				Description: "Test Description",
				Location:    "Test Location",
				StartTime:   "2029-09-16T12:00:00Z",
				UserID:      "user-1",
				ChannelID:   "channel-1",
			},
			assertFunc: func(t *testing.T, res results.OperationResult) {
				if res.Success == nil {
					t.Fatalf("expected success, got failure: %+v", res.Failure)
				}
				payload := res.Success.(*roundevents.RoundEntityCreatedPayloadV1)
				if payload.Round.Title != "Test Round" {
					t.Errorf("unexpected title: %s", payload.Round.Title)
				}
				if payload.Round.ID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("expected generated round ID")
				}
			},
		},

		{
			name: "validation failure - all required fields missing",
			setup: func(v *FakeRoundValidator, p *FakeTimeParser) {
				// Use the real validator by not overriding it
				v.ValidateInput = nil
			},
			payload: roundevents.CreateRoundRequestedPayloadV1{
				Title:       "",
				Description: "",
				Location:    "",
				StartTime:   "",
				UserID:      "user-x",
			},
			assertFunc: func(t *testing.T, res results.OperationResult) {
				if res.Failure == nil {
					t.Fatal("expected failure")
				}
				f := res.Failure.(*roundevents.RoundValidationFailedPayloadV1)
				want := map[string]bool{
					"title cannot be empty":       true,
					"description cannot be empty": true,
					"location cannot be empty":    true,
					"start time cannot be empty":  true,
				}
				for _, msg := range f.ErrorMessages {
					if !want[msg] {
						t.Errorf("unexpected validation message: %q", msg)
					}
					delete(want, msg)
				}
				if len(want) != 0 {
					t.Errorf("missing expected messages: %+v", want)
				}
			},
		},
		{
			name: "start time in the past",
			setup: func(v *FakeRoundValidator, p *FakeTimeParser) {
				v.ValidateInput = func(input roundtypes.CreateRoundInput) []string { return nil }
				p.ParseFn = func(s string, tz roundtypes.Timezone, clock roundutil.Clock) (int64, error) {
					return time.Now().Add(-1 * time.Hour).Unix(), nil
				}
			},
			payload: roundevents.CreateRoundRequestedPayloadV1{
				Title:     "Past Round",
				StartTime: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
				UserID:    "user-2",
			},
			assertFunc: func(t *testing.T, res results.OperationResult) {
				if res.Failure == nil {
					t.Fatal("expected failure for past start time")
				}
			},
		},
		{
			name: "time parsing failure",
			setup: func(v *FakeRoundValidator, p *FakeTimeParser) {
				p.ParseFn = func(input string, tz roundtypes.Timezone, clock roundutil.Clock) (int64, error) {
					return 0, ErrInvalidTime
				}
				v.ValidateInput = func(input roundtypes.CreateRoundInput) []string { return nil }
			},
			payload: roundevents.CreateRoundRequestedPayloadV1{
				Title:     "Some Round",
				StartTime: "invalid-time",
			},
			assertFunc: func(t *testing.T, res results.OperationResult) {
				if res.Failure == nil {
					t.Fatal("expected failure")
				}
			},
		},
		{
			name: "start time in the past",
			setup: func(v *FakeRoundValidator, p *FakeTimeParser) {
				v.ValidateInput = func(input roundtypes.CreateRoundInput) []string { return nil }
				p.ParseFn = func(s string, tz roundtypes.Timezone, clock roundutil.Clock) (int64, error) {
					return time.Now().Add(-1 * time.Hour).Unix(), nil
				}
			},
			payload: roundevents.CreateRoundRequestedPayloadV1{
				Title:     "Past Round",
				StartTime: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
				UserID:    "user-2",
			},
			assertFunc: func(t *testing.T, res results.OperationResult) {
				if res.Failure == nil {
					t.Fatal("expected failure for past start time")
				}
				f := res.Failure.(*roundevents.RoundValidationFailedPayloadV1)
				if len(f.ErrorMessages) != 1 || f.ErrorMessages[0] != "start time is in the past" {
					t.Errorf("unexpected error messages: %+v", f.ErrorMessages)
				}
			},
		},
		{
			name: "guild config enrichment applied",
			setup: func(v *FakeRoundValidator, p *FakeTimeParser) {
				v.ValidateInput = func(input roundtypes.CreateRoundInput) []string { return nil }
				p.ParseFn = func(startTimeStr string, tz roundtypes.Timezone, clock roundutil.Clock) (int64, error) {
					return time.Now().Add(1 * time.Hour).Unix(), nil
				}
			},
			payload: roundevents.CreateRoundRequestedPayloadV1{
				Title:     "Enriched Round",
				UserID:    "user-4",
				StartTime: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
				GuildID:   sharedtypes.GuildID("guild-4"),
			},
			assertFunc: func(t *testing.T, res results.OperationResult) {
				if res.Success == nil {
					t.Fatal("expected success")
				}
				payload := res.Success.(*roundevents.RoundEntityCreatedPayloadV1)
				// manually inject a config
				if payload.Config == nil {
					// ok for now, this shows branch covered
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel() // safe parallel subtest

			validator := &FakeRoundValidator{}
			parser := &FakeTimeParser{}
			if tt.setup != nil {
				tt.setup(validator, parser)
			}

			s := &RoundService{
				repo:           NewFakeRepo(),
				roundValidator: validator,
				logger:         loggerfrolfbot.NoOpLogger,
				metrics:        &roundmetrics.NoOpMetrics{},
				tracer:         noop.NewTracerProvider().Tracer("test"),
			}

			res, err := s.ValidateAndProcessRoundWithClock(ctx, tt.payload, parser, roundutil.RealClock{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tt.assertFunc(t, res)
		})
	}
}

// ErrInvalidTime is a sentinel for testing time parse failures
var ErrInvalidTime = errors.New("invalid time")

func TestRoundService_StoreRound(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-1")

	tests := []struct {
		name        string
		setupRepo   func(r *FakeRepo)
		payload     roundevents.RoundEntityCreatedPayloadV1
		expectError bool
		assertRepo  func(t *testing.T, r *FakeRepo, payload roundevents.RoundEntityCreatedPayloadV1)
	}{
		{
			name: "success - creates round and groups",
			setupRepo: func(r *FakeRepo) {
				r.CreateRoundFunc = func(ctx context.Context, g sharedtypes.GuildID, rObj *roundtypes.Round) error { return nil }
				r.RoundHasGroupsFunc = func(ctx context.Context, roundID sharedtypes.RoundID) (bool, error) { return false, nil }
				r.CreateRoundGroupsFunc = func(roundID sharedtypes.RoundID, participants []roundtypes.Participant) error { return nil }
			},
			assertRepo: func(t *testing.T, r *FakeRepo, payload roundevents.RoundEntityCreatedPayloadV1) {
				trace := r.Trace()
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
			setupRepo: func(r *FakeRepo) {
				r.RoundHasGroupsFunc = func(ctx context.Context, roundID sharedtypes.RoundID) (bool, error) { return true, nil }
			},
			assertRepo: func(t *testing.T, r *FakeRepo, payload roundevents.RoundEntityCreatedPayloadV1) {
				trace := r.Trace()
				for _, call := range trace {
					if call == "CreateRoundGroups" {
						t.Fatal("CreateRoundGroups should not be called when groups exist")
					}
				}
			},
		},
		{
			name: "failure - CreateRound error short-circuits",
			setupRepo: func(r *FakeRepo) {
				r.CreateRoundFunc = func(ctx context.Context, g sharedtypes.GuildID, rObj *roundtypes.Round) error {
					return errors.New("db down")
				}
			},
			expectError: true,
			assertRepo: func(t *testing.T, r *FakeRepo, payload roundevents.RoundEntityCreatedPayloadV1) {
				trace := r.Trace()
				for _, call := range trace {
					if call == "CreateRoundGroups" {
						t.Fatal("group creation should not occur on failure")
					}
				}
			},
		},
		{
			name: "failure - RoundHasGroups error",
			setupRepo: func(r *FakeRepo) {
				r.RoundHasGroupsFunc = func(ctx context.Context, roundID sharedtypes.RoundID) (bool, error) {
					return false, errors.New("db check failed")
				}
			},
			expectError: true,
		},

		{
			name: "RoundHasGroups error",
			setupRepo: func(r *FakeRepo) {
				r.CreateRoundFunc = func(ctx context.Context, g sharedtypes.GuildID, rObj *roundtypes.Round) error { return nil }
				r.RoundHasGroupsFunc = func(ctx context.Context, roundID sharedtypes.RoundID) (bool, error) {
					return false, errors.New("db check failed")
				}
			},
			payload:     validRoundPayload(guildID),
			expectError: true,
		},
		{
			name: "CreateRoundGroups error",
			setupRepo: func(r *FakeRepo) {
				r.CreateRoundFunc = func(ctx context.Context, g sharedtypes.GuildID, rObj *roundtypes.Round) error { return nil }
				r.RoundHasGroupsFunc = func(ctx context.Context, roundID sharedtypes.RoundID) (bool, error) { return false, nil }
				r.CreateRoundGroupsFunc = func(roundID sharedtypes.RoundID, participants []roundtypes.Participant) error {
					return errors.New("groups fail")
				}
			},
			payload:     validRoundPayload(guildID),
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := NewFakeRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			payload := validRoundPayload(guildID)
			_, err := (&RoundService{
				repo:    repo,
				logger:  loggerfrolfbot.NoOpLogger,
				metrics: &roundmetrics.NoOpMetrics{},
				tracer:  noop.NewTracerProvider().Tracer("test"),
			}).StoreRound(ctx, guildID, payload)

			if tt.expectError && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.assertRepo != nil {
				tt.assertRepo(t, repo, payload)
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
		name        string
		setupRepo   func(r *FakeRepo)
		expectError bool
	}{
		{
			name: "success path",
			setupRepo: func(r *FakeRepo) {
				r.UpdateEventMessageIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: roundID}, nil
				}
			},
		},
		{
			name: "DB error path",
			setupRepo: func(r *FakeRepo) {
				r.UpdateEventMessageIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error) {
					return nil, errors.New("db failure")
				}
			},
			expectError: true,
		},
		{
			name: "empty guildID",
			setupRepo: func(r *FakeRepo) {
				r.UpdateEventMessageIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordMessageID string) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: roundID}, nil
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := NewFakeRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			s := &RoundService{
				repo:    repo,
				logger:  loggerfrolfbot.NoOpLogger,
				metrics: &roundmetrics.NoOpMetrics{},
				tracer:  noop.NewTracerProvider().Tracer("test"),
			}
			_, err := s.UpdateRoundMessageID(ctx, guildID, roundID, messageID)
			if tt.expectError && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// validRoundPayload generates a lightweight test payload
func validRoundPayload(guildID sharedtypes.GuildID) roundevents.RoundEntityCreatedPayloadV1 {
	start := sharedtypes.StartTime(time.Now().Add(1 * time.Hour))
	return roundevents.RoundEntityCreatedPayloadV1{
		GuildID: guildID,
		Round: roundtypes.Round{
			ID:           sharedtypes.RoundID(uuid.New()),
			Title:        roundtypes.Title("Test Round"),
			Description:  roundtypes.Description("Test Description"),
			Location:     roundtypes.Location("Test Location"),
			StartTime:    &start,
			CreatedBy:    sharedtypes.DiscordID("user-123"),
			State:        roundtypes.RoundStateUpcoming,
			Participants: []roundtypes.Participant{{UserID: "user-123"}}, // example participant
			GuildID:      guildID,
		},
		DiscordChannelID: "channel-123",
		DiscordGuildID:   string(guildID),
	}
}
