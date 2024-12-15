// db/bundb/bundb.go
package bundb

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// DBService satisfies the db.Database interface
type DBService struct {
	User    *userdb.UserDBImpl
	RoundDB *rounddb.RoundDBImpl
	db      *bun.DB
}

// GetDB returns the underlying database connection pool.
func (dbService *DBService) GetDB() *bun.DB {
	return dbService.db
}

func NewBunDBService(ctx context.Context, dsn string) (*DBService, error) {
	log.Printf("NewBunDBService - Initializing with DSN: %s", dsn)

	sqldb, err := pgConn(dsn)
	if err != nil {
		log.Printf("NewBunDBService - Failed to connect to PostgreSQL: %v", err)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db := bunDB(sqldb)
	if db == nil {
		log.Println("NewBunDBService - bunDB returned nil")
		return nil, fmt.Errorf("failed to initialize bun.DB")
	}

	dbService := &DBService{
		User:    &userdb.UserDBImpl{DB: db},
		RoundDB: &rounddb.RoundDBImpl{DB: db},
		db:      db,
	}

	log.Printf("NewBunDBService - DBService initialized: %+v", dbService)

	log.Println("NewBunDBService - Registering models")
	db.RegisterModel(&userdb.User{})
	db.RegisterModel(&rounddb.Round{})
	log.Println("NewBunDBService - Models registered successfully")

	return dbService, nil
}

// bunDB returns a new bun.DB for given sql.DB connection pool.
func bunDB(sqldb *sql.DB) *bun.DB {
	db := bun.NewDB(sqldb, pgdialect.New())
	return db
}

func pgConn(dsn string) (*sql.DB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	if err := sqldb.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return sqldb, nil
}
