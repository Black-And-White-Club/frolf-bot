package usermigrations

import "github.com/uptrace/bun/migrate"

var Migrations = migrate.NewMigrations()

func init() {
	// Enable automatic discovery of the caller file name for migrations so that
	// each registered migration gets a stable ID derived from its file name.
	// This is required when using MustRegister in separate files without
	// providing explicit IDs.
	if err := Migrations.DiscoverCaller(); err != nil {
		panic(err)
	}
}
