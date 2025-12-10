package roundmigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding import fields to rounds table...")

		// Add import-related columns to rounds table
		if _, err := db.NewAddColumn().
			Model((*interface{})(nil)).
			Table("rounds").
			ColumnExpr("import_id TEXT").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}

		if _, err := db.NewAddColumn().
			Model((*interface{})(nil)).
			Table("rounds").
			ColumnExpr("import_status TEXT").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}

		if _, err := db.NewAddColumn().
			Model((*interface{})(nil)).
			Table("rounds").
			ColumnExpr("import_type TEXT").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}

		if _, err := db.NewAddColumn().
			Model((*interface{})(nil)).
			Table("rounds").
			ColumnExpr("file_data BYTEA").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}

		if _, err := db.NewAddColumn().
			Model((*interface{})(nil)).
			Table("rounds").
			ColumnExpr("file_name TEXT").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}

		if _, err := db.NewAddColumn().
			Model((*interface{})(nil)).
			Table("rounds").
			ColumnExpr("udisc_url TEXT").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}

		if _, err := db.NewAddColumn().
			Model((*interface{})(nil)).
			Table("rounds").
			ColumnExpr("import_notes TEXT").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}

		if _, err := db.NewAddColumn().
			Model((*interface{})(nil)).
			Table("rounds").
			ColumnExpr("import_error TEXT").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}

		if _, err := db.NewAddColumn().
			Model((*interface{})(nil)).
			Table("rounds").
			ColumnExpr("import_error_code TEXT").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}

		if _, err := db.NewAddColumn().
			Model((*interface{})(nil)).
			Table("rounds").
			ColumnExpr("imported_at TIMESTAMP").
			IfNotExists().
			Exec(ctx); err != nil {
			return err
		}

		fmt.Println("Import fields added to rounds table successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Removing import fields from rounds table...")

		// Drop import-related columns
		columns := []string{
			"import_id", "import_status", "import_type", "file_data",
			"file_name", "udisc_url", "import_notes", "import_error",
			"import_error_code", "imported_at",
		}

		for _, col := range columns {
			if _, err := db.NewDropColumn().
				Model((*interface{})(nil)).
				Table("rounds").
				Column(col).
				Exec(ctx); err != nil {
				return err
			}
		}

		fmt.Println("Import fields removed from rounds table successfully!")
		return nil
	})
}
