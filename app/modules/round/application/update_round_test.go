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

func TestRoundService_ValidateAndProcessRoundUpdate(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())

	tests := []struct {
		name    string
		setup   func(*FakeTimeParser)
		payload roundevents.UpdateRoundRequestedPayloadV1
		want    results.OperationResult[*roundevents.RoundUpdateValidatedPayloadV1, *roundevents.RoundUpdateErrorPayloadV1]
		wantErr bool
	}{
		{
			name: "valid request",
			payload: roundevents.UpdateRoundRequestedPayloadV1{
				RoundID:  testRoundID,
				UserID:   sharedtypes.DiscordID("user123"),
				Title:    titlePtr("New Title"),
				Timezone: timezonePtr("America/Chicago"),
			},
			want: results.OperationResult[*roundevents.RoundUpdateValidatedPayloadV1, *roundevents.RoundUpdateErrorPayloadV1]{
				Success: ptr(&roundevents.RoundUpdateValidatedPayloadV1{
					RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayloadV1{
						RoundID: testRoundID,
						Title:   titlePtr("New Title"),
						UserID:  sharedtypes.DiscordID("user123"),
					},
				}),
			},
			wantErr: false,
		},
		{
			name: "valid request with time parsing",
			payload: roundevents.UpdateRoundRequestedPayloadV1{
				RoundID:   testRoundID,
				UserID:    sharedtypes.DiscordID("user123"),
				Title:     titlePtr("New Title"),
				StartTime: stringPtr("tomorrow at 2pm"),
				Timezone:  timezonePtr("America/Chicago"),
			},
			setup: func(p *FakeTimeParser) {
				p.ParseFn = func(s string, tz roundtypes.Timezone, clock roundutil.Clock) (int64, error) {
					// Return a fixed future time
					return time.Now().Add(24 * time.Hour).Unix(), nil
				}
			},
			want: results.OperationResult[*roundevents.RoundUpdateValidatedPayloadV1, *roundevents.RoundUpdateErrorPayloadV1]{
				Success: ptr(&roundevents.RoundUpdateValidatedPayloadV1{
					RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayloadV1{
						RoundID: testRoundID,
						Title:   titlePtr("New Title"),
						UserID:  sharedtypes.DiscordID("user123"),
						// StartTime will be dynamic
					},
				}),
			},
			wantErr: false,
		},
		{
			name: "invalid request - zero round ID",
			payload: roundevents.UpdateRoundRequestedPayloadV1{
				RoundID: sharedtypes.RoundID(uuid.Nil),
			},
			want: results.OperationResult[*roundevents.RoundUpdateValidatedPayloadV1, *roundevents.RoundUpdateErrorPayloadV1]{
				Failure: ptr(&roundevents.RoundUpdateErrorPayloadV1{
					RoundUpdateRequest: nil,
					Error:              "validation failed: round ID cannot be zero; at least one field to update must be provided",
				}),
			},
			wantErr: false,
		},
		{
			name: "invalid request - no fields to update",
			payload: roundevents.UpdateRoundRequestedPayloadV1{
				RoundID: testRoundID,
				UserID:  sharedtypes.DiscordID("user123"),
			},
			want: results.OperationResult[*roundevents.RoundUpdateValidatedPayloadV1, *roundevents.RoundUpdateErrorPayloadV1]{
				Failure: ptr(&roundevents.RoundUpdateErrorPayloadV1{
					RoundUpdateRequest: nil,
					Error:              "validation failed: at least one field to update must be provided",
				}),
			},
			wantErr: false,
		},
		{
			name: "invalid request - time parsing failed",
			payload: roundevents.UpdateRoundRequestedPayloadV1{
				RoundID:   testRoundID,
				UserID:    sharedtypes.DiscordID("user123"),
				Title:     titlePtr("New Title"),
				StartTime: stringPtr("invalid time"),
				Timezone:  timezonePtr("America/Chicago"),
			},
			setup: func(p *FakeTimeParser) {
				p.ParseFn = func(s string, tz roundtypes.Timezone, clock roundutil.Clock) (int64, error) {
					return 0, errors.New("invalid time format")
				}
			},
			want: results.OperationResult[*roundevents.RoundUpdateValidatedPayloadV1, *roundevents.RoundUpdateErrorPayloadV1]{
				Failure: ptr(&roundevents.RoundUpdateErrorPayloadV1{
					RoundUpdateRequest: nil,
					Error:              "validation failed: time parsing failed: invalid time format",
				}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &FakeTimeParser{}
			if tt.setup != nil {
				tt.setup(parser)
			}

			s := NewRoundService(slog.New(slog.NewTextHandler(nil, nil)), &roundmetrics.NoOpMetrics{}, noop.NewTracerProvider().Tracer("test"), NewFakeRepo(), nil, nil, &FakeRoundValidator{}, &StubFactory{})

			// For "valid request with time parsing", we need to dynamically set the expected StartTime in 'want'
			if tt.name == "valid request with time parsing" {
				fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC).Unix()
				parser.ParseFn = func(s string, tz roundtypes.Timezone, clock roundutil.Clock) (int64, error) {
					return fixedTime, nil
				}
				if tt.want.Success != nil {
					st := sharedtypes.StartTime(time.Unix(fixedTime, 0).UTC())
					(*tt.want.Success).RoundUpdateRequestPayload.StartTime = &st
				}
			}

			got, err := s.ValidateAndProcessRoundUpdate(context.Background(), tt.payload, parser)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundService.ValidateAndProcessRoundUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want.Success != nil {
				if got.Success == nil {
					t.Errorf("expected success, got nil")
				} else {
					if diff := cmp.Diff(*got.Success, *tt.want.Success, cmpopts.EquateComparable(sharedtypes.StartTime{})); diff != "" {
						t.Errorf("RoundService.ValidateAndProcessRoundUpdate() mismatch (-got +want):\n%s", diff)
					}
				}
			} else if tt.want.Failure != nil {
				if got.Failure == nil {
					t.Errorf("expected failure, got nil")
				} else {
					if diff := cmp.Diff(*got.Failure, *tt.want.Failure); diff != "" {
						t.Errorf("RoundService.ValidateAndProcessRoundUpdate() mismatch (-got +want):\n%s", diff)
					}
				}
			}
		})
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
	currentRound := &roundtypes.Round{
		ID:      testRoundID,
		Title:   roundtypes.Title("Old Title"),
		GuildID: testGuildID,
	}

	tests := []struct {
		name    string
		setup   func(*FakeRepo)
		payload roundevents.RoundUpdateValidatedPayloadV1
		want    results.OperationResult[*roundevents.RoundEntityUpdatedPayloadV1, *roundevents.RoundUpdateErrorPayloadV1]
		wantErr bool
	}{
		{
			name: "valid update",
			payload: roundevents.RoundUpdateValidatedPayloadV1{
				GuildID: testGuildID,
				RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					Title:   titlePtr("New Title"),
					UserID:  sharedtypes.DiscordID("user123"),
				},
			},
			setup: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID) (*roundtypes.Round, error) {
					return currentRound, nil
				}
				r.UpdateRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID, rnd *roundtypes.Round) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:      testRoundID,
						Title:   roundtypes.Title("New Title"),
						GuildID: testGuildID,
					}, nil
				}
			},
			want: results.OperationResult[*roundevents.RoundEntityUpdatedPayloadV1, *roundevents.RoundUpdateErrorPayloadV1]{
				Success: ptr(&roundevents.RoundEntityUpdatedPayloadV1{
					GuildID: testGuildID,
					Round: roundtypes.Round{
						ID:      testRoundID,
						Title:   roundtypes.Title("New Title"),
						GuildID: testGuildID,
					},
				}),
			},
			wantErr: false,
		},
		{
			name: "invalid update - round not found",
			payload: roundevents.RoundUpdateValidatedPayloadV1{
				GuildID: testGuildID,
				RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					Title:   titlePtr("New Title"),
					UserID:  sharedtypes.DiscordID("user123"),
				},
			},
			setup: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID) (*roundtypes.Round, error) {
					return currentRound, nil
				}
				r.UpdateRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID, rnd *roundtypes.Round) (*roundtypes.Round, error) {
					return nil, errors.New("round not found")
				}
			},
			want: results.OperationResult[*roundevents.RoundEntityUpdatedPayloadV1, *roundevents.RoundUpdateErrorPayloadV1]{
				Failure: ptr(&roundevents.RoundUpdateErrorPayloadV1{
					GuildID: testGuildID,
					RoundUpdateRequest: &roundevents.RoundUpdateRequestPayloadV1{
						GuildID: testGuildID,
						RoundID: testRoundID,
						Title:   titlePtr("New Title"),
						UserID:  sharedtypes.DiscordID("user123"),
					},
					Error: "failed to update round in database: round not found",
				}),
			},
			wantErr: false,
		},
		{
			name: "invalid update - update failed",
			payload: roundevents.RoundUpdateValidatedPayloadV1{
				GuildID: testGuildID,
				RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					Title:   titlePtr("New Title"),
					UserID:  sharedtypes.DiscordID("user123"),
				},
			},
			setup: func(r *FakeRepo) {
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID) (*roundtypes.Round, error) {
					return currentRound, nil
				}
				r.UpdateRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID, rnd *roundtypes.Round) (*roundtypes.Round, error) {
					return nil, errors.New("update failed")
				}
			},
			want: results.OperationResult[*roundevents.RoundEntityUpdatedPayloadV1, *roundevents.RoundUpdateErrorPayloadV1]{
				Failure: ptr(&roundevents.RoundUpdateErrorPayloadV1{
					GuildID: testGuildID,
					RoundUpdateRequest: &roundevents.RoundUpdateRequestPayloadV1{
						GuildID: testGuildID,
						RoundID: testRoundID,
						Title:   titlePtr("New Title"),
						UserID:  sharedtypes.DiscordID("user123"),
					},
					Error: "failed to update round in database: update failed",
				}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewFakeRepo()
			if tt.setup != nil {
				tt.setup(repo)
			}
			s := NewRoundService(slog.New(slog.NewTextHandler(nil, nil)), &roundmetrics.NoOpMetrics{}, noop.NewTracerProvider().Tracer("test"), repo, nil, nil, &FakeRoundValidator{}, &StubFactory{})
			}

			got, err := s.UpdateRoundEntity(context.Background(), tt.payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error state: %v", err)
			}

			if tt.want.Success != nil {
				if got.Success == nil {
					t.Errorf("expected success, got nil")
				} else {
					if diff := cmp.Diff(*got.Success, *tt.want.Success, cmpopts.EquateComparable(sharedtypes.StartTime{})); diff != "" {
						t.Errorf("RoundService.UpdateRoundEntity() mismatch (-got +want):\n%s", diff)
					}
				}
			} else if tt.want.Failure != nil {
				if got.Failure == nil {
					t.Errorf("expected failure, got nil")
				} else {
					if diff := cmp.Diff(*got.Failure, *tt.want.Failure); diff != "" {
						t.Errorf("RoundService.UpdateRoundEntity() mismatch (-got +want):\n%s", diff)
					}
				}
			}
		})
	}
}

func TestRoundService_UpdateScheduledRoundEvents(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testStartUpdateTime := sharedtypes.StartTime(time.Now().UTC().Add(2 * time.Hour))

	tests := []struct {
		name    string
		payload roundevents.RoundScheduleUpdatePayloadV1
		setup   func(*FakeRepo, *FakeQueueService)
		want    results.OperationResult[*roundevents.RoundScheduleUpdatePayloadV1, *roundevents.RoundUpdateErrorPayloadV1]
		wantErr bool
	}{
		{
			name: "valid update",
			payload: roundevents.RoundScheduleUpdatePayloadV1{
				GuildID:   sharedtypes.GuildID("guild-123"),
				RoundID:   testRoundID,
				Title:     roundtypes.Title("New Title"),
				StartTime: &testStartUpdateTime,
				Location:  roundtypes.Location("New Location"),
			},
			setup: func(r *FakeRepo, q *FakeQueueService) {
				q.CancelRoundJobsFunc = func(ctx context.Context, roundID sharedtypes.RoundID) error { return nil }
				r.GetEventMessageIDFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID) (string, error) { return "event123", nil }
				r.GetRoundFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:    testRoundID,
						Title: roundtypes.Title("Old Title"),
					}, nil
				}
				q.ScheduleRoundReminderFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, t time.Time, p roundevents.DiscordReminderPayloadV1) error {
					return nil
				}
				q.ScheduleRoundStartFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, t time.Time, p roundevents.RoundStartedPayloadV1) error {
					return nil
				}
			},
			want: results.OperationResult[*roundevents.RoundScheduleUpdatePayloadV1, *roundevents.RoundUpdateErrorPayloadV1]{
				Success: ptr(&roundevents.RoundScheduleUpdatePayloadV1{
					GuildID:   sharedtypes.GuildID("guild-123"),
					RoundID:   testRoundID,
					Title:     roundtypes.Title("New Title"),
					Location:  roundtypes.Location("New Location"),
					StartTime: &testStartUpdateTime,
				}),
			},
			wantErr: false,
		},
		{
			name: "error cancelling jobs",
			payload: roundevents.RoundScheduleUpdatePayloadV1{
				GuildID:   sharedtypes.GuildID("guild-123"),
				RoundID:   testRoundID,
				Title:     roundtypes.Title("New Title"),
				StartTime: &testStartUpdateTime,
				Location:  roundtypes.Location("New Location"),
			},
			setup: func(r *FakeRepo, q *FakeQueueService) {
				q.CancelRoundJobsFunc = func(ctx context.Context, roundID sharedtypes.RoundID) error {
					return errors.New("cancel jobs failed")
				}
			},
			want: results.OperationResult[*roundevents.RoundScheduleUpdatePayloadV1, *roundevents.RoundUpdateErrorPayloadV1]{
				Failure: ptr(&roundevents.RoundUpdateErrorPayloadV1{
					RoundUpdateRequest: nil,
					Error:              "failed to cancel existing scheduled jobs: cancel jobs failed",
				}),
			},
			wantErr: false,
		},
		{
			name: "error getting event message ID",
			payload: roundevents.RoundScheduleUpdatePayloadV1{
				GuildID:   sharedtypes.GuildID("guild-123"),
				RoundID:   testRoundID,
				Title:     roundtypes.Title("New Title"),
				StartTime: &testStartUpdateTime,
				Location:  roundtypes.Location("New Location"),
			},
			setup: func(r *FakeRepo, q *FakeQueueService) {
				q.CancelRoundJobsFunc = func(ctx context.Context, roundID sharedtypes.RoundID) error { return nil }
				r.GetEventMessageIDFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, id sharedtypes.RoundID) (string, error) {
					return "", errors.New("event message ID not found")
				}
			},
			want: results.OperationResult[*roundevents.RoundScheduleUpdatePayloadV1, *roundevents.RoundUpdateErrorPayloadV1]{
				Failure: ptr(&roundevents.RoundUpdateErrorPayloadV1{
					RoundUpdateRequest: nil,
					Error:              "failed to get EventMessageID: event message ID not found",
				}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewFakeRepo()
			queue := NewFakeQueueService()
			if tt.setup != nil {
				tt.setup(repo, queue)
			}

			s := NewRoundService(slog.New(slog.NewTextHandler(nil, nil)), &roundmetrics.NoOpMetrics{}, noop.NewTracerProvider().Tracer("test"), repo, nil, queue, &FakeRoundValidator{}, &StubFactory{})

			got, err := s.UpdateScheduledRoundEvents(context.Background(), tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundService.UpdateScheduledRoundEvents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want.Success != nil {
				if got.Success == nil {
					t.Errorf("expected success, got nil")
				} else {
					if diff := cmp.Diff(*got.Success, *tt.want.Success, cmpopts.EquateComparable(sharedtypes.StartTime{})); diff != "" {
						t.Errorf("RoundService.UpdateScheduledRoundEvents() mismatch (-got +want):\n%s", diff)
					}
				}
			} else if tt.want.Failure != nil {
				if got.Failure == nil {
					t.Errorf("expected failure, got nil")
				} else {
					if diff := cmp.Diff(*got.Failure, *tt.want.Failure); diff != "" {
						t.Errorf("RoundService.UpdateScheduledRoundEvents() mismatch (-got +want):\n%s", diff)
					}
				}
			}
		})
	}
}
