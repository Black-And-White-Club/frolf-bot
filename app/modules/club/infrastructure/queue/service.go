package clubqueue

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	clubmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/club"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/uptrace/bun"
)

// QueueService defines the club challenge scheduling operations.
type QueueService interface {
	ScheduleOpenExpiry(ctx context.Context, challengeID uuid.UUID, expiresAt time.Time) error
	ScheduleAcceptedExpiry(ctx context.Context, challengeID uuid.UUID, expiresAt time.Time) error
	CancelChallengeJobs(ctx context.Context, challengeID uuid.UUID) error
	HealthCheck(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type Service struct {
	client  *river.Client[pgx.Tx]
	pool    *pgxpool.Pool
	db      *bun.DB
	logger  *slog.Logger
	metrics clubmetrics.ClubMetrics
}

func NewService(ctx context.Context, bunDB *bun.DB, logger *slog.Logger, dsn string, metrics clubmetrics.ClubMetrics, eventBus eventbus.EventBus, helpers utils.Helpers) (*Service, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping challenge queue db: %w", err)
	}

	workers := river.NewWorkers()
	river.AddWorker(workers, NewOpenChallengeExpiryWorker(logger, eventBus, helpers))
	river.AddWorker(workers, NewAcceptedChallengeExpiryWorker(logger, eventBus, helpers))

	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
			"club":             {MaxWorkers: 10},
		},
		Workers: workers,
	})
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("create river client: %w", err)
	}

	return &Service{
		client:  client,
		pool:    pool,
		db:      bunDB,
		logger:  logger.With(attr.String("component", "club_queue")),
		metrics: metrics,
	}, nil
}

func (s *Service) Start(ctx context.Context) error {
	s.logger.Info("starting club queue service")
	return s.runOperation(ctx, "start", func() error {
		return s.client.Start(ctx)
	})
}

func (s *Service) Stop(ctx context.Context) error {
	s.logger.Info("stopping club queue service")
	err := s.runOperation(ctx, "stop", func() error {
		return s.client.Stop(ctx)
	})
	s.pool.Close()
	return err
}

func (s *Service) ScheduleOpenExpiry(ctx context.Context, challengeID uuid.UUID, expiresAt time.Time) error {
	return s.runOperation(ctx, "schedule_open_expiry", func() error {
		_, err := s.client.Insert(ctx, OpenChallengeExpiryJob{ChallengeID: challengeID}, &river.InsertOpts{
			Queue:       "club",
			ScheduledAt: expiresAt,
			UniqueOpts:  river.UniqueOpts{ByArgs: true},
		})
		return err
	})
}

func (s *Service) ScheduleAcceptedExpiry(ctx context.Context, challengeID uuid.UUID, expiresAt time.Time) error {
	return s.runOperation(ctx, "schedule_accepted_expiry", func() error {
		_, err := s.client.Insert(ctx, AcceptedChallengeExpiryJob{ChallengeID: challengeID}, &river.InsertOpts{
			Queue:       "club",
			ScheduledAt: expiresAt,
			UniqueOpts:  river.UniqueOpts{ByArgs: true},
		})
		return err
	})
}

func (s *Service) CancelChallengeJobs(ctx context.Context, challengeID uuid.UUID) error {
	return s.runOperation(ctx, "cancel_challenge_jobs", func() error {
		// river_job is River's internal table (https://riverqueue.com/docs/job-cancellation).
		// River does not expose a bulk-cancel-by-metadata API, so we query the table directly
		// using the JSON args field. If River changes its schema this query must be updated.
		type RiverJobRow struct {
			ID int64 `bun:"id"`
		}

		var jobs []RiverJobRow
		err := s.db.NewSelect().
			Table("river_job").
			Column("id").
			Where("args->>'challenge_id' = ?", challengeID.String()).
			Where("state IN ('available', 'scheduled')").
			Scan(ctx, &jobs)
		if err != nil {
			return err
		}

		for _, job := range jobs {
			if _, err := s.client.JobCancel(ctx, job.ID); err != nil {
				s.logger.Warn("failed to cancel challenge job", attr.Int64("job_id", job.ID), attr.Error(err))
			}
		}
		return nil
	})
}

func (s *Service) HealthCheck(ctx context.Context) error {
	return s.runOperation(ctx, "health_check", func() error {
		return s.pool.Ping(ctx)
	})
}

func (s *Service) runOperation(ctx context.Context, operation string, op func() error) error {
	if s.metrics != nil {
		s.metrics.RecordOperationAttempt(ctx, operation, "ClubQueue")
	}

	start := time.Now()
	defer func() {
		if s.metrics != nil {
			s.metrics.RecordOperationDuration(ctx, operation, "ClubQueue", time.Since(start))
		}
	}()

	if err := op(); err != nil {
		if s.metrics != nil {
			s.metrics.RecordOperationFailure(ctx, operation, "ClubQueue")
		}
		return err
	}

	if s.metrics != nil {
		s.metrics.RecordOperationSuccess(ctx, operation, "ClubQueue")
	}
	return nil
}
