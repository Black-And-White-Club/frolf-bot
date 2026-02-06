package clubdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ErrNotFound is returned when a club is not found.
var ErrNotFound = errors.New("club not found")

// Impl implements the Repository interface using Bun ORM.
type Impl struct {
	db bun.IDB
}

// NewRepository creates a new club repository.
func NewRepository(db bun.IDB) Repository {
	return &Impl{db: db}
}

// resolveDB returns the provided db handle, falling back to the repository's
// default connection if db is nil.
func (r *Impl) resolveDB(db bun.IDB) bun.IDB {
	if db == nil {
		return r.db
	}
	return db
}

// GetByUUID retrieves a club by its UUID.
func (r *Impl) GetByUUID(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*Club, error) {
	db = r.resolveDB(db)
	club := new(Club)
	err := db.NewSelect().
		Model(club).
		Where("uuid = ?", clubUUID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get club by UUID: %w", err)
	}
	return club, nil
}

// GetByDiscordGuildID retrieves a club by its Discord guild ID.
func (r *Impl) GetByDiscordGuildID(ctx context.Context, db bun.IDB, guildID string) (*Club, error) {
	db = r.resolveDB(db)
	club := new(Club)
	err := db.NewSelect().
		Model(club).
		Where("discord_guild_id = ?", guildID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get club by Discord guild ID: %w", err)
	}
	return club, nil
}

// Upsert creates or updates a club.
func (r *Impl) Upsert(ctx context.Context, db bun.IDB, club *Club) error {
	db = r.resolveDB(db)
	club.UpdatedAt = time.Now()
	_, err := db.NewInsert().
		Model(club).
		On("CONFLICT (uuid) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("icon_url = EXCLUDED.icon_url").
		Set("discord_guild_id = EXCLUDED.discord_guild_id").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to upsert club: %w", err)
	}
	return nil
}

// UpdateName updates a club's name.
func (r *Impl) UpdateName(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, name string) error {
	db = r.resolveDB(db)
	result, err := db.NewUpdate().
		Model((*Club)(nil)).
		Set("name = ?", name).
		Set("updated_at = ?", time.Now()).
		Where("uuid = ?", clubUUID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update club name: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
