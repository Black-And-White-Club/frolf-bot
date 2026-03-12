package clubqueue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	clubevents "github.com/Black-And-White-Club/frolf-bot-shared/events/club"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/riverqueue/river"
)

type OpenChallengeExpiryWorker struct {
	river.WorkerDefaults[OpenChallengeExpiryJob]
	logger   *slog.Logger
	eventBus eventbus.EventBus
	helpers  utils.Helpers
}

func NewOpenChallengeExpiryWorker(logger *slog.Logger, eventBus eventbus.EventBus, helpers utils.Helpers) *OpenChallengeExpiryWorker {
	return &OpenChallengeExpiryWorker{logger: logger, eventBus: eventBus, helpers: helpers}
}

func (w *OpenChallengeExpiryWorker) Work(ctx context.Context, job *river.Job[OpenChallengeExpiryJob]) error {
	ctxLogger := w.logger.With(attr.Int64("job_id", job.ID), attr.String("challenge_id", job.Args.ChallengeID.String()))
	payload := clubevents.ChallengeExpireRequestedPayloadV1{
		ChallengeID: job.Args.ChallengeID.String(),
		Reason:      "open_expired",
	}
	msg, err := w.helpers.CreateNewMessage(payload, clubevents.ChallengeExpireRequestedV1)
	if err != nil {
		ctxLogger.Error("failed to create open challenge expiry message", attr.Error(err))
		return fmt.Errorf("create open challenge expiry message: %w", err)
	}
	if err := w.eventBus.Publish(clubevents.ChallengeExpireRequestedV1, msg); err != nil {
		ctxLogger.Error("failed to publish open challenge expiry", attr.Error(err))
		return fmt.Errorf("publish open challenge expiry: %w", err)
	}
	return nil
}

type AcceptedChallengeExpiryWorker struct {
	river.WorkerDefaults[AcceptedChallengeExpiryJob]
	logger   *slog.Logger
	eventBus eventbus.EventBus
	helpers  utils.Helpers
}

func NewAcceptedChallengeExpiryWorker(logger *slog.Logger, eventBus eventbus.EventBus, helpers utils.Helpers) *AcceptedChallengeExpiryWorker {
	return &AcceptedChallengeExpiryWorker{logger: logger, eventBus: eventBus, helpers: helpers}
}

func (w *AcceptedChallengeExpiryWorker) Work(ctx context.Context, job *river.Job[AcceptedChallengeExpiryJob]) error {
	ctxLogger := w.logger.With(attr.Int64("job_id", job.ID), attr.String("challenge_id", job.Args.ChallengeID.String()))
	payload := clubevents.ChallengeExpireRequestedPayloadV1{
		ChallengeID: job.Args.ChallengeID.String(),
		Reason:      "accepted_expired",
	}
	msg, err := w.helpers.CreateNewMessage(payload, clubevents.ChallengeExpireRequestedV1)
	if err != nil {
		ctxLogger.Error("failed to create accepted challenge expiry message", attr.Error(err))
		return fmt.Errorf("create accepted challenge expiry message: %w", err)
	}
	if err := w.eventBus.Publish(clubevents.ChallengeExpireRequestedV1, msg); err != nil {
		ctxLogger.Error("failed to publish accepted challenge expiry", attr.Error(err))
		return fmt.Errorf("publish accepted challenge expiry: %w", err)
	}
	return nil
}
