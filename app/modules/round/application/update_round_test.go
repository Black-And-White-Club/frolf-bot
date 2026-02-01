package roundservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestRoundService_ValidateRoundUpdate(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())

	tests := []struct {
		name    string
		setup   func(*FakeTimeParser)
		payload roundtypes.UpdateRoundRequest
		want    UpdateRoundResult
		wantErr bool
	}{
		{
			name: "valid request",
			payload: roundtypes.UpdateRoundRequest{
				RoundID:  testRoundID,
				UserID:   sharedtypes.DiscordID("user123"),
				Title:    stringPtr("New Title"),
				Timezone: stringPtr("America/Chicago"),
			},
			want: results.SuccessResult[*roundtypes.UpdateRoundResult, error](&roundtypes.UpdateRoundResult{
				Round: &roundtypes.Round{
					ID:    testRoundID,
					Title: roundtypes.Title("New Title"),
				},
			}),
			wantErr: false,
		},
		{
			name: "valid request with time parsing",
			payload: roundtypes.UpdateRoundRequest{
				RoundID:   testRoundID,
				UserID:    sharedtypes.DiscordID("user123"),
				Title:     stringPtr("New Title"),
				StartTime: stringPtr("tomorrow at 2pm"),
				Timezone:  stringPtr("America/Chicago"),
			},
			setup: func(p *FakeTimeParser) {
				p.ParseFn = func(s string, tz roundtypes.Timezone, clock roundutil.Clock) (int64, error) {
					// Return a fixed future time
					return time.Now().Add(24 * time.Hour).Unix(), nil
				}
			},
			want: results.SuccessResult[*roundtypes.UpdateRoundResult, error](&roundtypes.UpdateRoundResult{
				Round: &roundtypes.Round{
					ID:    testRoundID,
					Title: roundtypes.Title("New Title"),
					// StartTime will be dynamic
				},
			}),
			wantErr: false,
		},
		{
			name: "invalid request - zero round ID",
			payload: roundtypes.UpdateRoundRequest{
				RoundID: sharedtypes.RoundID(uuid.Nil),
			},
			want:    results.FailureResult[*roundtypes.UpdateRoundResult, error](errors.New("validation failed: round ID cannot be zero; at least one field to update must be provided")),
			wantErr: false,
		},
		{
			name: "invalid request - no fields to update",
			payload: roundtypes.UpdateRoundRequest{
				RoundID: testRoundID,
				UserID:  sharedtypes.DiscordID("user123"),
			},
			want:    results.FailureResult[*roundtypes.UpdateRoundResult, error](errors.New("validation failed: at least one field to update must be provided")),
			wantErr: false,
		},
		{
			name: "invalid request - time parsing failed",
			payload: roundtypes.UpdateRoundRequest{
				RoundID:   testRoundID,
				UserID:    sharedtypes.DiscordID("user123"),
				Title:     stringPtr("New Title"),
				StartTime: stringPtr("invalid time"),
				Timezone:  stringPtr("America/Chicago"),
			},
			setup: func(p *FakeTimeParser) {
				p.ParseFn = func(s string, tz roundtypes.Timezone, clock roundutil.Clock) (int64, error) {
					return 0, errors.New("invalid time format")
				}
			},
			want:    results.FailureResult[*roundtypes.UpdateRoundResult, error](errors.New("validation failed: time parsing failed: invalid time format")),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &FakeTimeParser{}
			if tt.setup != nil {
				tt.setup(parser)
			}

			repo := NewFakeRepo()
			s := NewRoundService(repo, nil, nil, nil, &roundmetrics.NoOpMetrics{}, slog.New(slog.NewTextHandler(io.Discard, nil)), noop.NewTracerProvider().Tracer("test"), &FakeRoundValidator{}, nil)

			// For "valid request with time parsing", we need to dynamically set the expected StartTime in 'want'
			if tt.name == "valid request with time parsing" {
				fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC).Unix()
				parser.ParseFn = func(s string, tz roundtypes.Timezone, clock roundutil.Clock) (int64, error) {
					return fixedTime, nil
				}
				if tt.want.Success != nil {
					st := sharedtypes.StartTime(time.Unix(fixedTime, 0).UTC())
					(*tt.want.Success).Round.StartTime = &st
				}

				// Use FakeClock to ensure the fixed time is considered in the future
				clock := &roundutil.FakeClock{
					NowUTCFn: func() time.Time {
						return time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
					},
				}
				got, err := s.ValidateRoundUpdateWithClock(context.Background(), &tt.payload, parser, clock)
				if (err != nil) != tt.wantErr {
					t.Errorf("RoundService.ValidateRoundUpdate() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				checkResult(t, got, tt.want)
				return
			}

			got, err := s.ValidateRoundUpdate(context.Background(), &tt.payload, parser)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundService.ValidateRoundUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			checkResult(t, got, tt.want)
		})
	}
}

func checkResult(t *testing.T, got, want UpdateRoundResult) {
	if want.Success != nil {
		if got.Success == nil {
			t.Errorf("expected success, got nil")
		} else {
			if diff := cmp.Diff(*got.Success, *want.Success, cmpopts.EquateComparable(sharedtypes.StartTime{})); diff != "" {
				t.Errorf("RoundService.ValidateRoundUpdate() mismatch (-got +want):\n%s", diff)
			}
		}
	} else if want.Failure != nil {
		if got.Failure == nil {
			t.Errorf("expected failure, got nil")
		} else {
			if (*got.Failure).Error() != (*want.Failure).Error() {
				t.Errorf("RoundService.ValidateRoundUpdate() failure mismatch: got %v, want %v", *got.Failure, *want.Failure)
			}
		}
	}
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}

// Helper function for title pointers
func titlePtr(t string) *roundtypes.Title {
	title := roundtypes.Title(t)
	return &title
}

// Helper function for timezone pointers
func timezonePtr(t string) *roundtypes.Timezone {
	timezone := roundtypes.Timezone(t)
	return &timezone
}

func TestRoundService_UpdateRoundEntity(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")

	tests := []struct {
		name    string
		setup   func(*FakeRepo)
		payload roundtypes.UpdateRoundRequest
		want    UpdateRoundResult
		wantErr bool
	}{
		{
			name: "successful update",
			payload: roundtypes.UpdateRoundRequest{
				GuildID: testGuildID,
				RoundID: testRoundID,
				Title:   stringPtr("Updated Title"),
			},
			setup: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:    id,
						Title: roundtypes.Title("Old Title"),
					}, nil
				}
				r.UpdateRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID, r *roundtypes.Round) (*roundtypes.Round, error) {
					return r, nil
				}
			},
			want: results.SuccessResult[*roundtypes.UpdateRoundResult, error](&roundtypes.UpdateRoundResult{
				Round: &roundtypes.Round{
					ID:    testRoundID,
					Title: roundtypes.Title("Updated Title"),
				},
			}),
			wantErr: false,
		},
		{
			name: "repo error",
			payload: roundtypes.UpdateRoundRequest{
				GuildID: testGuildID,
				RoundID: testRoundID,
				Title:   stringPtr("Updated Title"),
			},
			setup: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:    id,
						Title: roundtypes.Title("Old Title"),
					}, nil
				}
				r.UpdateRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, rID sharedtypes.RoundID, r *roundtypes.Round) (*roundtypes.Round, error) {
					return nil, errors.New("database error")
				}
			},
			want:    UpdateRoundResult{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewFakeRepo()
			if tt.setup != nil {
				tt.setup(repo)
			}

			s := NewRoundService(repo, nil, nil, nil, &roundmetrics.NoOpMetrics{}, slog.New(slog.NewTextHandler(io.Discard, nil)), noop.NewTracerProvider().Tracer("test"), &FakeRoundValidator{}, nil)

			got, err := s.UpdateRoundEntity(context.Background(), &tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundService.UpdateRoundEntity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want.Success != nil {
				if got.Success == nil {
					t.Errorf("expected success, got nil")
				} else {
					// We need to ignore some fields when comparing if needed, but for now let's try direct comparison
					if diff := cmp.Diff((*got.Success).Round.Title, (*tt.want.Success).Round.Title); diff != "" {
						t.Errorf("RoundService.UpdateRoundEntity() title mismatch (-got +want):\n%s", diff)
					}
					if diff := cmp.Diff((*got.Success).Round.ID, (*tt.want.Success).Round.ID); diff != "" {
						t.Errorf("RoundService.UpdateRoundEntity() ID mismatch (-got +want):\n%s", diff)
					}
				}
			} else if tt.want.Failure != nil {
				if got.Failure == nil {
					t.Errorf("expected failure, got nil")
				} else {
					if (*got.Failure).Error() != (*tt.want.Failure).Error() {
						t.Errorf("RoundService.UpdateRoundEntity() failure mismatch: got %v, want %v", *got.Failure, *tt.want.Failure)
					}
				}
			}
		})
	}
}

func TestRoundService_UpdateScheduledRoundEvents(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testStartTime := sharedtypes.StartTime(time.Now().Add(2 * time.Hour).UTC())

	tests := []struct {
		name    string
		setup   func(*FakeRepo, *FakeQueueService)
		payload roundtypes.UpdateScheduledRoundEventsRequest
		want    UpdateScheduledRoundEventsResult
		wantErr bool
	}{
		{
			name: "successful update",
			payload: roundtypes.UpdateScheduledRoundEventsRequest{
				GuildID:   testGuildID,
				RoundID:   testRoundID,
				Title:     stringPtr("New Title"),
				StartTime: &testStartTime,
			},
			setup: func(r *FakeRepo, q *FakeQueueService) {
				r.GetEventMessageIDFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID) (string, error) {
					return "msg-123", nil
				}
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:    id,
						Title: roundtypes.Title("Old Title"),
					}, nil
				}
				q.CancelRoundJobsFunc = func(ctx context.Context, rID sharedtypes.RoundID) error { return nil }
				q.ScheduleRoundReminderFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, t time.Time, p roundevents.DiscordReminderPayloadV1) error {
					return nil
				}
				q.ScheduleRoundStartFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, t time.Time, p roundevents.RoundStartedPayloadV1) error {
					return nil
				}
			},
			want:    results.SuccessResult[bool, error](true),
			wantErr: false,
		},
		{
			name: "error cancelling jobs",
			payload: roundtypes.UpdateScheduledRoundEventsRequest{
				GuildID:   testGuildID,
				RoundID:   testRoundID,
				StartTime: &testStartTime,
			},
			setup: func(r *FakeRepo, q *FakeQueueService) {
				q.CancelRoundJobsFunc = func(ctx context.Context, rID sharedtypes.RoundID) error {
					return errors.New("cancel error")
				}
			},
			want:    UpdateScheduledRoundEventsResult{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewFakeRepo()
			queue := NewFakeQueueService()
			if tt.setup != nil {
				tt.setup(repo, queue)
			}

			s := NewRoundService(repo, queue, nil, nil, &roundmetrics.NoOpMetrics{}, slog.New(slog.NewTextHandler(io.Discard, nil)), noop.NewTracerProvider().Tracer("test"), &FakeRoundValidator{}, nil)

			got, err := s.UpdateScheduledRoundEvents(context.Background(), &tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundService.UpdateScheduledRoundEvents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want.Success != nil {
				if got.Success == nil {
					t.Errorf("expected success, got nil")
				}
			} else if tt.want.Failure != nil {
				if got.Failure == nil {
					t.Errorf("expected failure, got nil")
				} else {
					if (*got.Failure).Error() != (*tt.want.Failure).Error() {
						t.Errorf("RoundService.UpdateScheduledRoundEvents() failure mismatch: got %v, want %v", *got.Failure, *tt.want.Failure)
					}
				}
			}
		})
	}
}
