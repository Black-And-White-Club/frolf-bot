package clubdb

import (
	"context"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Repository defines the contract for club persistence.
type Repository interface {
	// GetByUUID retrieves a club by its UUID.
	GetByUUID(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*Club, error)

	// GetByDiscordGuildID retrieves a club by its Discord guild ID.
	GetByDiscordGuildID(ctx context.Context, db bun.IDB, guildID string) (*Club, error)

	// Upsert creates or updates a club.
	Upsert(ctx context.Context, db bun.IDB, club *Club) error

	// UpdateName updates a club's name.
	UpdateName(ctx context.Context, db bun.IDB, clubUUID uuid.UUID, name string) error
}
