package guildservice

import (
	"context"
	"log/slog"
	"time"

	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/uptrace/bun"
)

const (
	outboxPollInterval = 2 * time.Second
	outboxBatchSize    = 50
)

// OutboxForwarder polls the guild_outbox table and publishes pending events to
// the message bus. It uses SELECT … FOR UPDATE SKIP LOCKED so that multiple
// instances (e.g. after a rolling deploy) do not double-publish.
//
// Usage: call Run(ctx) in a background goroutine; it exits when ctx is cancelled.
type OutboxForwarder struct {
	db        *bun.DB
	repo      guilddb.Repository
	publisher outboxPublisher
	logger    *slog.Logger
}

// outboxPublisher is the minimal publish interface needed by the forwarder.
type outboxPublisher interface {
	Publish(topic string, messages ...*message.Message) error
}

// NewOutboxForwarder creates a new OutboxForwarder.
func NewOutboxForwarder(db *bun.DB, repo guilddb.Repository, publisher outboxPublisher, logger *slog.Logger) *OutboxForwarder {
	return &OutboxForwarder{
		db:        db,
		repo:      repo,
		publisher: publisher,
		logger:    logger,
	}
}

// Run polls for unpublished outbox events and forwards them until ctx is cancelled.
func (f *OutboxForwarder) Run(ctx context.Context) {
	ticker := time.NewTicker(outboxPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := f.forward(ctx); err != nil {
				f.logger.ErrorContext(ctx, "guild.outbox.forward failed", "err", err)
			}
		}
	}
}

func (f *OutboxForwarder) forward(ctx context.Context) error {
	return f.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		events, err := f.repo.PollAndLockOutboxEvents(ctx, tx, outboxBatchSize)
		if err != nil {
			return err
		}

		for _, evt := range events {
			msg := message.NewMessage(watermill.NewUUID(), evt.Payload)
			msg.SetContext(ctx)

			if err := f.publisher.Publish(evt.Topic, msg); err != nil {
				// Log and continue — the row stays unpublished and will be retried.
				f.logger.ErrorContext(ctx, "guild.outbox.publish failed",
					"id", evt.ID,
					"topic", evt.Topic,
					"err", err,
				)
				continue
			}

			if err := f.repo.MarkOutboxEventPublished(ctx, tx, evt.ID); err != nil {
				f.logger.ErrorContext(ctx, "guild.outbox.mark_published failed",
					"id", evt.ID,
					"err", err,
				)
			}
		}
		return nil
	})
}
