package migrationrunner

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

type fakeModuleMigrator struct {
	module      string
	record      *[]string
	initErr     error
	migrateErr  error
	rollbackErr error
	group       *migrate.MigrationGroup
}

func (f *fakeModuleMigrator) Init(context.Context) error {
	*f.record = append(*f.record, "init:"+f.module)
	return f.initErr
}

func (f *fakeModuleMigrator) Migrate(context.Context, ...migrate.MigrationOption) (*migrate.MigrationGroup, error) {
	*f.record = append(*f.record, "migrate:"+f.module)
	if f.group == nil {
		return &migrate.MigrationGroup{}, f.migrateErr
	}
	return f.group, f.migrateErr
}

func (f *fakeModuleMigrator) Rollback(context.Context, ...migrate.MigrationOption) (*migrate.MigrationGroup, error) {
	*f.record = append(*f.record, "rollback:"+f.module)
	if f.group == nil {
		return &migrate.MigrationGroup{}, f.rollbackErr
	}
	return f.group, f.rollbackErr
}

func buildFakeMigrators(record *[]string) map[string]ModuleMigrator {
	moduleNames := OrderedModuleNamesFromConfig()
	migrators := make(map[string]ModuleMigrator, len(moduleNames))
	for _, moduleName := range moduleNames {
		migrators[moduleName] = &fakeModuleMigrator{module: moduleName, record: record}
	}
	return migrators
}

func TestOrderedModuleNames(t *testing.T) {
	tests := []struct {
		name    string
		reverse bool
		want    []string
	}{
		{
			name:    "forward order",
			reverse: false,
			want:    OrderedModuleNamesFromConfig(),
		},
		{
			name:    "reverse order",
			reverse: true,
			want:    []string{"leaderboard", "score", "round", "club", "user", "guild"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			migrators := map[string]int{}
			for _, moduleName := range OrderedModuleNamesFromConfig() {
				migrators[moduleName] = 1
			}

			got, err := OrderedModuleNames(migrators, tc.reverse)
			if err != nil {
				t.Fatalf("OrderedModuleNames returned error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("unexpected module order: got=%v want=%v", got, tc.want)
			}
		})
	}
}

func TestOrderedModuleNamesValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(map[string]int)
		wantError string
	}{
		{
			name: "missing module",
			mutate: func(migrators map[string]int) {
				delete(migrators, "score")
			},
			wantError: "missing=[score]",
		},
		{
			name: "unknown module",
			mutate: func(migrators map[string]int) {
				migrators["weird"] = 1
			},
			wantError: "unknown=[weird]",
		},
		{
			name: "empty map",
			mutate: func(map[string]int) {
			},
			wantError: "no migrators configured",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			migrators := map[string]int{}
			for _, moduleName := range OrderedModuleNamesFromConfig() {
				migrators[moduleName] = 1
			}
			if tc.name == "empty map" {
				migrators = map[string]int{}
			}
			tc.mutate(migrators)

			_, err := OrderedModuleNames(migrators, false)
			if err == nil {
				t.Fatalf("expected error containing %q", tc.wantError)
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("unexpected error: got=%q want substring=%q", err, tc.wantError)
			}
		})
	}
}

func TestInitMigrateRollbackModules(t *testing.T) {
	tests := []struct {
		name               string
		setup              func(migrators map[string]ModuleMigrator)
		run                func(context.Context, map[string]ModuleMigrator) error
		wantOrder          []string
		wantErrContains    string
		wantErrShouldExist bool
	}{
		{
			name: "init modules in dependency order",
			run: func(ctx context.Context, migrators map[string]ModuleMigrator) error {
				return InitModules(ctx, migrators)
			},
			wantOrder: []string{
				"init:guild",
				"init:user",
				"init:club",
				"init:round",
				"init:score",
				"init:leaderboard",
			},
		},
		{
			name: "init stops on module error",
			setup: func(migrators map[string]ModuleMigrator) {
				migrators["club"] = &fakeModuleMigrator{
					module:  "club",
					record:  migrators["guild"].(*fakeModuleMigrator).record,
					initErr: errors.New("boom"),
				}
			},
			run: func(ctx context.Context, migrators map[string]ModuleMigrator) error {
				return InitModules(ctx, migrators)
			},
			wantOrder: []string{
				"init:guild",
				"init:user",
				"init:club",
			},
			wantErrContains:    "init club migrations",
			wantErrShouldExist: true,
		},
		{
			name: "migrate modules in dependency order",
			run: func(ctx context.Context, migrators map[string]ModuleMigrator) error {
				_, err := MigrateModules(ctx, migrators)
				return err
			},
			wantOrder: []string{
				"migrate:guild",
				"migrate:user",
				"migrate:club",
				"migrate:round",
				"migrate:score",
				"migrate:leaderboard",
			},
		},
		{
			name: "migrate stops on module error",
			setup: func(migrators map[string]ModuleMigrator) {
				migrators["round"] = &fakeModuleMigrator{
					module:     "round",
					record:     migrators["guild"].(*fakeModuleMigrator).record,
					migrateErr: errors.New("boom"),
				}
			},
			run: func(ctx context.Context, migrators map[string]ModuleMigrator) error {
				_, err := MigrateModules(ctx, migrators)
				return err
			},
			wantOrder: []string{
				"migrate:guild",
				"migrate:user",
				"migrate:club",
				"migrate:round",
			},
			wantErrContains:    "migrate round module",
			wantErrShouldExist: true,
		},
		{
			name: "rollback modules in reverse order",
			run: func(ctx context.Context, migrators map[string]ModuleMigrator) error {
				_, err := RollbackModules(ctx, migrators)
				return err
			},
			wantOrder: []string{
				"rollback:leaderboard",
				"rollback:score",
				"rollback:round",
				"rollback:club",
				"rollback:user",
				"rollback:guild",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			record := make([]string, 0, 8)
			migrators := buildFakeMigrators(&record)
			if tc.setup != nil {
				tc.setup(migrators)
			}

			err := tc.run(context.Background(), migrators)
			if tc.wantErrShouldExist && err == nil {
				t.Fatalf("expected error containing %q", tc.wantErrContains)
			}
			if !tc.wantErrShouldExist && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErrShouldExist && err != nil && !strings.Contains(err.Error(), tc.wantErrContains) {
				t.Fatalf("unexpected error: got=%v want substring=%q", err, tc.wantErrContains)
			}
			if !reflect.DeepEqual(record, tc.wantOrder) {
				t.Fatalf("unexpected call order:\n got=%v\nwant=%v", record, tc.wantOrder)
			}
		})
	}
}

func TestOrderedModuleConfigs_TableNames(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "all module table names use bun_migrations_ prefix"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			for _, module := range OrderedModuleConfigs() {
				if !strings.HasPrefix(module.TableName, "bun_migrations_") {
					t.Fatalf("module %s has unexpected table name: %s", module.Name, module.TableName)
				}
				if module.Migrations == nil {
					t.Fatalf("module %s has nil migrations", module.Name)
				}
				if module.Name == "" {
					t.Fatalf("encountered empty module name in config: %+v", module)
				}
			}
		})
	}
}

func migratorTableName(t *testing.T, migrator *migrate.Migrator) string {
	t.Helper()

	value := reflect.ValueOf(migrator)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		t.Fatal("migrator must be a non-nil pointer")
	}

	field := value.Elem().FieldByName("table")
	if !field.IsValid() || field.Kind() != reflect.String {
		t.Fatal("migrator table field missing")
	}

	return field.String()
}

func TestBuildBunMigrators_UsesModuleTables(t *testing.T) {
	t.Parallel()

	var db *bun.DB
	migrators := BuildBunMigrators(db)

	for _, module := range OrderedModuleConfigs() {
		migrator, ok := migrators[module.Name]
		if !ok {
			t.Fatalf("missing migrator for module %q", module.Name)
		}
		if got := migratorTableName(t, migrator); got != module.TableName {
			t.Fatalf("module %s table mismatch: got=%s want=%s", module.Name, got, module.TableName)
		}
	}
}

func TestBuildSharedTableMigrators_UsesSharedTable(t *testing.T) {
	t.Parallel()

	var db *bun.DB
	migrators := BuildSharedTableMigrators(db)

	for _, module := range OrderedModuleConfigs() {
		migrator, ok := migrators[module.Name]
		if !ok {
			t.Fatalf("missing migrator for module %q", module.Name)
		}
		if got := migratorTableName(t, migrator); got != sharedMigrationTableName {
			t.Fatalf("module %s table mismatch: got=%s want=%s", module.Name, got, sharedMigrationTableName)
		}
	}
}

func TestAsModuleMigrators(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "adapts all migrators"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bunMigrators := map[string]*migrate.Migrator{
				"guild": nil,
				"user":  nil,
			}
			got := AsModuleMigrators(bunMigrators)
			if len(got) != len(bunMigrators) {
				t.Fatalf("unexpected adapted migrator count: got=%d want=%d", len(got), len(bunMigrators))
			}
			for moduleName := range bunMigrators {
				if _, ok := got[moduleName]; !ok {
					t.Fatalf("missing adapted module %q", moduleName)
				}
			}
		})
	}
}

func TestOrderedModuleNamesFromConfig(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{
			name: "matches dependency module configs",
			want: []string{"guild", "user", "club", "round", "score", "leaderboard"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := OrderedModuleNamesFromConfig()
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("unexpected module names: got=%v want=%v", got, tc.want)
			}
		})
	}
}

func TestOrderedModuleNames_ErrorFormatting(t *testing.T) {
	tests := []struct {
		name      string
		migrators map[string]int
		want      []string
	}{
		{
			name:      "missing and unknown sorted in error",
			migrators: map[string]int{"guild": 1, "user": 1, "club": 1, "round": 1, "bogus": 1},
			want:      []string{"missing=[leaderboard score]", "unknown=[bogus]"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := OrderedModuleNames(tc.migrators, false)
			if err == nil {
				t.Fatal("expected validation error")
			}
			for _, part := range tc.want {
				if !strings.Contains(err.Error(), part) {
					t.Fatalf("error %q missing expected segment %q", err.Error(), part)
				}
			}
		})
	}
}

func TestInitModules_InvalidMigratorSet(t *testing.T) {
	tests := []struct {
		name      string
		migrators map[string]ModuleMigrator
		wantError string
	}{
		{
			name:      "empty",
			migrators: map[string]ModuleMigrator{},
			wantError: "no migrators configured",
		},
		{
			name: "unknown module",
			migrators: map[string]ModuleMigrator{
				"weird": &fakeModuleMigrator{module: "weird", record: &[]string{}},
			},
			wantError: "unknown=[weird]",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := InitModules(context.Background(), tc.migrators)
			if err == nil {
				t.Fatalf("expected error containing %q", tc.wantError)
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("unexpected error: got=%v want substring=%q", err, tc.wantError)
			}
		})
	}
}

func TestMigrateModules_PropagatesGroup(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "returns module groups from migrator"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			record := make([]string, 0)
			migrators := buildFakeMigrators(&record)
			migrators["guild"] = &fakeModuleMigrator{
				module: "guild",
				record: &record,
				group:  &migrate.MigrationGroup{ID: 42},
			}

			results, err := MigrateModules(context.Background(), migrators)
			if err != nil {
				t.Fatalf("MigrateModules returned error: %v", err)
			}

			if len(results) != len(OrderedModuleNamesFromConfig()) {
				t.Fatalf("unexpected results length: got=%d want=%d", len(results), len(OrderedModuleNamesFromConfig()))
			}
			if results[0].Module != "guild" {
				t.Fatalf("unexpected first result module: got=%s", results[0].Module)
			}
			if results[0].Group == nil || results[0].Group.ID != 42 {
				t.Fatalf("unexpected first result group: got=%v", results[0].Group)
			}
		})
	}
}

func TestRollbackModules_PropagatesErrorContext(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "includes module name in rollback errors"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			record := make([]string, 0)
			migrators := buildFakeMigrators(&record)
			migrators["score"] = &fakeModuleMigrator{
				module:      "score",
				record:      &record,
				rollbackErr: fmt.Errorf("forced failure"),
			}

			_, err := RollbackModules(context.Background(), migrators)
			if err == nil {
				t.Fatal("expected rollback error")
			}
			if !strings.Contains(err.Error(), "rollback score module") {
				t.Fatalf("unexpected rollback error: %v", err)
			}
		})
	}
}
