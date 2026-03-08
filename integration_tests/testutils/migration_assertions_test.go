package testutils

import (
	"context"
	"slices"
	"testing"

	"github.com/uptrace/bun"
)

func assertTableMissing(t *testing.T, ctx context.Context, db *bun.DB, tableName string) {
	t.Helper()

	var exists bool
	err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = ?
		)`,
		tableName,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to query table existence for %q: %v", tableName, err)
	}
	if exists {
		t.Fatalf("expected table %q to be absent", tableName)
	}
}

func assertForeignKeyConstraintReferences(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	constraintName string,
	expectedTable string,
) {
	t.Helper()

	var referencedTable string
	err := db.QueryRowContext(
		ctx,
		`SELECT ccu.table_name
		 FROM information_schema.table_constraints tc
		 JOIN information_schema.constraint_column_usage ccu
		   ON tc.constraint_schema = ccu.constraint_schema
		  AND tc.constraint_name = ccu.constraint_name
		 WHERE tc.table_schema = 'public'
		   AND tc.constraint_type = 'FOREIGN KEY'
		   AND tc.constraint_name = ?`,
		constraintName,
	).Scan(&referencedTable)
	if err != nil {
		t.Fatalf("failed to query foreign key %q: %v", constraintName, err)
	}
	if referencedTable != expectedTable {
		t.Fatalf(
			"unexpected referenced table for constraint %q: got=%q want=%q",
			constraintName,
			referencedTable,
			expectedTable,
		)
	}
}

func assertPrimaryKeyColumns(t *testing.T, ctx context.Context, db *bun.DB, tableName string, expected []string) {
	t.Helper()

	rows, err := db.QueryContext(
		ctx,
		`SELECT kcu.column_name
		 FROM information_schema.table_constraints tc
		 JOIN information_schema.key_column_usage kcu
		   ON tc.constraint_name = kcu.constraint_name
		  AND tc.table_schema = kcu.table_schema
		 WHERE tc.table_schema = 'public'
		   AND tc.table_name = ?
		   AND tc.constraint_type = 'PRIMARY KEY'
		 ORDER BY kcu.ordinal_position`,
		tableName,
	)
	if err != nil {
		t.Fatalf("failed querying primary key columns for %q: %v", tableName, err)
	}
	defer rows.Close()

	actual := make([]string, 0, len(expected))
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			t.Fatalf("failed scanning primary key column row: %v", err)
		}
		actual = append(actual, column)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("error iterating primary key column rows: %v", err)
	}

	if !slices.Equal(actual, expected) {
		t.Fatalf("unexpected primary key columns for %q: got=%v want=%v", tableName, actual, expected)
	}
}
