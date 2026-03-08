package testutils

import (
	"context"
	"strings"
	"testing"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

func runSingleModuleMigration(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	migrations *migrate.Migrations,
	migrationNameContains string,
) {
	t.Helper()

	migrator := migrate.NewMigrator(db, migrations, migrate.WithUpsert(true))
	if err := migrator.Init(ctx); err != nil {
		t.Fatalf("failed to init module migrator: %v", err)
	}

	allMigrations, err := migrator.MigrationsWithStatus(ctx)
	if err != nil {
		t.Fatalf("failed to list module migrations: %v", err)
	}

	var targetName string
	for _, migration := range allMigrations {
		if strings.Contains(migration.Name, migrationNameContains) || strings.Contains(migration.Comment, migrationNameContains) {
			targetName = migration.Name
			break
		}
	}
	if targetName == "" {
		t.Fatalf("migration containing %q not found", migrationNameContains)
	}

	if err := migrator.RunMigration(ctx, targetName); err != nil {
		t.Fatalf("failed to run migration %q: %v", targetName, err)
	}
}

func runAllModuleMigrations(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	migrations *migrate.Migrations,
) {
	t.Helper()

	migrator := migrate.NewMigrator(db, migrations, migrate.WithUpsert(true))
	if err := migrator.Init(ctx); err != nil {
		t.Fatalf("failed to init module migrator: %v", err)
	}

	if _, err := migrator.Migrate(ctx); err != nil {
		t.Fatalf("failed to run all module migrations: %v", err)
	}
}
