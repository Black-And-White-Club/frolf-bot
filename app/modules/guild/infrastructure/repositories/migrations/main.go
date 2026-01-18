package migrations

import (
	"github.com/uptrace/bun/migrate"
)

var Migrations = migrate.NewMigrations()

func init() {
	// Option A: If you want to use the Discover functionality:
	// if err := Migrations.DiscoverCaller(); err != nil {
	//   panic(err)
	// }

	// Option B (Recommended): Explicitly register your migrations.
	// This is safer because you can control the order and logic.

	// 1. Initial Table Creation (Existing)
	// Migrations.MustRegister(CreateGuildConfigsTable, DropGuildConfigsTable)

	// 2. Add your new Refinement migration here!
	// Registering it here ensures it is part of the 'guild' migration group.
}
