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

func NewRoundStartWorker(logger *slog.Logger, eventBus eventbus.EventBus, helpers utils.Helpers) *RoundStartWorker {
	return &RoundStartWorker{
		logger:   logger,
		eventBus: eventBus,
		helpers:  helpers,
	}
}

func (w *RoundStartWorker) Work(ctx context.Context, job *river.Job[RoundStartJob]) error {
	ctxLogger := w.logger.With(
		attr.Int64("job_id", job.ID),
		attr.String("guild_id", string(job.Args.GuildID)),
		attr.String("round_id", job.Args.RoundID.String()),
		attr.String("operation", "process_round_start_job"),
	)

	ctxLogger.Info("Processing round start job")

	// Build a minimal backend request payload. The worker must not attempt to
	// materialize the full domain payload; DB is the source of truth.
	payload := roundevents.RoundStartRequestedPayloadV1{
		GuildID: job.Args.GuildID,
		RoundID: job.Args.RoundID,
	}

	msg, err := w.helpers.CreateNewMessage(payload, roundevents.RoundStartRequestedV1)
	if err != nil {
		ctxLogger.Error("Failed to create round start requested message", attr.Error(err))
		return fmt.Errorf("failed to create round start requested message: %w", err)
	}

	// Attach guild_id metadata to help routing/middleware
	if msg.Metadata.Get("guild_id") == "" && job.Args.GuildID != "" {
		msg.Metadata.Set("guild_id", string(job.Args.GuildID))
	}

	if err := w.eventBus.Publish(roundevents.RoundStartRequestedV1, msg); err != nil {
		ctxLogger.Error("Failed to publish round start requested event", attr.Error(err))
		return fmt.Errorf("failed to publish round start requested event: %w", err)
	}

	ctxLogger.Info("Round start job processed successfully - requested event published")
	return nil
}

// RoundReminderWorker processes round reminder jobs
type RoundReminderWorker struct {
	river.WorkerDefaults[RoundReminderJob]
	logger   *slog.Logger
	eventBus eventbus.EventBus
	helpers  utils.Helpers
}

func NewRoundReminderWorker(logger *slog.Logger, eventBus eventbus.EventBus, helpers utils.Helpers) *RoundReminderWorker {
	return &RoundReminderWorker{
		logger:   logger,
		eventBus: eventBus,
		helpers:  helpers,
	}
}

func (w *RoundReminderWorker) Work(ctx context.Context, job *river.Job[RoundReminderJob]) error {
	ctxLogger := w.logger.With(
		attr.Int64("job_id", job.ID),
		attr.String("guild_id", string(job.Args.GuildID)),
		attr.String("round_id", job.Args.RoundID.String()),
		attr.String("operation", "process_round_reminder_job"),
	)

	ctxLogger.Info("Processing round reminder job")

	// SAFEGUARD: Defensive enrichment
	if job.Args.RoundData.GuildID == "" && job.Args.GuildID != "" {
		job.Args.RoundData.GuildID = job.Args.GuildID
	}
	if job.Args.RoundData.DiscordGuildID == "" && job.Args.GuildID != "" {
		job.Args.RoundData.DiscordGuildID = string(job.Args.GuildID)
	}

	// Create Watermill message using helpers
	msg, err := w.helpers.CreateNewMessage(job.Args.RoundData, roundevents.RoundReminderScheduledV1)
	if err != nil {
		ctxLogger.Error("Failed to create round reminder message", attr.Error(err))
		return fmt.Errorf("failed to create round reminder message: %w", err)
	}

	// Ensure guild_id metadata is present
	if msg.Metadata.Get("guild_id") == "" && job.Args.GuildID != "" {
		msg.Metadata.Set("guild_id", string(job.Args.GuildID))
	}

	// Publish to the scheduled topic for the Router to handle
	if err := w.eventBus.Publish(roundevents.RoundReminderScheduledV1, msg); err != nil {
		ctxLogger.Error("Failed to publish round reminder event", attr.Error(err))
		return fmt.Errorf("failed to publish round reminder event: %w", err)
	}

	ctxLogger.Info("Round reminder job processed successfully - event published")
	return nil
}
