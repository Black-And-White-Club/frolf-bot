package migrationrunner

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	clubmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories/migrations"
	guildmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories/migrations"
	leaderboardmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/migrations"
	roundmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/migrations"
	scoremigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories/migrations"
	usermigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// ModuleConfig describes one migration module's dependency order and tracking table.
type ModuleConfig struct {
	Name       string
	TableName  string
	Migrations *migrate.Migrations
}

var dependencyOrderedModules = []ModuleConfig{
	{Name: "guild", TableName: "bun_migrations_guild", Migrations: guildmigrations.Migrations},
	{Name: "user", TableName: "bun_migrations_user", Migrations: usermigrations.Migrations},
	{Name: "club", TableName: "bun_migrations_club", Migrations: clubmigrations.Migrations},
	{Name: "round", TableName: "bun_migrations_round", Migrations: roundmigrations.Migrations},
	{Name: "score", TableName: "bun_migrations_score", Migrations: scoremigrations.Migrations},
	{Name: "leaderboard", TableName: "bun_migrations_leaderboard", Migrations: leaderboardmigrations.Migrations},
}

const sharedMigrationTableName = "bun_migrations"

// OrderedModuleConfigs returns migration modules in dependency order.
func OrderedModuleConfigs() []ModuleConfig {
	copied := make([]ModuleConfig, len(dependencyOrderedModules))
	copy(copied, dependencyOrderedModules)
	return copied
}

// OrderedModuleNamesFromConfig returns the canonical migration module order.
func OrderedModuleNamesFromConfig() []string {
	names := make([]string, 0, len(dependencyOrderedModules))
	for _, module := range dependencyOrderedModules {
		names = append(names, module.Name)
	}
	return names
}

// OrderedModuleNames validates migrator keys and returns canonical module order.
func OrderedModuleNames[T any](migrators map[string]T, reverse bool) ([]string, error) {
	if len(migrators) == 0 {
		return nil, errors.New("no migrators configured")
	}

	expected := OrderedModuleNamesFromConfig()
	known := make(map[string]struct{}, len(expected))
	for _, moduleName := range expected {
		known[moduleName] = struct{}{}
	}

	var missing []string
	for _, moduleName := range expected {
		if _, ok := migrators[moduleName]; !ok {
			missing = append(missing, moduleName)
		}
	}

	var unknown []string
	for moduleName := range migrators {
		if _, ok := known[moduleName]; !ok {
			unknown = append(unknown, moduleName)
		}
	}

	slices.Sort(missing)
	slices.Sort(unknown)

	if len(missing) > 0 || len(unknown) > 0 {
		return nil, fmt.Errorf("invalid migrator set: missing=%v unknown=%v", missing, unknown)
	}

	ordered := make([]string, len(expected))
	copy(ordered, expected)
	if reverse {
		slices.Reverse(ordered)
	}

	return ordered, nil
}

// BuildBunMigrators builds Bun migrators for each module using canonical table names.
func BuildBunMigrators(db *bun.DB) map[string]*migrate.Migrator {
	migrators := make(map[string]*migrate.Migrator, len(dependencyOrderedModules))
	for _, module := range dependencyOrderedModules {
		migrators[module.Name] = migrate.NewMigrator(
			db,
			module.Migrations,
			migrate.WithTableName(module.TableName),
		)
	}
	return migrators
}

// BuildSharedTableMigrators keeps startup migrations on the historical shared table
// so existing databases do not replay already-applied migrations.
func BuildSharedTableMigrators(db *bun.DB) map[string]*migrate.Migrator {
	migrators := make(map[string]*migrate.Migrator, len(dependencyOrderedModules))
	for _, module := range dependencyOrderedModules {
		migrators[module.Name] = migrate.NewMigrator(
			db,
			module.Migrations,
			migrate.WithTableName(sharedMigrationTableName),
		)
	}
	return migrators
}

// ModuleMigrator is the subset of migrator behavior used by migration runners.
type ModuleMigrator interface {
	Init(ctx context.Context) error
	Migrate(ctx context.Context, opts ...migrate.MigrationOption) (*migrate.MigrationGroup, error)
	Rollback(ctx context.Context, opts ...migrate.MigrationOption) (*migrate.MigrationGroup, error)
}

// AsModuleMigrators adapts Bun migrators to the module runner interface.
func AsModuleMigrators(migrators map[string]*migrate.Migrator) map[string]ModuleMigrator {
	adapted := make(map[string]ModuleMigrator, len(migrators))
	for moduleName, migrator := range migrators {
		adapted[moduleName] = migrator
	}
	return adapted
}

// ModuleRunResult captures one module execution result.
type ModuleRunResult struct {
	Module string
	Group  *migrate.MigrationGroup
}

// InitModules initializes migration tables for all modules in dependency order.
func InitModules(ctx context.Context, migrators map[string]ModuleMigrator) error {
	moduleNames, err := OrderedModuleNames(migrators, false)
	if err != nil {
		return err
	}
	for _, moduleName := range moduleNames {
		if err := migrators[moduleName].Init(ctx); err != nil {
			return fmt.Errorf("init %s migrations: %w", moduleName, err)
		}
	}
	return nil
}

// MigrateModules applies migrations for all modules in dependency order.
func MigrateModules(ctx context.Context, migrators map[string]ModuleMigrator) ([]ModuleRunResult, error) {
	moduleNames, err := OrderedModuleNames(migrators, false)
	if err != nil {
		return nil, err
	}

	results := make([]ModuleRunResult, 0, len(moduleNames))
	for _, moduleName := range moduleNames {
		group, migrateErr := migrators[moduleName].Migrate(ctx)
		if migrateErr != nil {
			return nil, fmt.Errorf("migrate %s module: %w", moduleName, migrateErr)
		}
		results = append(results, ModuleRunResult{Module: moduleName, Group: group})
	}
	return results, nil
}

// RollbackModules rolls back each module in reverse dependency order.
func RollbackModules(ctx context.Context, migrators map[string]ModuleMigrator) ([]ModuleRunResult, error) {
	moduleNames, err := OrderedModuleNames(migrators, true)
	if err != nil {
		return nil, err
	}

	results := make([]ModuleRunResult, 0, len(moduleNames))
	for _, moduleName := range moduleNames {
		group, rollbackErr := migrators[moduleName].Rollback(ctx)
		if rollbackErr != nil {
			return nil, fmt.Errorf("rollback %s module: %w", moduleName, rollbackErr)
		}
		results = append(results, ModuleRunResult{Module: moduleName, Group: group})
	}
	return results, nil
}

// MigrateRiver runs River queue migrations.
func MigrateRiver(ctx context.Context, dsn string) error {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect for River migrations: %w", err)
	}
	defer pool.Close()

	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("create River migrator: %w", err)
	}

	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{}); err != nil {
		if strings.Contains(err.Error(), "already exists") && strings.Contains(err.Error(), "river_migration") {
			return nil
		}
		return fmt.Errorf("run River migrations: %w", err)
	}

	return nil
}
