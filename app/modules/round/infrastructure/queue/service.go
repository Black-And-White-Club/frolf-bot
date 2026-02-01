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
	ScheduleRoundStart(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, startTime time.Time, payload roundevents.RoundStartedPayloadV1) error
	ScheduleRoundReminder(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, reminderTime time.Time, payload roundevents.DiscordReminderPayloadV1) error
	CancelRoundJobs(ctx context.Context, roundID sharedtypes.RoundID) error
	GetScheduledJobs(ctx context.Context, roundID sharedtypes.RoundID) ([]JobInfo, error)
	HealthCheck(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Ensure Service implements QueueService
var _ QueueService = (*Service)(nil)

// Service handles job scheduling for the round module using River
type Service struct {
	client   *river.Client[pgx.Tx]
	pool     *pgxpool.Pool
	logger   *slog.Logger
	db       *bun.DB
	metrics  Metrics
	eventBus eventbus.EventBus
	helpers  utils.Helpers
}

// ServiceOptions allows customization of the queue service
type ServiceOptions struct {
	FetchPollInterval *time.Duration
}

// NewService creates a new River-based queue service for round scheduling
func NewService(ctx context.Context, bunDB *bun.DB, logger *slog.Logger, dsn string, metrics Metrics, eventBus eventbus.EventBus, helpers utils.Helpers) (*Service, error) {
	return NewServiceWithOptions(ctx, bunDB, logger, dsn, metrics, eventBus, helpers, nil)
}

// NewServiceWithOptions creates a new River-based queue service with custom options
func NewServiceWithOptions(ctx context.Context, bunDB *bun.DB, logger *slog.Logger, dsn string, metrics Metrics, eventBus eventbus.EventBus, helpers utils.Helpers, opts *ServiceOptions) (*Service, error) {
	ctxLogger := logger.With(
		attr.String("operation", "new_round_queue_service"),
		attr.String("component", "river_queue"),
	)

	start := time.Now()
	metrics.RecordOperationAttempt(ctx, "initialize_service", "river")

	ctxLogger.Info("Initializing Round queue service")

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

	err = pool.Ping(ctx)
	if err != nil {
		pool.Close()
		ctxLogger.Error("Failed to ping database for River", attr.Error(err))
		metrics.RecordOperationFailure(ctx, "initialize_service", "river")
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create River workers registry and register workers
	workers := river.NewWorkers()
	river.AddWorker(workers, NewRoundStartWorker(ctxLogger, eventBus, helpers))
	river.AddWorker(workers, NewRoundReminderWorker(ctxLogger, eventBus, helpers))

	riverConfig := &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 50},
			"round":            {MaxWorkers: 25},
		},
		Workers: workers,
	}

	if opts != nil && opts.FetchPollInterval != nil {
		riverConfig.FetchPollInterval = *opts.FetchPollInterval
	}

	riverClient, err := river.NewClient(riverpgxv5.New(pool), riverConfig)
	if err != nil {
		pool.Close()
		ctxLogger.Error("Failed to create River client", attr.Error(err))
		metrics.RecordOperationFailure(ctx, "initialize_service", "river")
		return nil, fmt.Errorf("failed to create River client: %w", err)
	}

	service := &Service{
		client:   riverClient,
		pool:     pool,
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

func (s *Service) Start(ctx context.Context) error {
	s.logger.Info("Starting Round queue service")
	return s.client.Start(ctx)
}

func (s *Service) Stop(ctx context.Context) error {
	s.logger.Info("Stopping Round queue service")
	if err := s.client.Stop(ctx); err != nil {
		s.pool.Close()
		return err
	}
	s.pool.Close()
	return nil
}

func (s *Service) ScheduleRoundStart(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, startTime time.Time, payload roundevents.RoundStartedPayloadV1) error {
	if payload.GuildID == "" {
		payload.GuildID = guildID
	}

	job := RoundStartJob{
		GuildID: guildID,
		RoundID: roundID,
	}

	_, err := s.client.Insert(ctx, job, &river.InsertOpts{
		Queue:       "round",
		ScheduledAt: startTime,
		UniqueOpts: river.UniqueOpts{
			ByArgs: true,
		},
	})
	return err
}

func (s *Service) ScheduleRoundReminder(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, reminderTime time.Time, payload roundevents.DiscordReminderPayloadV1) error {
	if payload.GuildID == "" {
		payload.GuildID = guildID
	}

	job := RoundReminderJob{
		GuildID:   guildID,
		RoundID:   roundID,
		RoundData: payload,
	}

	_, err := s.client.Insert(ctx, job, &river.InsertOpts{
		Queue:       "round",
		ScheduledAt: reminderTime,
		UniqueOpts: river.UniqueOpts{
			ByArgs: true,
		},
	})
	return err
}

func (s *Service) CancelRoundJobs(ctx context.Context, roundID sharedtypes.RoundID) error {
	type RiverJobRow struct {
		ID int64 `bun:"id"`
	}
	var jobs []RiverJobRow
	err := s.db.NewSelect().
		Table("river_job").
		Column("id").
		Where("args->>'round_id' = ?", roundID.String()).
		Where("state IN ('available', 'scheduled')").
		Scan(ctx, &jobs)

	if err != nil {
		return err
	}

	for _, job := range jobs {
		if _, err := s.client.JobCancel(ctx, job.ID); err != nil {
			s.logger.Warn("Failed to cancel job", attr.Int64("job_id", job.ID), attr.Error(err))
		}
	}
	return nil
}

func (s *Service) GetScheduledJobs(ctx context.Context, roundID sharedtypes.RoundID) ([]JobInfo, error) {
	var jobs []struct {
		ID          int64      `bun:"id"`
		Kind        string     `bun:"kind"`
		State       string     `bun:"state"`
		ScheduledAt *time.Time `bun:"scheduled_at"`
		CreatedAt   time.Time  `bun:"created_at"`
		Attempt     int16      `bun:"attempt"`
		MaxAttempts int16      `bun:"max_attempts"`
	}

	err := s.db.NewSelect().
		Table("river_job").
		Where("args->>'round_id' = ?", roundID.String()).
		Order("scheduled_at ASC NULLS LAST", "created_at ASC").
		Scan(ctx, &jobs)

	if err != nil {
		return nil, err
	}

	res := make([]JobInfo, len(jobs))
	for i, j := range jobs {
		sched := ""
		if j.ScheduledAt != nil {
			sched = j.ScheduledAt.Format(time.RFC3339)
		}
		res[i] = JobInfo{
			ID:          j.ID,
			Kind:        j.Kind,
			RoundID:     roundID.String(),
			State:       j.State,
			ScheduledAt: sched,
			CreatedAt:   j.CreatedAt.Format(time.RFC3339),
			Attempt:     int(j.Attempt),
			MaxAttempts: int(j.MaxAttempts),
		}
	}
	return res, nil
}

func (s *Service) HealthCheck(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
