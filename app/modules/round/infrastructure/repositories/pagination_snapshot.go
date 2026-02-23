package rounddb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// RoundEmbedPaginationSnapshot stores serialized pagination state for Discord embeds.
type RoundEmbedPaginationSnapshot struct {
	bun.BaseModel `bun:"table:round_embed_pagination_snapshots,alias:reps"`

	MessageID    string          `bun:"message_id,pk,notnull"`
	SnapshotJSON json.RawMessage `bun:"snapshot_json,type:jsonb,notnull"`
	ExpiresAt    time.Time       `bun:"expires_at,notnull"`
	CreatedAt    time.Time       `bun:"created_at,notnull,default:now()"`
	UpdatedAt    time.Time       `bun:"updated_at,notnull,default:now()"`
}

// PaginationSnapshotStore defines persistence operations for embed pagination snapshots.
type PaginationSnapshotStore interface {
	Upsert(ctx context.Context, db bun.IDB, messageID string, snapshot json.RawMessage, expiresAt time.Time) error
	Get(ctx context.Context, db bun.IDB, messageID string) (json.RawMessage, bool, error)
	Delete(ctx context.Context, db bun.IDB, messageID string) error
}

// PaginationSnapshotRepository implements PaginationSnapshotStore using Bun.
type PaginationSnapshotRepository struct {
	db bun.IDB
}

// NewPaginationSnapshotRepository creates a new pagination snapshot repository.
func NewPaginationSnapshotRepository(db bun.IDB) PaginationSnapshotStore {
	return &PaginationSnapshotRepository{db: db}
}

func (r *PaginationSnapshotRepository) Upsert(ctx context.Context, db bun.IDB, messageID string, snapshot json.RawMessage, expiresAt time.Time) error {
	if messageID == "" {
		return errors.New("message id is empty")
	}
	if len(snapshot) == 0 {
		return errors.New("snapshot is empty")
	}
	if expiresAt.IsZero() {
		return errors.New("expires at is zero")
	}
	if db == nil {
		db = r.db
	}

	record := &RoundEmbedPaginationSnapshot{
		MessageID:    messageID,
		SnapshotJSON: append(json.RawMessage(nil), snapshot...),
		ExpiresAt:    expiresAt.UTC(),
	}

	_, err := db.NewInsert().
		Model(record).
		On("CONFLICT (message_id) DO UPDATE").
		Set("snapshot_json = EXCLUDED.snapshot_json").
		Set("expires_at = EXCLUDED.expires_at").
		Set("updated_at = now()").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("upsert pagination snapshot: %w", err)
	}

	return nil
}

func (r *PaginationSnapshotRepository) Get(ctx context.Context, db bun.IDB, messageID string) (json.RawMessage, bool, error) {
	if messageID == "" {
		return nil, false, nil
	}
	if db == nil {
		db = r.db
	}

	record := new(RoundEmbedPaginationSnapshot)
	err := db.NewSelect().
		Model(record).
		Where("message_id = ?", messageID).
		Where("expires_at > now()").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("get pagination snapshot: %w", err)
	}

	return append(json.RawMessage(nil), record.SnapshotJSON...), true, nil
}

func (r *PaginationSnapshotRepository) Delete(ctx context.Context, db bun.IDB, messageID string) error {
	if messageID == "" {
		return nil
	}
	if db == nil {
		db = r.db
	}

	_, err := db.NewDelete().
		Model((*RoundEmbedPaginationSnapshot)(nil)).
		Where("message_id = ?", messageID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete pagination snapshot: %w", err)
	}

	return nil
}
