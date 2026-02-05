package userdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// GetUserByUUID retrieves a global user by their internal UUID.
func (r *Impl) GetUserByUUID(ctx context.Context, db bun.IDB, userUUID uuid.UUID) (*User, error) {
	if db == nil {
		db = r.db
	}
	user := &User{}
	err := db.NewSelect().
		Model(user).
		Where("uuid = ?", userUUID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("userdb.GetUserByUUID: %w", err)
	}
	return user, nil
}
