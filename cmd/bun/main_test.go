package main

import (
	"reflect"
	"testing"

	"github.com/uptrace/bun/migrate"
)

func TestOrderedModuleNames(t *testing.T) {
	t.Parallel()

	migrators := map[string]*migrate.Migrator{
		"guild":       nil,
		"user":        nil,
		"club":        nil,
		"round":       nil,
		"score":       nil,
		"leaderboard": nil,
	}

	t.Run("forward order", func(t *testing.T) {
		t.Parallel()

		got, err := orderedModuleNames(migrators, false)
		if err != nil {
			t.Fatalf("orderedModuleNames returned error: %v", err)
		}

		if !reflect.DeepEqual(got, dependencyOrderedModules) {
			t.Fatalf("unexpected forward order: got=%v want=%v", got, dependencyOrderedModules)
		}
	})

	t.Run("reverse order", func(t *testing.T) {
		t.Parallel()

		want := []string{"leaderboard", "score", "round", "club", "user", "guild"}

		got, err := orderedModuleNames(migrators, true)
		if err != nil {
			t.Fatalf("orderedModuleNames returned error: %v", err)
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected reverse order: got=%v want=%v", got, want)
		}
	})
}

func TestOrderedModuleNames_ValidationErrors(t *testing.T) {
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
			"weird":       nil,
		}

		_, err := orderedModuleNames(migrators, false)
		if err == nil {
			t.Fatal("expected error for unknown module")
		}
	})
}
