// internal/db/bundb/bundb.go
package bundb

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/Black-And-White-Club/tcr-bot/app/models"
	"github.com/Black-And-White-Club/tcr-bot/internal/db" // Ensure this import is correct
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// DBService satisfies the db.Database interface
type DBService struct {
	User        db.UserDB        // Ensure this is the correct interface type
	Leaderboard db.LeaderboardDB // Ensure this is the correct interface type
	Round       db.RoundDB       // Ensure this is the correct interface type
	Score       db.ScoreDB       // Ensure this is the correct interface type
	db          *bun.DB
}

// GetDB returns the underlying database connection pool.
func (dbService *DBService) GetDB() *bun.DB {
	return dbService.db
}

func NewBunDBService(ctx context.Context, dsn string) (*DBService, error) {
	log.Printf("NewBunDBService - Initializing with DSN: %s", dsn)

	// Step 1: Create pgConn
	sqldb, err := pgConn(dsn)
	if err != nil {
		log.Printf("NewBunDBService - Failed to connect to PostgreSQL: %v", err)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Step 2: Initialize bunDB
	db := bunDB(sqldb)
	if db == nil {
		log.Println("NewBunDBService - bunDB returned nil")
		return nil, fmt.Errorf("failed to initialize bun.DB")
	}

	// Step 3: Initialize DBService
	dbService := &DBService{
		User:        &userDB{db: db},
		Leaderboard: &leaderboardDB{db: db},
		Round:       &roundDB{db: db},
		Score:       &scoreDB{db: db},
		db:          db,
	}

	log.Printf("NewBunDBService - DBService initialized: %+v", dbService)

	// Step 4: Register Models
	log.Println("NewBunDBService - Registering models")
	db.RegisterModel(&models.User{})
	db.RegisterModel(&models.Leaderboard{})
	db.RegisterModel(&models.Round{})
	db.RegisterModel(&models.Score{})
	db.RegisterModel(&models.Participant{})
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
