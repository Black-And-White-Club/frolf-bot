// db/bundb/bundb.go
package bundb

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/repositories"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/repositories"
	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories"
	"github.com/Black-And-White-Club/tcr-bot/config"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// DBService satisfies the db.Database interface
type DBService struct {
	UserDB  *userdb.UserDBImpl
	RoundDB *rounddb.RoundDBImpl
	// ScoreDB       *scoredb.ScoreDBImpl
	LeaderboardDB *leaderboarddb.LeaderboardDBImpl
	db            *bun.DB
}

// GetDB returns the underlying database connection pool.
func (dbService *DBService) GetDB() *bun.DB {
	return dbService.db
}

// NewBunDBService initializes a new DBService with the provided Postgres configuration.
func NewBunDBService(ctx context.Context, cfg config.PostgresConfig) (*DBService, error) {
	log.Printf("NewBunDBService - Initializing with DSN: %s", cfg.DSN)

	sqldb, err := pgConn(cfg.DSN)
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
		UserDB:  &userdb.UserDBImpl{DB: db},
		RoundDB: &rounddb.RoundDBImpl{DB: db},
		// ScoreDB:       &scoredb.ScoreDBImpl{DB: db},
		LeaderboardDB: &leaderboarddb.LeaderboardDBImpl{DB: db},
		db:            db,
	}

	log.Printf("NewBunDBService - DBService initialized: %+v", dbService)

	log.Println("NewBunDBService - Registering models")
	// Use the correct model types from their respective modules
	db.RegisterModel(&userdb.User{})
	db.RegisterModel(&rounddb.Round{})
	db.RegisterModel(&scoredb.Score{})
	db.RegisterModel(&leaderboarddb.Leaderboard{})
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
