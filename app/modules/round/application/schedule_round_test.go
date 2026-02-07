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
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
)

func ptr[T any](v T) *T {
	return &v
}

func TestRoundService_ScheduleRoundEvents(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}

	now := time.Now().UTC()
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("test-guild")
	testRoundTitle := "Test Round"
	testLocation := "Test Location"
	testDescription := "Test Description"
	testMessageID := "12345"

	tests := []struct {
		name            string
		startTimeOffset time.Duration
		setup           func(*FakeQueueService)
		want            results.OperationResult[*roundtypes.ScheduleRoundEventsResult, error]
		wantErr         bool
	}{
		{
			name:            "successful scheduling",
			startTimeOffset: 2 * time.Hour,
			setup: func(q *FakeQueueService) {
				q.CancelRoundJobsFunc = func(ctx context.Context, rID sharedtypes.RoundID) error { return nil }
				q.ScheduleRoundReminderFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, t time.Time, p roundevents.DiscordReminderPayloadV1) error {
					return nil
				}
				q.ScheduleRoundStartFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, t time.Time, p roundevents.RoundStartedPayloadV1) error {
					return nil
				}
			},
			want: results.OperationResult[*roundtypes.ScheduleRoundEventsResult, error]{
				Success: ptr(&roundtypes.ScheduleRoundEventsResult{
					RoundID:        testRoundID,
					GuildID:        testGuildID,
					Title:          testRoundTitle,
					Description:    testDescription,
					Location:       testLocation,
					StartTime:      *startTimePtr(now.Add(2 * time.Hour)),
					EventMessageID: testMessageID,
				}),
			},
			wantErr: false,
		},
		{
			name:            "error cancelling jobs",
			startTimeOffset: 2 * time.Hour,
			setup: func(q *FakeQueueService) {
				q.CancelRoundJobsFunc = func(ctx context.Context, rID sharedtypes.RoundID) error {
					return errors.New("job cancellation error")
				}
			},
			want:    results.OperationResult[*roundtypes.ScheduleRoundEventsResult, error]{},
			wantErr: true,
		},
		{
			name:            "error scheduling reminder",
			startTimeOffset: 2 * time.Hour,
			setup: func(q *FakeQueueService) {
				q.CancelRoundJobsFunc = func(ctx context.Context, rID sharedtypes.RoundID) error { return nil }
				q.ScheduleRoundReminderFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, t time.Time, p roundevents.DiscordReminderPayloadV1) error {
					return errors.New("reminder scheduling error")
				}
			},
			want:    results.OperationResult[*roundtypes.ScheduleRoundEventsResult, error]{},
			wantErr: true,
		},
		{
			name:            "error scheduling round start",
			startTimeOffset: 2 * time.Hour,
			setup: func(q *FakeQueueService) {
				q.CancelRoundJobsFunc = func(ctx context.Context, rID sharedtypes.RoundID) error { return nil }
				q.ScheduleRoundReminderFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, t time.Time, p roundevents.DiscordReminderPayloadV1) error {
					return nil
				}
				// q.ScheduleRoundStartFunc = func(ctx context.Context, g sharedtypes.GuildID, rID sharedtypes.RoundID, t time.Time, p roundevents.RoundStartedPayloadV1) error {
				// 	return errors.New("round start scheduling error")
				// }
			},
			want: results.OperationResult[*roundtypes.ScheduleRoundEventsResult, error]{
				Success: ptr(&roundtypes.ScheduleRoundEventsResult{
					RoundID:        testRoundID,
					GuildID:        testGuildID,
					Title:          testRoundTitle,
					Description:    testDescription,
					Location:       testLocation,
					StartTime:      *startTimePtr(now.Add(2 * time.Hour)),
					EventMessageID: testMessageID,
				}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queue := NewFakeQueueService()
			if tt.setup != nil {
				tt.setup(queue)
			}

			s := NewRoundService(nil, queue, nil, nil, mockMetrics, logger, tracer, nil, nil)

			req := &roundtypes.ScheduleRoundEventsRequest{
				RoundID:        testRoundID,
				GuildID:        testGuildID,
				Title:          testRoundTitle,
				Description:    testDescription,
				Location:       testLocation,
				StartTime:      *startTimePtr(now.Add(tt.startTimeOffset)),
				EventMessageID: testMessageID,
			}

			got, err := s.ScheduleRoundEvents(ctx, req)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundService.ScheduleRoundEvents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want.Failure != nil {
				if got.Failure == nil {
					t.Errorf("expected failure, got nil")
				} else if (*got.Failure).Error() != (*tt.want.Failure).Error() {
					t.Errorf("expected failure error %v, got %v", *tt.want.Failure, *got.Failure)
				}
			} else if got.Failure != nil {
				t.Errorf("expected no failure, got %v", *got.Failure)
			}

			if tt.want.Success != nil {
				if got.Success == nil {
					t.Errorf("expected success, got nil")
				} else {
					if diff := cmp.Diff(*got.Success, *tt.want.Success, cmpopts.EquateComparable(sharedtypes.StartTime{})); diff != "" {
						t.Errorf("RoundService.ScheduleRoundEvents() mismatch (-got +want):\n%s", diff)
					}
				}
			}
		})
	}
}
