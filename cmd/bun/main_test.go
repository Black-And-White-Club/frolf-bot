package main

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Black-And-White-Club/frolf-bot/app/shared/migrationrunner"
	"github.com/uptrace/bun/migrate"
)

func TestOrderedModuleNames(t *testing.T) {
	__codexTDCases := []struct {
		name string
	}{
		{name: "default"},
	}

	for _, __codexTDCase := range __codexTDCases {
		t.Run(__codexTDCase.name, func(t *testing.T) {
			t.Parallel()

			migrators := map[string]*migrate.Migrator{
				"guild":       nil,
				"user":        nil,
				"club":        nil,
				"round":       nil,
				"score":       nil,
				"leaderboard": nil,
				"betting":     nil,
			}

			t.Run("forward order", func(t *testing.T) {
				t.Parallel()

				got, err := orderedModuleNames(migrators, false)
				if err != nil {
					t.Fatalf("orderedModuleNames returned error: %v", err)
				}

				want := migrationrunner.OrderedModuleNamesFromConfig()
				if !slices.Equal(got, want) {
					t.Fatalf("unexpected forward order: got=%v want=%v", got, want)
				}
			})

			t.Run("reverse order", func(t *testing.T) {
				t.Parallel()

				want := []string{"betting", "leaderboard", "score", "round", "club", "user", "guild"}

				got, err := orderedModuleNames(migrators, true)
				if err != nil {
					t.Fatalf("orderedModuleNames returned error: %v", err)
				}

				if !slices.Equal(got, want) {
					t.Fatalf("unexpected reverse order: got=%v want=%v", got, want)
				}
			})
		})
	}
}

type fakeModuleMigrator struct {
	initErr error
	record  *[]string
	name    string
}

func (f *fakeModuleMigrator) Init(context.Context) error {
	*f.record = append(*f.record, f.name)
	return f.initErr
}

func (f *fakeModuleMigrator) Migrate(context.Context, ...migrate.MigrationOption) (*migrate.MigrationGroup, error) {
	return &migrate.MigrationGroup{}, nil
}

func (f *fakeModuleMigrator) Rollback(context.Context, ...migrate.MigrationOption) (*migrate.MigrationGroup, error) {
	return &migrate.MigrationGroup{}, nil
}

func TestInitModules_LogsOnlyAttemptedModules(t *testing.T) {
	t.Parallel()

	record := make([]string, 0, len(migrationrunner.OrderedModuleNamesFromConfig()))
	migrators := make(map[string]migrationrunner.ModuleMigrator, len(migrationrunner.OrderedModuleNamesFromConfig()))
	for _, moduleName := range migrationrunner.OrderedModuleNamesFromConfig() {
		migrators[moduleName] = &fakeModuleMigrator{
			name:   moduleName,
			record: &record,
		}
	}
	migrators["club"] = &fakeModuleMigrator{
		name:    "club",
		record:  &record,
		initErr: errors.New("boom"),
	}

	var out bytes.Buffer
	err := initModules(context.Background(), &out, migrators)
	if err == nil {
		t.Fatal("expected initModules error")
	}
	if !strings.Contains(err.Error(), "init club migrations") {
		t.Fatalf("unexpected error: %v", err)
	}

	wantRecord := []string{"guild", "user", "club"}
	if !slices.Equal(record, wantRecord) {
		t.Fatalf("unexpected init order: got=%v want=%v", record, wantRecord)
	}

	wantLog := strings.Join([]string{
		"Initializing migrations for module: guild",
		"Initializing migrations for module: user",
		"Initializing migrations for module: club",
		"",
	}, "\n")
	if out.String() != wantLog {
		t.Fatalf("unexpected log output:\n got=%q\nwant=%q", out.String(), wantLog)
	}
}

func TestOrderedModuleNames_ValidationErrors(t *testing.T) {
	__codexTDCases := []struct {
		name string
	}{
		{name: "default"},
	}

	for _, __codexTDCase := range __codexTDCases {
		t.Run(__codexTDCase.name, func(t *testing.T) {
			t.Parallel()

			t.Run("missing module", func(t *testing.T) {
				t.Parallel()

				migrators := map[string]*migrate.Migrator{
					"guild":       nil,
					"user":        nil,
					"club":        nil,
					"round":       nil,
					"score":       nil,
					"leaderboard": nil,
					"betting":     nil,
				}
				delete(migrators, "score")

				_, err := orderedModuleNames(migrators, false)
				if err == nil {
					t.Fatal("expected error for missing module")
				}
			})

			t.Run("unknown module", func(t *testing.T) {
				t.Parallel()

				migrators := map[string]*migrate.Migrator{
					"guild":       nil,
					"user":        nil,
					"club":        nil,
					"round":       nil,
					"score":       nil,
					"leaderboard": nil,
					"betting":     nil,
					"weird":       nil,
				}

				_, err := orderedModuleNames(migrators, false)
				if err == nil {
					t.Fatal("expected error for unknown module")
				}
			})
		})
	}
}
