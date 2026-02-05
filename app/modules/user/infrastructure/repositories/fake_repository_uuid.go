package userdb

import (
	"context"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// GetUserByUUID stub.
func (f *FakeRepository) GetUserByUUID(ctx context.Context, db bun.IDB, userUUID uuid.UUID) (*User, error) {
	if f.GetUserByUUIDFn != nil {
		return f.GetUserByUUIDFn(ctx, db, userUUID)
	}
	return nil, nil
}
