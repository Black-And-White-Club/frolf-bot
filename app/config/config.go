package config

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// Config struct to hold the configuration settings
type Config struct {
	DB  *bun.DB
	DSN string

	// NATS configuration
	NATS struct {
		URL string `yaml:"url"`
	} `yaml:"nats"`
}

// NewConfig creates a new Config instance with database connection and NATS URL
func NewConfig(ctx context.Context) *Config {
	// Database connection setup
	dsn := os.Getenv("DATABASE_URL")

	fmt.Println("DSN:", dsn)

	// Use sql.OpenDB to open the database connection
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())

	// Load NATS URL from environment variable
	natsURL := os.Getenv("NATS_URL")

	return &Config{
		DB:  db,
		DSN: dsn,
		NATS: struct {
			URL string `yaml:"url"`
		}{
			URL: natsURL,
		},
	}
}
