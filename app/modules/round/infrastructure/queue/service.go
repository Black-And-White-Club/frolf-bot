package roundqueue

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/uptrace/bun"
)

// Metrics interface (using your existing round metrics)
type Metrics interface {
	RecordOperationAttempt(ctx context.Context, operation, service string)
	RecordOperationSuccess(ctx context.Context, operation, service string)
	RecordOperationFailure(ctx context.Context, operation, service string)
	RecordOperationDuration(ctx context.Context, operation, service string, duration time.Duration)
}

// QueueService interface defines the contract for job scheduling operations
type QueueService interface {
	// ScheduleRoundStart schedules a round start job to be executed at the specified time
	ScheduleRoundStart(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, startTime time.Time, payload roundevents.RoundStartedPayload) error
	// ScheduleRoundReminder schedules a round reminder job to be executed at the specified time
	ScheduleRoundReminder(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, reminderTime time.Time, payload roundevents.DiscordReminderPayload) error
	// CancelRoundJobs cancels all scheduled jobs for a specific round
	CancelRoundJobs(ctx context.Context, roundID sharedtypes.RoundID) error
	// GetScheduledJobs returns information about scheduled jobs for a round (for debugging)
	GetScheduledJobs(ctx context.Context, roundID sharedtypes.RoundID) ([]JobInfo, error)
	// HealthCheck verifies the queue service is healthy
	HealthCheck(ctx context.Context) error
	// Start starts the queue service
	Start(ctx context.Context) error
	// Stop stops the queue service
	Stop(ctx context.Context) error
}

// Ensure Service implements QueueService
var _ QueueService = (*Service)(nil)

// Service handles job scheduling for the round module using River
type Service struct {
	client   *river.Client[pgx.Tx]
	logger   *slog.Logger
	db       *bun.DB
	metrics  Metrics
	eventBus eventbus.EventBus
	helpers  utils.Helpers
}

// NewService creates a new River-based queue service for round scheduling
func NewService(ctx context.Context, bunDB *bun.DB, logger *slog.Logger, dsn string, metrics Metrics, eventBus eventbus.EventBus, helpers utils.Helpers) (*Service, error) {
	ctxLogger := logger.With(
		attr.String("operation", "new_round_queue_service"),
		attr.String("component", "river_queue"),
	)

	start := time.Now()
	metrics.RecordOperationAttempt(ctx, "initialize_service", "river")

	ctxLogger.Info("Initializing Round queue service")

	// Create pgx pool for River (River requires pgx, not database/sql)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		ctxLogger.Error("Failed to parse DSN for River", attr.Error(err))
		metrics.RecordOperationFailure(ctx, "initialize_service", "river")
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		ctxLogger.Error("Failed to create pgx pool for River", attr.Error(err))
		metrics.RecordOperationFailure(ctx, "initialize_service", "river")
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		ctxLogger.Error("Failed to ping database for River", attr.Error(err))
		metrics.RecordOperationFailure(ctx, "initialize_service", "river")
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create River workers registry and register workers
	workers := river.NewWorkers()

	// Register workers with event bus and helpers
	river.AddWorker(workers, NewRoundStartWorker(ctxLogger, eventBus, helpers))
	river.AddWorker(workers, NewRoundReminderWorker(ctxLogger, eventBus, helpers))

	// Create River client with configuration
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 50},
			"round":            {MaxWorkers: 25}, // Dedicated queue for round jobs
		},
		Workers: workers,
	})
	if err != nil {
		pool.Close()
		ctxLogger.Error("Failed to create River client", attr.Error(err))
		metrics.RecordOperationFailure(ctx, "initialize_service", "river")
		return nil, fmt.Errorf("failed to create River client: %w", err)
	}

	service := &Service{
		client:   riverClient,
		logger:   ctxLogger,
		db:       bunDB,
		metrics:  metrics,
		eventBus: eventBus,
		helpers:  helpers,
	}

	duration := time.Since(start)
	metrics.RecordOperationSuccess(ctx, "initialize_service", "river")
	metrics.RecordOperationDuration(ctx, "initialize_service", "river", duration)

	ctxLogger.Info("Round queue service initialized successfully")
	return service, nil
}

// Start starts the River queue service
func (s *Service) Start(ctx context.Context) error {
	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "start_service", "river")

	s.logger.Info("Starting Round queue service")

	if err := s.client.Start(ctx); err != nil {
		s.logger.Error("Failed to start River client", attr.Error(err))
		s.metrics.RecordOperationFailure(ctx, "start_service", "river")
		return fmt.Errorf("failed to start River client: %w", err)
	}

	duration := time.Since(start)
	s.metrics.RecordOperationSuccess(ctx, "start_service", "river")
	s.metrics.RecordOperationDuration(ctx, "start_service", "river", duration)

	s.logger.Info("Round queue service started successfully")
	return nil
}

// Stop stops the River queue service
func (s *Service) Stop(ctx context.Context) error {
	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "stop_service", "river")

	s.logger.Info("Stopping Round queue service")

	if err := s.client.Stop(ctx); err != nil {
		s.logger.Error("Failed to stop River client", attr.Error(err))
		s.metrics.RecordOperationFailure(ctx, "stop_service", "river")
		return fmt.Errorf("failed to stop River client: %w", err)
	}

	duration := time.Since(start)
	s.metrics.RecordOperationSuccess(ctx, "stop_service", "river")
	s.metrics.RecordOperationDuration(ctx, "stop_service", "river", duration)

	s.logger.Info("Round queue service stopped successfully")
	return nil
}

// ScheduleRoundStart schedules a round start job to be executed at the specified time
func (s *Service) ScheduleRoundStart(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, startTime time.Time, payload roundevents.RoundStartedPayload) error {
	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "schedule_round_start", "river")

	ctxLogger := s.logger.With(
		attr.RoundID("round_id", roundID),
		attr.Time("start_time", startTime),
		attr.String("operation", "schedule_round_start"),
	)

	ctxLogger.Info("Scheduling round start job")

	// Validate timing - ensure we have at least a small buffer
	now := time.Now()
	if startTime.Before(now.Add(5 * time.Second)) {
		ctxLogger.Warn("Round start time is too close to current time",
			attr.Time("current_time", now),
			attr.Duration("buffer", startTime.Sub(now)))
		s.metrics.RecordOperationFailure(ctx, "schedule_round_start", "river")
		return fmt.Errorf("start time must be at least 5 seconds in the future")
	}

	// Create the job
	job := RoundStartJob{
		GuildID:   guildID,
		RoundID:   roundID.String(),
		RoundData: payload,
	}

	// Schedule the job
	jobResult, err := s.client.Insert(ctx, job, &river.InsertOpts{
		Queue:       "round",
		ScheduledAt: startTime,
		UniqueOpts: river.UniqueOpts{
			ByArgs: true, // Prevent duplicate scheduling for same round
		},
	})
	if err != nil {
		ctxLogger.Error("Failed to schedule round start job", attr.Error(err))
		s.metrics.RecordOperationFailure(ctx, "schedule_round_start", "river")
		return fmt.Errorf("failed to schedule round start job: %w", err)
	}

	duration := time.Since(start)
	s.metrics.RecordOperationSuccess(ctx, "schedule_round_start", "river")
	s.metrics.RecordOperationDuration(ctx, "schedule_round_start", "river", duration)

	ctxLogger.Info("Round start job scheduled successfully",
		attr.Duration("delay", startTime.Sub(now)),
		attr.Int64("job_id", jobResult.Job.ID))
	return nil
}

// ScheduleRoundReminder schedules a round reminder job to be executed at the specified time
func (s *Service) ScheduleRoundReminder(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, reminderTime time.Time, payload roundevents.DiscordReminderPayload) error {
	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "schedule_round_reminder", "river")

	ctxLogger := s.logger.With(
		attr.RoundID("round_id", roundID),
		attr.Time("reminder_time", reminderTime),
		attr.String("operation", "schedule_round_reminder"),
	)

	ctxLogger.Info("Scheduling round reminder job")

	// Validate timing - if reminder is in the past, just skip it
	now := time.Now()
	if reminderTime.Before(now.Add(5 * time.Second)) {
		ctxLogger.Info("Reminder time is in the past or too close, skipping",
			attr.Time("current_time", now),
			attr.Duration("difference", reminderTime.Sub(now)))
		// This is not a failure, just a skip
		s.metrics.RecordOperationSuccess(ctx, "schedule_round_reminder", "river")
		return nil
	}

	// Create the job
	job := RoundReminderJob{
		GuildID:   guildID,
		RoundID:   roundID.String(),
		RoundData: payload,
	}

	// Schedule the job
	jobResult, err := s.client.Insert(ctx, job, &river.InsertOpts{
		Queue:       "round",
		ScheduledAt: reminderTime,
		UniqueOpts: river.UniqueOpts{
			ByArgs: true, // Prevent duplicate scheduling for same round
		},
	})
	if err != nil {
		ctxLogger.Error("Failed to schedule round reminder job", attr.Error(err))
		s.metrics.RecordOperationFailure(ctx, "schedule_round_reminder", "river")
		return fmt.Errorf("failed to schedule round reminder job: %w", err)
	}

	duration := time.Since(start)
	s.metrics.RecordOperationSuccess(ctx, "schedule_round_reminder", "river")
	s.metrics.RecordOperationDuration(ctx, "schedule_round_reminder", "river", duration)

	ctxLogger.Info("Round reminder job scheduled successfully",
		attr.Duration("delay", reminderTime.Sub(now)),
		attr.Int64("job_id", jobResult.Job.ID))
	return nil
}

// CancelRoundJobs cancels all scheduled jobs for a specific round
func (s *Service) CancelRoundJobs(ctx context.Context, roundID sharedtypes.RoundID) error {
	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "cancel_round_jobs", "river")

	ctxLogger := s.logger.With(
		attr.RoundID("round_id", roundID),
		attr.String("operation", "cancel_round_jobs"),
	)

	ctxLogger.Info("Cancelling scheduled jobs for round")

	// Query for jobs with this round ID in their args
	type RiverJobRow struct {
		ID          int64                  `bun:"id"`
		Kind        string                 `bun:"kind"`
		State       string                 `bun:"state"`
		Args        map[string]interface{} `bun:"args"`
		ScheduledAt *time.Time             `bun:"scheduled_at"`
	}

	var jobs []RiverJobRow
	err := s.db.NewSelect().
		Table("river_job").
		Column("id", "kind", "state", "args", "scheduled_at").
		Where("kind IN (?, ?)", "round_start", "round_reminder").
		Where("state IN (?, ?)", "available", "scheduled").
		Where("args->>'round_id' = ?", roundID.String()).
		Scan(ctx, &jobs)
	if err != nil {
		ctxLogger.Error("Failed to query jobs for cancellation", attr.Error(err))
		s.metrics.RecordOperationFailure(ctx, "cancel_round_jobs", "river")
		return fmt.Errorf("failed to query jobs for cancellation: %w", err)
	}

	if len(jobs) == 0 {
		ctxLogger.Info("No jobs found to cancel")
		duration := time.Since(start)
		s.metrics.RecordOperationSuccess(ctx, "cancel_round_jobs", "river")
		s.metrics.RecordOperationDuration(ctx, "cancel_round_jobs", "river", duration)
		return nil
	}

	// Cancel each job
	cancelledCount := 0
	for _, job := range jobs {
		_, err := s.client.JobCancel(ctx, job.ID)
		if err != nil {
			ctxLogger.Warn("Failed to cancel job",
				attr.Int64("job_id", job.ID),
				attr.String("job_kind", job.Kind),
				attr.Error(err))
			continue
		}
		cancelledCount++
	}

	duration := time.Since(start)
	if cancelledCount == len(jobs) {
		s.metrics.RecordOperationSuccess(ctx, "cancel_round_jobs", "river")
	} else {
		s.metrics.RecordOperationFailure(ctx, "cancel_round_jobs", "river")
	}
	s.metrics.RecordOperationDuration(ctx, "cancel_round_jobs", "river", duration)

	ctxLogger.Info("Jobs cancellation completed",
		attr.Int("total_found", len(jobs)),
		attr.Int("cancelled_count", cancelledCount))

	return nil
}

// GetScheduledJobs returns information about scheduled jobs for a round (for debugging)
func (s *Service) GetScheduledJobs(ctx context.Context, roundID sharedtypes.RoundID) ([]JobInfo, error) {
	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "get_scheduled_jobs", "river")

	ctxLogger := s.logger.With(
		attr.RoundID("round_id", roundID),
		attr.String("operation", "get_scheduled_jobs"),
	)

	ctxLogger.Info("Getting scheduled jobs for round")

	type RiverJobRow struct {
		ID          int64                  `bun:"id"`
		Kind        string                 `bun:"kind"`
		State       string                 `bun:"state"`
		Args        map[string]interface{} `bun:"args"`
		ScheduledAt *time.Time             `bun:"scheduled_at"`
		CreatedAt   time.Time              `bun:"created_at"`
		Attempt     int16                  `bun:"attempt"`
		MaxAttempts int16                  `bun:"max_attempts"`
	}

	var jobs []RiverJobRow
	err := s.db.NewSelect().
		Table("river_job").
		Column("id", "kind", "state", "args", "scheduled_at", "created_at", "attempt", "max_attempts").
		Where("kind IN (?, ?)", "round_start", "round_reminder").
		Where("args->>'round_id' = ?", roundID.String()).
		Order("scheduled_at ASC NULLS LAST", "created_at ASC").
		Scan(ctx, &jobs)
	if err != nil {
		ctxLogger.Error("Failed to query scheduled jobs", attr.Error(err))
		s.metrics.RecordOperationFailure(ctx, "get_scheduled_jobs", "river")
		return nil, fmt.Errorf("failed to query scheduled jobs: %w", err)
	}

	// Convert to JobInfo
	result := make([]JobInfo, len(jobs))
	for i, job := range jobs {
		scheduledAt := ""
		if job.ScheduledAt != nil {
			scheduledAt = job.ScheduledAt.Format(time.RFC3339)
		}

		result[i] = JobInfo{
			ID:          job.ID,
			Kind:        job.Kind,
			RoundID:     roundID.String(),
			State:       job.State,
			ScheduledAt: scheduledAt,
			CreatedAt:   job.CreatedAt.Format(time.RFC3339),
			Attempt:     int(job.Attempt),
			MaxAttempts: int(job.MaxAttempts),
		}
	}

	duration := time.Since(start)
	s.metrics.RecordOperationSuccess(ctx, "get_scheduled_jobs", "river")
	s.metrics.RecordOperationDuration(ctx, "get_scheduled_jobs", "river", duration)

	ctxLogger.Info("Retrieved scheduled jobs",
		attr.Int("job_count", len(result)))

	return result, nil
}

// HealthCheck verifies the queue service is healthy
func (s *Service) HealthCheck(ctx context.Context) error {
	start := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "health_check", "river")

	if s.client == nil {
		s.metrics.RecordOperationFailure(ctx, "health_check", "river")
		return fmt.Errorf("river client is nil")
	}

	// Try a simple database query to verify connectivity
	var count int
	err := s.db.NewSelect().
		Table("river_job").
		ColumnExpr("COUNT(*)").
		Scan(ctx, &count)
	if err != nil {
		s.logger.Error("Queue service health check failed", attr.Error(err))
		s.metrics.RecordOperationFailure(ctx, "health_check", "river")
		return fmt.Errorf("queue service health check failed: %w", err)
	}

	duration := time.Since(start)
	s.metrics.RecordOperationSuccess(ctx, "health_check", "river")
	s.metrics.RecordOperationDuration(ctx, "health_check", "river", duration)

	s.logger.Debug("Queue service health check passed", attr.Int("total_jobs", count))
	return nil
}

// GetClient returns the underlying River client for advanced operations
func (s *Service) GetClient() *river.Client[pgx.Tx] {
	return s.client
}
