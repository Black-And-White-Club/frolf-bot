package leaderboarddb

import (
	"github.com/uptrace/bun"
)

// Impl implements the Repository interface using Bun ORM.
type Impl struct {
	db bun.IDB
}

// NewRepository creates a new leaderboard repository.
func NewRepository(db bun.IDB) Repository {
	return &Impl{db: db}
}
