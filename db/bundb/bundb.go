package bundb

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// DBService satisfies the db.Database interface
type DBService struct {
	UserDB        userdb.Repository
	RoundDB       rounddb.Repository
	ScoreDB       scoredb.Repository
	LeaderboardDB leaderboarddb.Repository
	GuildDB       guilddb.Repository
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
	return newDBServiceWithDB(db)
}

// NewTestDBService creates a new DBService with the provided bun.DB instance
// This is useful for integration tests where we inject a test database
func NewTestDBService(db *bun.DB) (*DBService, error) {
	return newDBServiceWithDB(db)
}

// newDBServiceWithDB is a helper function to create a DBService with a provided bun.DB
func newDBServiceWithDB(db *bun.DB) (*DBService, error) {
	if db == nil {
		log.Println("newDBServiceWithDB - received nil db")
		return nil, fmt.Errorf("failed to initialize: nil db provided")
	}

	// Register models
	log.Println("newDBServiceWithDB - Registering models")
	db.RegisterModel(&userdb.User{})
	db.RegisterModel(&rounddb.Round{})
	db.RegisterModel(&scoredb.Score{})
	db.RegisterModel(&leaderboarddb.Leaderboard{})
	db.RegisterModel(&guilddb.GuildConfig{})
	log.Println("newDBServiceWithDB - Models registered successfully")

	dbService := &DBService{
		UserDB:        userdb.NewRepository(db),
		RoundDB:       rounddb.NewRepository(db),
		ScoreDB:       scoredb.NewRepository(db),
		LeaderboardDB: leaderboarddb.NewRepository(db),
		GuildDB:       guilddb.NewRepository(db),
		db:            db,
	}

	log.Printf("newDBServiceWithDB - DBService initialized: %+v", dbService)
	return dbService, nil
}

// BunDB returns a new bun.DB for given sql.DB connection pool - exported for testing
func BunDB(sqlDB *sql.DB) *bun.DB {
	return bun.NewDB(sqlDB, pgdialect.New())
}

// Internal version for regular use
func bunDB(sqldb *sql.DB) *bun.DB {
	return BunDB(sqldb)
}

// PgConn creates a new SQL DB connection - exported for testing
func PgConn(dsn string) (*sql.DB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	if err := sqldb.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return sqldb, nil
}

// Internal version for regular use
func pgConn(dsn string) (*sql.DB, error) {
	return PgConn(dsn)
}
