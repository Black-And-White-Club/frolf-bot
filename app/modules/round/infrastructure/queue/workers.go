package roundqueue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/riverqueue/river"
)

// RoundStartWorker processes round start jobs by publishing round.started events
type RoundStartWorker struct {
	river.WorkerDefaults[RoundStartJob]
	logger   *slog.Logger
	eventBus eventbus.EventBus
	helpers  utils.Helpers
}

// NewRoundStartWorker creates a new round start worker
func NewRoundStartWorker(logger *slog.Logger, eventBus eventbus.EventBus, helpers utils.Helpers) *RoundStartWorker {
	return &RoundStartWorker{
		logger:   logger,
		eventBus: eventBus,
		helpers:  helpers,
	}
}

// Work processes a round start job by publishing the round.started event
func (w *RoundStartWorker) Work(ctx context.Context, job *river.Job[RoundStartJob]) error {
	ctxLogger := w.logger.With(
		attr.Int64("job_id", job.ID),
		attr.String("guild_id", string(job.Args.GuildID)),
		attr.String("round_id", job.Args.RoundID),
		attr.String("operation", "process_round_start_job"),
		attr.String("job_kind", job.Kind),
	)

	ctxLogger.Info("Processing round start job")

	// Create Watermill message using helpers (same as your handlers)
	msg, err := w.helpers.CreateNewMessage(job.Args.RoundData, roundevents.RoundStarted)
	if err != nil {
		ctxLogger.Error("Failed to create round started message", attr.Error(err))
		return fmt.Errorf("failed to create round started message: %w", err)
	}

	// Publish using your eventbus interface - it expects topic and messages
	if err := w.eventBus.Publish(roundevents.RoundStarted, msg); err != nil {
		ctxLogger.Error("Failed to publish round started event", attr.Error(err))
		return fmt.Errorf("failed to publish round started event: %w", err)
	}

	ctxLogger.Info("Round start job processed successfully - event published")
	return nil
}

// RoundReminderWorker processes round reminder jobs by publishing round.reminder events
type RoundReminderWorker struct {
	river.WorkerDefaults[RoundReminderJob]
	logger   *slog.Logger
	eventBus eventbus.EventBus
	helpers  utils.Helpers
}

// NewRoundReminderWorker creates a new round reminder worker
func NewRoundReminderWorker(logger *slog.Logger, eventBus eventbus.EventBus, helpers utils.Helpers) *RoundReminderWorker {
	return &RoundReminderWorker{
		logger:   logger,
		eventBus: eventBus,
		helpers:  helpers,
	}
}

// Work processes a round reminder job by publishing the round.reminder event
func (w *RoundReminderWorker) Work(ctx context.Context, job *river.Job[RoundReminderJob]) error {
	ctxLogger := w.logger.With(
		attr.Int64("job_id", job.ID),
		attr.String("guild_id", string(job.Args.GuildID)),
		attr.String("round_id", job.Args.RoundID),
		attr.String("operation", "process_round_reminder_job"),
		attr.String("job_kind", job.Kind),
	)

	ctxLogger.Info("Processing round reminder job")

	// Create Watermill message using helpers (same as your handlers)
	msg, err := w.helpers.CreateNewMessage(job.Args.RoundData, roundevents.RoundReminder)
	if err != nil {
		ctxLogger.Error("Failed to create round reminder message", attr.Error(err))
		return fmt.Errorf("failed to create round reminder message: %w", err)
	}

	// Publish using your eventbus interface - it expects topic and messages
	if err := w.eventBus.Publish(roundevents.RoundReminder, msg); err != nil {
		ctxLogger.Error("Failed to publish round reminder event", attr.Error(err))
		return fmt.Errorf("failed to publish round reminder event: %w", err)
	}

	ctxLogger.Info("Round reminder job processed successfully - event published")
	return nil
}
