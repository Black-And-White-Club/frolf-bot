// db/bundb/bundb.go
package bundb

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	// Import for db.RoundDB
	rounddb "github.com/Black-And-White-Club/tcr-bot/round/db"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// DBService satisfies the db.Database interface
type DBService struct {
	// User        db.UserDB       // Comment out until User module is refactored
	// Leaderboard db.LeaderboardDB // Comment out until Leaderboard module is refactored
	Round rounddb.RoundDB // Use the interface from round/db
	// Score       db.ScoreDB     // Comment out until Score module is refactored
	db *bun.DB
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
		// User:        &userDB{db: db},       // Comment out until User module is refactored
		// Leaderboard: &leaderboardDB{db: db}, // Comment out until Leaderboard module is refactored
		Round: &rounddb.RoundDBImpl{DB: db}, // Use uppercase DB
		// Score:       &scoreDB{db: db},     // Comment out until Score module is refactored
		db: db,
	}

	log.Printf("NewBunDBService - DBService initialized: %+v", dbService)

	// Step 4: Register Models
	log.Println("NewBunDBService - Registering models")
	// db.RegisterModel(&models.User{})                // Comment out until User module is refactored
	// db.RegisterModel(&models.Leaderboard{})        // Comment out until Leaderboard module is refactored
	db.RegisterModel(&rounddb.Round{}) // Use the Round model from round/db
	// db.RegisterModel(&models.Score{})              // Comment out until Score module is refactored
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
